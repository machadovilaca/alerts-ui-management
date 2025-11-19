package management

import (
	"context"

	monitoringv1 "github.com/prometheus-operator/prometheus-operator/pkg/apis/monitoring/v1"
	"k8s.io/apimachinery/pkg/types"
)

const (
	DefaultGroupName = "user-defined-rules"
)

func (c *client) CreateUserDefinedAlertRule(ctx context.Context, alertRule monitoringv1.Rule, prOptions PrometheusRuleOptions) (string, error) {
	if prOptions.Name == "" || prOptions.Namespace == "" {
		return "", &ValidationError{Message: "PrometheusRule Name and Namespace must be specified"}
	}

	nn := types.NamespacedName{
		Name:      prOptions.Name,
		Namespace: prOptions.Namespace,
	}

	if IsPlatformAlertRule(nn) {
		return "", &NotAllowedError{Message: "cannot add user-defined alert rule to a platform-managed PrometheusRule"}
	}

	// Check if rule with the same ID already exists
	ruleId := c.mapper.GetAlertingRuleId(&alertRule)
	_, err := c.mapper.FindAlertRuleById(ruleId)
	if err == nil {
		return "", &ConflictError{Message: "alert rule with exact config already exists"}
	}

	if prOptions.GroupName == "" {
		prOptions.GroupName = DefaultGroupName
	}

	err = c.k8sClient.PrometheusRules().AddRule(ctx, nn, prOptions.GroupName, alertRule)
	if err != nil {
		return "", err
	}

	return string(c.mapper.GetAlertingRuleId(&alertRule)), nil
}
