package mapper

import (
	"context"
	"crypto/sha256"
	"fmt"
	"log"
	"sort"
	"strings"
	"sync"

	osmv1 "github.com/openshift/api/monitoring/v1"
	monitoringv1 "github.com/prometheus-operator/prometheus-operator/pkg/apis/monitoring/v1"
	"k8s.io/apimachinery/pkg/types"

	"github.com/machadovilaca/alerts-ui-management/pkg/k8s"
)

type mapper struct {
	k8sClient k8s.Client
	mu        sync.RWMutex

	prometheusRules     map[PrometheusRuleId][]PrometheusAlertRuleId
	alertRelabelConfigs map[AlertRelabelConfigId][]PrometheusAlertRuleLabels
}

var _ Client = (*mapper)(nil)

func (m *mapper) GetAlertingRuleId(alertRule *monitoringv1.Rule) PrometheusAlertRuleId {
	var kind, name string
	if alertRule.Alert != "" {
		kind = "alert"
		name = alertRule.Alert
	} else if alertRule.Record != "" {
		kind = "record"
		name = alertRule.Record
	} else {
		return ""
	}

	expr := alertRule.Expr.String()
	forDuration := ""
	if alertRule.For != nil {
		forDuration = string(*alertRule.For)
	}

	var sortedLabels []string
	if alertRule.Labels != nil {
		for key, value := range alertRule.Labels {
			sortedLabels = append(sortedLabels, fmt.Sprintf("%s=%s", key, value))
		}
		sort.Strings(sortedLabels)
	}

	var sortedAnnotations []string
	if alertRule.Annotations != nil {
		for key, value := range alertRule.Annotations {
			sortedAnnotations = append(sortedAnnotations, fmt.Sprintf("%s=%s", key, value))
		}
		sort.Strings(sortedAnnotations)
	}

	// Build the hash input string
	hashInput := strings.Join([]string{
		kind,
		name,
		expr,
		forDuration,
		strings.Join(sortedLabels, ","),
		strings.Join(sortedAnnotations, ","),
	}, "\n")

	// Generate SHA256 hash
	hash := sha256.Sum256([]byte(hashInput))

	return PrometheusAlertRuleId(fmt.Sprintf("%x", hash))
}

func (m *mapper) FindAlertRuleById(alertRuleId PrometheusAlertRuleId) (*PrometheusRuleId, *AlertRelabelConfigId, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	for promRuleId, rules := range m.prometheusRules {
		for _, ruleId := range rules {
			if ruleId == alertRuleId {
				return &promRuleId, nil, nil
			}
		}
	}

	return nil, nil, fmt.Errorf("alert rule with id %s not found", alertRuleId)
}

func (m *mapper) WatchPrometheusRules(ctx context.Context) {
	go func() {
		callbacks := k8s.PrometheusRuleInformerCallback{
			OnAdd: func(pr *monitoringv1.PrometheusRule) {
				m.AddPrometheusRule(pr)
			},
			OnUpdate: func(pr *monitoringv1.PrometheusRule) {
				m.AddPrometheusRule(pr)
			},
			OnDelete: func(pr *monitoringv1.PrometheusRule) {
				m.DeletePrometheusRule(pr)
			},
		}

		err := m.k8sClient.PrometheusRuleInformer().Run(ctx, callbacks)
		if err != nil {
			log.Fatalf("Failed to run PrometheusRule informer: %v", err)
		}
	}()
}

func (m *mapper) AddPrometheusRule(pr *monitoringv1.PrometheusRule) {
	m.mu.Lock()
	defer m.mu.Unlock()

	promRuleId := PrometheusRuleId(types.NamespacedName{Namespace: pr.Namespace, Name: pr.Name})
	delete(m.prometheusRules, promRuleId)

	rules := make([]PrometheusAlertRuleId, 0)
	for _, group := range pr.Spec.Groups {
		for _, rule := range group.Rules {
			if rule.Alert != "" {
				ruleId := m.GetAlertingRuleId(&rule)
				if ruleId != "" {
					rules = append(rules, ruleId)
				}
			}
		}
	}

	m.prometheusRules[promRuleId] = rules
}

func (m *mapper) DeletePrometheusRule(pr *monitoringv1.PrometheusRule) {
	m.mu.Lock()
	defer m.mu.Unlock()

	delete(m.prometheusRules, PrometheusRuleId(types.NamespacedName{Namespace: pr.Namespace, Name: pr.Name}))
}

func (m *mapper) WatchAlertRelabelConfigs(ctx context.Context) {
}

func (m *mapper) AddAlertRelabelConfig(arc *osmv1.AlertRelabelConfig) {
	m.mu.Lock()
	defer m.mu.Unlock()

	arcId := AlertRelabelConfigId(types.NamespacedName{Namespace: arc.Namespace, Name: arc.Name})
	delete(m.alertRelabelConfigs, arcId)

	rules := make([]PrometheusAlertRuleLabels, 0)
	for _, config := range arc.Spec.Configs {
		if hasAlertnameLabel(config.SourceLabels) {
			labels := parseRelabelConfigToLabels(config)
			if labels != nil {
				rules = append(rules, labels)
			}
		}
	}

	m.alertRelabelConfigs[arcId] = rules
}

func hasAlertnameLabel(sourceLabels []osmv1.LabelName) bool {
	for _, label := range sourceLabels {
		if label == "alertname" {
			return true
		}
	}

	return false
}

func parseRelabelConfigToLabels(config osmv1.RelabelConfig) PrometheusAlertRuleLabels {
	separator := config.Separator
	if separator == "" {
		separator = ";"
	}

	regex := config.Regex
	if regex == "" {
		return nil
	}

	values := strings.Split(regex, separator)
	if len(values) != len(config.SourceLabels) {
		return nil
	}

	labels := make(PrometheusAlertRuleLabels)
	for i, labelName := range config.SourceLabels {
		labels[string(labelName)] = values[i]
	}

	return labels
}

func (m *mapper) DeleteAlertRelabelConfig(arc *osmv1.AlertRelabelConfig) {
	m.mu.Lock()
	defer m.mu.Unlock()

	delete(m.alertRelabelConfigs, AlertRelabelConfigId(types.NamespacedName{Namespace: arc.Namespace, Name: arc.Name}))
}
