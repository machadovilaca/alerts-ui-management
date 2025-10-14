package management

import (
	"context"
	"fmt"

	monitoringv1 "github.com/prometheus-operator/prometheus-operator/pkg/apis/monitoring/v1"

	"github.com/machadovilaca/alerts-ui-management/pkg/management/mapper"
)

func (c *client) GetRuleById(ctx context.Context, alertRuleId string) (monitoringv1.Rule, error) {
	prId, arcId, err := c.mapper.FindAlertRuleById(mapper.PrometheusAlertRuleId(alertRuleId))
	if err != nil {
		return monitoringv1.Rule{}, err
	}

	pr, err := c.k8sClient.PrometheusRules().Get(ctx, prId.Namespace, prId.Name)
	if err != nil {
		return monitoringv1.Rule{}, err
	}

	var rule *monitoringv1.Rule

	for groupIdx := range pr.Spec.Groups {
		for ruleIdx := range pr.Spec.Groups[groupIdx].Rules {
			foundRule := &pr.Spec.Groups[groupIdx].Rules[ruleIdx]
			if c.mapper.GetAlertingRuleId(foundRule) == mapper.PrometheusAlertRuleId(alertRuleId) {
				rule = foundRule
				break
			}
		}
	}

	if rule != nil {
		return c.updateRuleBasedOnRelabelConfig(*rule, arcId)
	}

	return monitoringv1.Rule{}, fmt.Errorf("alert rule with id %s not found in PrometheusRule %s/%s", alertRuleId, prId.Namespace, prId.Name)
}

func (c *client) updateRuleBasedOnRelabelConfig(rule monitoringv1.Rule, arcId *mapper.AlertRelabelConfigId) (monitoringv1.Rule, error) {
	if arcId == nil {
		return rule, nil
	}

	specs := c.mapper.GetAlertRelabelConfigSpec(*arcId)

	for _, spec := range specs {
		if spec.Labels["alertname"] == rule.Alert {
			// TODO: (machadovilaca) Implement all relabeling actions
			// 'Replace', 'Keep', 'Drop', 'HashMod', 'LabelMap', 'LabelDrop', or 'LabelKeep'

			switch spec.Config.Action {
			case "Drop":
				return monitoringv1.Rule{}, fmt.Errorf("alert rule with id %s has been dropped by relabeling configuration", *arcId)
			case "Replace":
				return handleReplaceAction(rule, spec)
			case "Keep":
				// Keep action is a no-op in this context since the rule is already matched
			case "HashMod":
				// HashMod action is not implemented yet
			case "LabelMap":
				// LabelMap action is not implemented yet
			case "LabelDrop":
				// LabelDrop action is not implemented yet
			case "LabelKeep":
				// LabelKeep action is not implemented yet
			default:
				// Unsupported action, ignore
			}
		}
	}

	return rule, nil
}

func handleReplaceAction(rule monitoringv1.Rule, spec mapper.AlertRelabelConfigSpec) (monitoringv1.Rule, error) {
	if spec.Config.TargetLabel == "severity" {
		rule.Labels["severity"] = spec.Config.Replacement
	}

	return rule, nil
}
