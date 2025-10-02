package management_test

import (
	"context"

	monitoringv1 "github.com/prometheus-operator/prometheus-operator/pkg/apis/monitoring/v1"
	"k8s.io/apimachinery/pkg/types"

	"github.com/machadovilaca/alerts-ui-management/pkg/k8s"
)

type mockK8sClient struct {
	prometheusRules mockPrometheusRules
	testConnection  func(ctx context.Context) error
}

func (m *mockK8sClient) TestConnection(ctx context.Context) error {
	if m.testConnection != nil {
		return m.testConnection(ctx)
	}
	return nil
}

func (m *mockK8sClient) PrometheusRules() k8s.PrometheusRuleInterface {
	return &m.prometheusRules
}

type mockPrometheusRules struct {
	listFunc    func(ctx context.Context) ([]monitoringv1.PrometheusRule, error)
	addRuleFunc func(ctx context.Context, namespacedName types.NamespacedName, groupName string, rule monitoringv1.Rule) error
}

func (m *mockPrometheusRules) List(ctx context.Context) ([]monitoringv1.PrometheusRule, error) {
	if m.listFunc != nil {
		return m.listFunc(ctx)
	}
	return []monitoringv1.PrometheusRule{}, nil
}

func (m *mockPrometheusRules) AddRule(ctx context.Context, namespacedName types.NamespacedName, groupName string, rule monitoringv1.Rule) error {
	if m.addRuleFunc != nil {
		return m.addRuleFunc(ctx, namespacedName, groupName, rule)
	}
	return nil
}
