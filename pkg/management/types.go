package management

import (
	"context"

	monitoringv1 "github.com/prometheus-operator/prometheus-operator/pkg/apis/monitoring/v1"
)

// Client is the interface for managing alert rules
type Client interface {
	// ListRules lists all alert rules in the specified PrometheusRule resource
	ListRules(ctx context.Context, options Options) ([]monitoringv1.Rule, error)

	// CreateUserDefinedAlertRule creates a new user-defined alert rule
	CreateUserDefinedAlertRule(ctx context.Context, alertRule monitoringv1.Rule, options Options) (alertRuleId string, err error)

	// DeleteRuleById deletes an alert rule by its ID
	DeleteRuleById(ctx context.Context, alertRuleId string) error
}

// Options for creating a user-defined alert rule
type Options struct {
	// Name and Namespace of the PrometheusRule resource where the alert rule will be added
	PrometheusRuleName string `json:"prometheusRuleName"`

	// Namespace of the PrometheusRule resource where the alert rule will be added
	PrometheusRuleNamespace string `json:"prometheusRuleNamespace"`

	// GroupName is the name of the group within the PrometheusRule resource where the alert rule will be added
	GroupName string `json:"groupName"`
}
