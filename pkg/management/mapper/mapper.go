package mapper

import (
	"context"
	"crypto/sha256"
	"fmt"
	"log"
	"slices"
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
	alertRelabelConfigs map[AlertRelabelConfigId][]*AlertRelabelConfigSpec
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

	return PrometheusAlertRuleId(fmt.Sprintf("%s/%x", name, hash))
}

func (m *mapper) FindAlertRuleById(alertRuleId PrometheusAlertRuleId) (*PrometheusRuleId, *AlertRelabelConfigId, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	var (
		prId  *PrometheusRuleId
		arcId *AlertRelabelConfigId
	)

	for id, rules := range m.prometheusRules {
		if slices.Contains(rules, alertRuleId) {
			prId = &id
			break
		}
	}

	// If the PrometheusRuleId is not found, return an error
	if prId == nil {
		return nil, nil, fmt.Errorf("alert rule with id %s not found", alertRuleId)
	}

	alertname := strings.SplitN(string(alertRuleId), "/", 2)[0]

	for id, configs := range m.alertRelabelConfigs {
		for _, config := range configs {
			if config.Labels["alertname"] == alertname {
				arcId = &id
				break
			}
		}
	}

	return prId, arcId, nil
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
	go func() {
		callbacks := k8s.AlertRelabelConfigInformerCallback{
			OnAdd: func(arc *osmv1.AlertRelabelConfig) {
				m.AddAlertRelabelConfig(arc)
			},
			OnUpdate: func(arc *osmv1.AlertRelabelConfig) {
				m.AddAlertRelabelConfig(arc)
			},
			OnDelete: func(arc *osmv1.AlertRelabelConfig) {
				m.DeleteAlertRelabelConfig(arc)
			},
		}

		err := m.k8sClient.AlertRelabelConfigInformer().Run(ctx, callbacks)
		if err != nil {
			log.Fatalf("Failed to run AlertRelabelConfig informer: %v", err)
		}
	}()
}

func (m *mapper) AddAlertRelabelConfig(arc *osmv1.AlertRelabelConfig) {
	m.mu.Lock()
	defer m.mu.Unlock()

	arcId := AlertRelabelConfigId(types.NamespacedName{Namespace: arc.Namespace, Name: arc.Name})
	delete(m.alertRelabelConfigs, arcId)

	rules := make([]*AlertRelabelConfigSpec, 0)
	for _, config := range arc.Spec.Configs {
		if slices.Contains(config.SourceLabels, "alertname") {
			arcSpec := parseAlertRelabelConfigSpec(config)
			if arcSpec != nil {
				rules = append(rules, arcSpec)
			}
		}
	}

	m.alertRelabelConfigs[arcId] = rules
}

func parseAlertRelabelConfigSpec(config osmv1.RelabelConfig) *AlertRelabelConfigSpec {
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

	arcSpec := &AlertRelabelConfigSpec{
		Config: config,
		Labels: make(map[string]string),
	}

	for i, labelName := range config.SourceLabels {
		arcSpec.Labels[string(labelName)] = values[i]
	}

	return arcSpec
}

func (m *mapper) DeleteAlertRelabelConfig(arc *osmv1.AlertRelabelConfig) {
	m.mu.Lock()
	defer m.mu.Unlock()

	delete(m.alertRelabelConfigs, AlertRelabelConfigId(types.NamespacedName{Namespace: arc.Namespace, Name: arc.Name}))
}

func (m *mapper) GetAlertRelabelConfigSpec(arcId AlertRelabelConfigId) []AlertRelabelConfigSpec {
	m.mu.RLock()
	defer m.mu.RUnlock()

	configs, exists := m.alertRelabelConfigs[arcId]
	if !exists {
		return nil
	}

	result := make([]AlertRelabelConfigSpec, len(configs))
	for i, config := range configs {
		result[i] = *config
	}

	return result
}
