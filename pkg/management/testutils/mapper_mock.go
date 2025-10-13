package testutils

import (
	"context"

	monitoringv1 "github.com/prometheus-operator/prometheus-operator/pkg/apis/monitoring/v1"

	"github.com/machadovilaca/alerts-ui-management/pkg/management/mapper"
)

// MockMapperClient is a simple mock for the mapper.Client interface
type MockMapperClient struct {
	GetAlertingRuleIdFunc    func(alertRule *monitoringv1.Rule) mapper.PrometheusAlertRuleId
	FindAlertRuleByIdFunc    func(alertRuleId mapper.PrometheusAlertRuleId) (mapper.PrometheusRuleId, error)
	WatchPrometheusRulesFunc func(ctx context.Context)
	AddPrometheusRuleFunc    func(pr *monitoringv1.PrometheusRule)
	DeletePrometheusRuleFunc func(pr *monitoringv1.PrometheusRule)
}

func (m *MockMapperClient) GetAlertingRuleId(alertRule *monitoringv1.Rule) mapper.PrometheusAlertRuleId {
	if m.GetAlertingRuleIdFunc != nil {
		return m.GetAlertingRuleIdFunc(alertRule)
	}
	return mapper.PrometheusAlertRuleId("mock-id")
}

func (m *MockMapperClient) FindAlertRuleById(alertRuleId mapper.PrometheusAlertRuleId) (mapper.PrometheusRuleId, error) {
	if m.FindAlertRuleByIdFunc != nil {
		return m.FindAlertRuleByIdFunc(alertRuleId)
	}
	return mapper.PrometheusRuleId{}, nil
}

func (m *MockMapperClient) WatchPrometheusRules(ctx context.Context) {
	if m.WatchPrometheusRulesFunc != nil {
		m.WatchPrometheusRulesFunc(ctx)
	}
}

func (m *MockMapperClient) AddPrometheusRule(pr *monitoringv1.PrometheusRule) {
	if m.AddPrometheusRuleFunc != nil {
		m.AddPrometheusRuleFunc(pr)
	}
}

func (m *MockMapperClient) DeletePrometheusRule(pr *monitoringv1.PrometheusRule) {
	if m.DeletePrometheusRuleFunc != nil {
		m.DeletePrometheusRuleFunc(pr)
	}
}
