package management

import (
	"context"
	"fmt"

	monitoringv1 "github.com/prometheus-operator/prometheus-operator/pkg/apis/monitoring/v1"
	"k8s.io/apimachinery/pkg/types"
)

const (
	AlertRuleIdLabelKey = "auim_id"
	DefaultGroupName    = "user-defined-rules"
)

var volatileAnnotationKeys = map[string]bool{
	AlertRuleIdLabelKey: true,
}

func (c *client) CreateUserDefinedAlertRule(ctx context.Context, alertRule monitoringv1.Rule, options Options) error {
	if alertRule.Annotations == nil {
		alertRule.Annotations = make(map[string]string)
	}
	alertRule.Annotations[AlertRuleIdLabelKey] = string(getAlertingRuleId(&alertRule))

	if options.PrometheusRuleName == "" || options.PrometheusRuleNamespace == "" {
		return fmt.Errorf("PrometheusRule Name and Namespace must be specified")
	}
	nn := types.NamespacedName{
		Name:      options.PrometheusRuleName,
		Namespace: options.PrometheusRuleNamespace,
	}

	if options.GroupName == "" {
		options.GroupName = DefaultGroupName
	}

	err := c.k8sClient.PrometheusRules().AddRule(ctx, nn, options.GroupName, alertRule)
	if err != nil {
		return err
	}

	return nil
}

func (c *client) DeleteRuleById(ctx context.Context, alertRuleId string) error {
	prId, err := c.idMapper.FindAlertRuleById(PrometheusAlertRuleId(alertRuleId))
	if err != nil {
		return err
	}

	pr, err := c.k8sClient.PrometheusRules().Get(ctx, prId.Namespace, prId.Name)
	if err != nil {
		return err
	}

	updated := false
	var newGroups []monitoringv1.RuleGroup

	for _, group := range pr.Spec.Groups {
		newRules := c.filterRulesById(group.Rules, alertRuleId, &updated)

		// Only keep groups that still have rules
		if len(newRules) > 0 {
			group.Rules = newRules
			newGroups = append(newGroups, group)
		} else if len(newRules) != len(group.Rules) {
			// Group became empty due to rule deletion
			updated = true
		}
	}

	if updated {
		if len(newGroups) == 0 {
			// No groups left, delete the entire PrometheusRule
			err = c.k8sClient.PrometheusRules().Delete(ctx, pr.Namespace, pr.Name)
			if err != nil {
				return fmt.Errorf("failed to delete PrometheusRule %s/%s: %w", pr.Namespace, pr.Name, err)
			}
		} else {
			// Update PrometheusRule with remaining groups
			pr.Spec.Groups = newGroups
			err = c.k8sClient.PrometheusRules().Update(ctx, *pr)
			if err != nil {
				return fmt.Errorf("failed to update PrometheusRule %s/%s: %w", pr.Namespace, pr.Name, err)
			}
		}
	}

	return nil
}

func (c *client) filterRulesById(rules []monitoringv1.Rule, alertRuleId string, updated *bool) []monitoringv1.Rule {
	var newRules []monitoringv1.Rule

	for _, rule := range rules {
		if c.shouldDeleteRule(rule, alertRuleId) {
			*updated = true
			continue
		}
		newRules = append(newRules, rule)
	}

	return newRules
}

func (c *client) shouldDeleteRule(rule monitoringv1.Rule, alertRuleId string) bool {
	if rule.Annotations != nil {
		id, exists := rule.Annotations[AlertRuleIdLabelKey]
		if exists && id == alertRuleId {
			return true
		}
	}

	if alertRuleId == string(getAlertingRuleId(&rule)) {
		return true
	}

	return false
}
