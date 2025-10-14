package mapper

import (
	"context"

	osmv1 "github.com/openshift/api/monitoring/v1"
	monitoringv1 "github.com/prometheus-operator/prometheus-operator/pkg/apis/monitoring/v1"
	"k8s.io/apimachinery/pkg/types"
)

// PrometheusRuleId is a unique identifier for a PrometheusRule resource in Kubernetes, represented by its NamespacedName.
type PrometheusRuleId types.NamespacedName

// AlertRelabelConfigId is a unique identifier for an AlertRelabelConfig resource in Kubernetes, represented by its NamespacedName.
type AlertRelabelConfigId types.NamespacedName

// PrometheusAlertRuleId is a hash-based identifier for an alerting rule within a PrometheusRule, represented by a string.
type PrometheusAlertRuleId string

// AlertRelabelConfigSpec represents an AlertRelabelConfig group with its relabeling configuration and associated labels.
type AlertRelabelConfigSpec struct {
	Config osmv1.RelabelConfig `json:"config,omitempty"`
	Labels map[string]string   `json:"labels,omitempty"`
}

// Client defines the interface for mapping between Prometheus alerting rules and their unique identifiers.
type Client interface {
	// GetAlertingRuleId returns the unique identifier for a given alerting rule.
	GetAlertingRuleId(alertRule *monitoringv1.Rule) PrometheusAlertRuleId

	// FindAlertRuleById returns the PrometheusRuleId for a given alerting rule ID.
	FindAlertRuleById(alertRuleId PrometheusAlertRuleId) (*PrometheusRuleId, *AlertRelabelConfigId, error)

	// WatchPrometheusRules starts watching for changes to PrometheusRules.
	WatchPrometheusRules(ctx context.Context)

	// AddPrometheusRule adds or updates a PrometheusRule in the mapper.
	AddPrometheusRule(pr *monitoringv1.PrometheusRule)

	// DeletePrometheusRule removes a PrometheusRule from the mapper.
	DeletePrometheusRule(pr *monitoringv1.PrometheusRule)

	// WatchAlertRelabelConfigs starts watching for changes to AlertRelabelConfigs.
	WatchAlertRelabelConfigs(ctx context.Context)

	// AddAlertRelabelConfig adds or updates an AlertRelabelConfig in the mapper.
	AddAlertRelabelConfig(arc *osmv1.AlertRelabelConfig)

	// DeleteAlertRelabelConfig removes an AlertRelabelConfig from the mapper.
	DeleteAlertRelabelConfig(arc *osmv1.AlertRelabelConfig)

	// GetAlertRelabelConfigSpec returns the AlertRelabelConfigSpec associated with a given AlertRelabelConfigId.
	GetAlertRelabelConfigSpec(arcId AlertRelabelConfigId) []AlertRelabelConfigSpec
}
