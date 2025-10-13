package management

import (
	"context"
	"fmt"

	monitoringv1 "github.com/prometheus-operator/prometheus-operator/pkg/apis/monitoring/v1"

	"github.com/machadovilaca/alerts-ui-management/pkg/management/mapper"
)

func (c *client) GetRuleById(ctx context.Context, alertRuleId string) (*monitoringv1.Rule, error) {
	prId, err := c.mapper.FindAlertRuleById(mapper.PrometheusAlertRuleId(alertRuleId))
	if err != nil {
		return nil, err
	}

	pr, err := c.k8sClient.PrometheusRules().Get(ctx, prId.Namespace, prId.Name)
	if err != nil {
		return nil, err
	}

	for groupIdx := range pr.Spec.Groups {
		for ruleIdx := range pr.Spec.Groups[groupIdx].Rules {
			rule := &pr.Spec.Groups[groupIdx].Rules[ruleIdx]
			if c.mapper.GetAlertingRuleId(rule) == mapper.PrometheusAlertRuleId(alertRuleId) {
				return rule, nil
			}
		}
	}

	return nil, fmt.Errorf("alert rule with id %s not found in PrometheusRule %s/%s", alertRuleId, prId.Namespace, prId.Name)
}
