package management

import (
	"context"
	"errors"
	"fmt"

	monitoringv1 "github.com/prometheus-operator/prometheus-operator/pkg/apis/monitoring/v1"
)

func (c *client) ListRules(ctx context.Context, options Options) ([]monitoringv1.Rule, error) {
	if options.PrometheusRuleName != "" && options.PrometheusRuleNamespace == "" {
		return nil, errors.New("PrometheusRule Namespace must be specified when Name is provided")
	}

	// Name and Namespace specified
	if options.PrometheusRuleName != "" && options.PrometheusRuleNamespace != "" {
		pr, err := c.k8sClient.PrometheusRules().Get(ctx, options.PrometheusRuleNamespace, options.PrometheusRuleName)
		if err != nil {
			return nil, fmt.Errorf("failed to get PrometheusRule %s/%s: %w", options.PrometheusRuleNamespace, options.PrometheusRuleName, err)
		}
		return c.extractRulesFromPrometheusRule(*pr, &options), nil
	}

	// Name not specified
	allPrometheusRules, err := c.k8sClient.PrometheusRules().List(ctx, options.PrometheusRuleNamespace)
	if err != nil {
		return nil, fmt.Errorf("failed to list PrometheusRules: %w", err)
	}

	var allRules []monitoringv1.Rule
	for _, pr := range allPrometheusRules {
		rules := c.extractRulesFromPrometheusRule(pr, &options)
		allRules = append(allRules, rules...)
	}

	return allRules, nil
}

func (c *client) extractRulesFromPrometheusRule(pr monitoringv1.PrometheusRule, options *Options) []monitoringv1.Rule {
	// Group name not specified
	if options.GroupName == "" {
		var allRules []monitoringv1.Rule
		for _, group := range pr.Spec.Groups {
			allRules = append(allRules, group.Rules...)
		}
		return allRules
	}

	// Group name specified
	for _, group := range pr.Spec.Groups {
		if group.Name == options.GroupName {
			return group.Rules
		}
	}

	return []monitoringv1.Rule{}
}
