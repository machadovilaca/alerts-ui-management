package management

import (
	"context"
	"errors"

	monitoringv1 "github.com/prometheus-operator/prometheus-operator/pkg/apis/monitoring/v1"
	"k8s.io/apimachinery/pkg/types"
)

const (
	DefaultGroupName = "user-defined-rules"
)

func (c *client) CreateUserDefinedAlertRule(ctx context.Context, alertRule monitoringv1.Rule, options Options) (string, error) {
	ruleId := c.mapper.GetAlertingRuleId(&alertRule)

	// Check if rule with the same ID already exists
	_, err := c.mapper.FindAlertRuleById(ruleId)
	if err == nil {
		return "", errors.New("alert rule with exact config already exists")
	}

	if options.PrometheusRuleName == "" || options.PrometheusRuleNamespace == "" {
		return "", errors.New("PrometheusRule Name and Namespace must be specified")
	}
	nn := types.NamespacedName{
		Name:      options.PrometheusRuleName,
		Namespace: options.PrometheusRuleNamespace,
	}

	if options.GroupName == "" {
		options.GroupName = DefaultGroupName
	}

	err = c.k8sClient.PrometheusRules().AddRule(ctx, nn, options.GroupName, alertRule)
	if err != nil {
		return "", err
	}

	return string(c.mapper.GetAlertingRuleId(&alertRule)), nil
}
