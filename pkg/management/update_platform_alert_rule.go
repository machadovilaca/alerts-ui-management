package management

import (
	"context"
	"errors"
	"fmt"

	osmv1 "github.com/openshift/api/monitoring/v1"
	monitoringv1 "github.com/prometheus-operator/prometheus-operator/pkg/apis/monitoring/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"

	"github.com/machadovilaca/alerts-ui-management/pkg/management/mapper"
)

func (c *client) UpdatePlatformAlertRule(ctx context.Context, alertRuleId string, alertRule monitoringv1.Rule) error {
	prId, err := c.mapper.FindAlertRuleById(mapper.PrometheusAlertRuleId(alertRuleId))
	if err != nil {
		return err
	}

	if !IsPlatformAlertRule(types.NamespacedName(*prId)) {
		return errors.New("cannot update non-platform alert rule")
	}

	originalRule, err := c.getOriginalPlatformRule(ctx, prId, alertRuleId)
	if err != nil {
		return err
	}

	labelChanges := calculateLabelChanges(originalRule.Labels, alertRule.Labels)
	if len(labelChanges) == 0 {
		return errors.New("no label changes detected; platform alert rules can only have labels updated")
	}

	return c.applyLabelChangesViaAlertRelabelConfig(ctx, prId, originalRule.Alert, labelChanges)
}

func (c *client) getOriginalPlatformRule(ctx context.Context, prId *mapper.PrometheusRuleId, alertRuleId string) (*monitoringv1.Rule, error) {
	pr, err := c.k8sClient.PrometheusRules().Get(ctx, prId.Namespace, prId.Name)
	if err != nil {
		return nil, fmt.Errorf("failed to get PrometheusRule %s/%s: %w", prId.Namespace, prId.Name, err)
	}

	for groupIdx := range pr.Spec.Groups {
		for ruleIdx := range pr.Spec.Groups[groupIdx].Rules {
			rule := &pr.Spec.Groups[groupIdx].Rules[ruleIdx]
			if c.shouldUpdateRule(*rule, alertRuleId) {
				return rule, nil
			}
		}
	}

	return nil, fmt.Errorf("alert rule with id %s not found in PrometheusRule %s/%s", alertRuleId, prId.Namespace, prId.Name)
}

type labelChange struct {
	action      string
	sourceLabel string
	targetLabel string
	value       string
}

func calculateLabelChanges(originalLabels, newLabels map[string]string) []labelChange {
	var changes []labelChange

	for key, newValue := range newLabels {
		originalValue, exists := originalLabels[key]
		if !exists || originalValue != newValue {
			changes = append(changes, labelChange{
				action:      "Replace",
				targetLabel: key,
				value:       newValue,
			})
		}
	}

	for key := range originalLabels {
		if _, exists := newLabels[key]; !exists {
			changes = append(changes, labelChange{
				action:      "LabelDrop",
				sourceLabel: key,
			})
		}
	}

	return changes
}

func (c *client) applyLabelChangesViaAlertRelabelConfig(ctx context.Context, prId *mapper.PrometheusRuleId, alertName string, changes []labelChange) error {
	// Try to get existing AlertRelabelConfig by constructing its expected name
	arcName := fmt.Sprintf("%s-%s-relabel", prId.Name, alertName)
	arc, err := c.k8sClient.AlertRelabelConfigs().Get(ctx, prId.Namespace, arcName)

	arcExists := err == nil

	if !arcExists {
		// Create new AlertRelabelConfig
		arc = &osmv1.AlertRelabelConfig{
			ObjectMeta: metav1.ObjectMeta{
				Name:      arcName,
				Namespace: prId.Namespace,
			},
			Spec: osmv1.AlertRelabelConfigSpec{
				Configs: []osmv1.RelabelConfig{},
			},
		}
	}

	arc.Spec.Configs = c.buildRelabelConfigs(alertName, changes)

	if arcExists {
		err = c.k8sClient.AlertRelabelConfigs().Update(ctx, *arc)
		if err != nil {
			return fmt.Errorf("failed to update AlertRelabelConfig %s/%s: %w", arc.Namespace, arc.Name, err)
		}
	} else {
		_, err = c.k8sClient.AlertRelabelConfigs().Create(ctx, *arc)
		if err != nil {
			return fmt.Errorf("failed to create AlertRelabelConfig %s/%s: %w", arc.Namespace, arc.Name, err)
		}
	}

	return nil
}

func (c *client) buildRelabelConfigs(alertName string, changes []labelChange) []osmv1.RelabelConfig {
	var configs []osmv1.RelabelConfig

	for _, change := range changes {
		switch change.action {
		case "Replace":
			config := osmv1.RelabelConfig{
				SourceLabels: []osmv1.LabelName{"alertname", osmv1.LabelName(change.targetLabel)},
				Regex:        fmt.Sprintf("%s;.*", alertName),
				TargetLabel:  change.targetLabel,
				Replacement:  change.value,
				Action:       "Replace",
			}
			configs = append(configs, config)
		case "LabelDrop":
			config := osmv1.RelabelConfig{
				SourceLabels: []osmv1.LabelName{"alertname"},
				Regex:        alertName,
				TargetLabel:  change.sourceLabel,
				Replacement:  "",
				Action:       "Replace",
			}
			configs = append(configs, config)
		}
	}

	return configs
}
