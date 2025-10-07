package testutils

import (
	"context"

	"github.com/machadovilaca/alerts-ui-management/pkg/management/mapper"
	monitoringv1 "github.com/prometheus-operator/prometheus-operator/pkg/apis/monitoring/v1"
)

// MockMapperClient is a simple mock for the mapper.Client interface
type MockMapperClient struct{}

func (m *MockMapperClient) GetAlertingRuleId(alertRule *monitoringv1.Rule) mapper.PrometheusAlertRuleId {
	return mapper.PrometheusAlertRuleId("mock-id")
}

func (m *MockMapperClient) FindAlertRuleById(alertRuleId mapper.PrometheusAlertRuleId) (mapper.PrometheusRuleId, error) {
	return mapper.PrometheusRuleId{}, nil
}

func (m *MockMapperClient) WatchPrometheusRules(ctx context.Context) {}

func (m *MockMapperClient) AddPrometheusRule(pr *monitoringv1.PrometheusRule) {}

func (m *MockMapperClient) DeletePrometheusRule(pr *monitoringv1.PrometheusRule) {}
