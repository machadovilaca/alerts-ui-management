package management

import (
	"context"
	"errors"
	"fmt"

	monitoringv1 "github.com/prometheus-operator/prometheus-operator/pkg/apis/monitoring/v1"
	"k8s.io/apimachinery/pkg/types"
)

func (c *client) ListRules(ctx context.Context, prOptions PrometheusRuleOptions, arOptions AlertRuleOptions) ([]monitoringv1.Rule, error) {
	if prOptions.Name != "" && prOptions.Namespace == "" {
		return nil, errors.New("PrometheusRule Namespace must be specified when Name is provided")
	}

	// Name and Namespace specified
	if prOptions.Name != "" && prOptions.Namespace != "" {
		pr, err := c.k8sClient.PrometheusRules().Get(ctx, prOptions.Namespace, prOptions.Name)
		if err != nil {
			return nil, fmt.Errorf("failed to get PrometheusRule %s/%s: %w", prOptions.Namespace, prOptions.Name, err)
		}
		return c.extractAndFilterRules(*pr, &prOptions, &arOptions), nil
	}

	// Name not specified
	allPrometheusRules, err := c.k8sClient.PrometheusRules().List(ctx, prOptions.Namespace)
	if err != nil {
		return nil, fmt.Errorf("failed to list PrometheusRules: %w", err)
	}

	var allRules []monitoringv1.Rule
	for _, pr := range allPrometheusRules {
		rules := c.extractAndFilterRules(pr, &prOptions, &arOptions)
		allRules = append(allRules, rules...)
	}

	return allRules, nil
}

func (c *client) extractAndFilterRules(pr monitoringv1.PrometheusRule, prOptions *PrometheusRuleOptions, arOptions *AlertRuleOptions) []monitoringv1.Rule {
	var rules []monitoringv1.Rule

	for _, group := range pr.Spec.Groups {
		// Filter by group name if specified
		if prOptions.GroupName != "" && group.Name != prOptions.GroupName {
			continue
		}

		for _, rule := range group.Rules {
			// Skip recording rules (only process alert rules)
			if rule.Alert == "" {
				continue
			}

			// Apply alert rule filters
			if !c.matchesAlertRuleFilters(rule, pr, arOptions) {
				continue
			}

			rules = append(rules, rule)
		}
	}

	return rules
}

func (c *client) matchesAlertRuleFilters(rule monitoringv1.Rule, pr monitoringv1.PrometheusRule, arOptions *AlertRuleOptions) bool {
	// Filter by alert name
	if arOptions.Name != "" && string(rule.Alert) != arOptions.Name {
		return false
	}

	// Filter by source (platform or user-defined)
	if arOptions.Source != "" {
		prId := types.NamespacedName{Name: pr.Name, Namespace: pr.Namespace}
		isPlatform := IsPlatformAlertRule(prId)

		if arOptions.Source == "platform" && !isPlatform {
			return false
		}
		if arOptions.Source == "user-defined" && isPlatform {
			return false
		}
	}

	// Filter by labels
	if len(arOptions.Labels) > 0 {
		for key, value := range arOptions.Labels {
			ruleValue, exists := rule.Labels[key]
			if !exists || ruleValue != value {
				return false
			}
		}
	}

	return true
}
