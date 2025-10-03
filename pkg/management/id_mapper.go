package management

import (
	"context"
	"crypto/sha256"
	"fmt"
	"log"
	"sort"
	"strings"
	"sync"

	"github.com/machadovilaca/alerts-ui-management/pkg/k8s"
	monitoringv1 "github.com/prometheus-operator/prometheus-operator/pkg/apis/monitoring/v1"
	"k8s.io/apimachinery/pkg/types"
)

type PrometheusRuleId types.NamespacedName
type PrometheusAlertRuleId string

type idMapper struct {
	k8sClient k8s.Client

	mu    sync.RWMutex
	files map[PrometheusRuleId][]PrometheusAlertRuleId
}

func newIdMapper(k8sClient k8s.Client) *idMapper {
	return &idMapper{
		k8sClient: k8sClient,
		files:     make(map[PrometheusRuleId][]PrometheusAlertRuleId),
	}
}

func (im *idMapper) FindAlertRuleById(alertRuleId PrometheusAlertRuleId) (PrometheusRuleId, error) {
	im.mu.RLock()
	defer im.mu.RUnlock()

	for promRuleId, rules := range im.files {
		for _, ruleId := range rules {
			if ruleId == alertRuleId {
				return promRuleId, nil
			}
		}
	}

	return PrometheusRuleId{}, fmt.Errorf("alert rule with id %s not found", alertRuleId)
}

func (im *idMapper) WatchPrometheusRules(ctx context.Context) {
	go func() {
		callbacks := k8s.PrometheusRuleInformerCallback{
			OnAdd: func(pr *monitoringv1.PrometheusRule) {
				im.addPrometheusRule(pr)
			},
			OnUpdate: func(pr *monitoringv1.PrometheusRule) {
				im.addPrometheusRule(pr)
			},
			OnDelete: func(pr *monitoringv1.PrometheusRule) {
				im.deletePrometheusRule(pr)
			},
		}

		err := im.k8sClient.PrometheusRuleInformer().Run(ctx, callbacks)
		if err != nil {
			log.Fatalf("Failed to run PrometheusRule informer: %v", err)
		}
	}()
}

func (im *idMapper) addPrometheusRule(pr *monitoringv1.PrometheusRule) {
	im.mu.Lock()
	defer im.mu.Unlock()

	promRuleId := PrometheusRuleId(types.NamespacedName{Namespace: pr.Namespace, Name: pr.Name})
	delete(im.files, promRuleId)

	rules := make([]PrometheusAlertRuleId, 0)
	for _, group := range pr.Spec.Groups {
		for _, rule := range group.Rules {
			if rule.Alert != "" {
				ruleId := getAlertingRuleId(&rule)
				if ruleId != "" {
					rules = append(rules, ruleId)
				}
			}
		}
	}

	im.files[promRuleId] = rules
}

func (im *idMapper) deletePrometheusRule(pr *monitoringv1.PrometheusRule) {
	im.mu.Lock()
	defer im.mu.Unlock()

	delete(im.files, PrometheusRuleId(types.NamespacedName{Namespace: pr.Namespace, Name: pr.Name}))
}

func getAlertingRuleId(alertRule *monitoringv1.Rule) PrometheusAlertRuleId {
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
			if !volatileAnnotationKeys[key] {
				sortedAnnotations = append(sortedAnnotations, fmt.Sprintf("%s=%s", key, value))
			}
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
