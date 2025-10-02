package management

import (
	"context"

	monitoringv1 "github.com/prometheus-operator/prometheus-operator/pkg/apis/monitoring/v1"
	"k8s.io/apimachinery/pkg/types"

	"github.com/machadovilaca/alerts-ui-management/pkg/k8s"
)

// MockK8sClient is a mock implementation of k8s.Client for testing
type MockK8sClient struct {
	prometheusRules MockPrometheusRuleInterface
}

func NewMockK8sClient() *MockK8sClient {
	return &MockK8sClient{
		prometheusRules: MockPrometheusRuleInterface{},
	}
}

func (m *MockK8sClient) TestConnection(_ context.Context) error {
	return nil
}

func (m *MockK8sClient) PrometheusRules() k8s.PrometheusRuleInterface {
	return &m.prometheusRules
}

// MockPrometheusRuleInterface is a mock implementation of k8s.PrometheusRuleInterface
type MockPrometheusRuleInterface struct {
	ListFunc    func(ctx context.Context) ([]monitoringv1.PrometheusRule, error)
	UpdateFunc  func(ctx context.Context, pr monitoringv1.PrometheusRule) error
	DeleteFunc  func(ctx context.Context, namespace string, name string) error
	AddRuleFunc func(ctx context.Context, namespacedName types.NamespacedName, groupName string, rule monitoringv1.Rule) error

	// Storage for tracking calls
	ListCalls    []ListCall
	UpdateCalls  []UpdateCall
	DeleteCalls  []DeleteCall
	AddRuleCalls []AddRuleCall
}

type ListCall struct {
	Ctx context.Context
}

type UpdateCall struct {
	Ctx context.Context
	Pr  monitoringv1.PrometheusRule
}

type DeleteCall struct {
	Ctx       context.Context
	Namespace string
	Name      string
}

type AddRuleCall struct {
	Ctx            context.Context
	NamespacedName types.NamespacedName
	GroupName      string
	Rule           monitoringv1.Rule
}

func (m *MockPrometheusRuleInterface) List(ctx context.Context) ([]monitoringv1.PrometheusRule, error) {
	m.ListCalls = append(m.ListCalls, ListCall{Ctx: ctx})
	if m.ListFunc != nil {
		return m.ListFunc(ctx)
	}
	return []monitoringv1.PrometheusRule{}, nil
}

func (m *MockPrometheusRuleInterface) Update(ctx context.Context, pr monitoringv1.PrometheusRule) error {
	m.UpdateCalls = append(m.UpdateCalls, UpdateCall{Ctx: ctx, Pr: pr})
	if m.UpdateFunc != nil {
		return m.UpdateFunc(ctx, pr)
	}
	return nil
}

func (m *MockPrometheusRuleInterface) Delete(ctx context.Context, namespace string, name string) error {
	m.DeleteCalls = append(m.DeleteCalls, DeleteCall{Ctx: ctx, Namespace: namespace, Name: name})
	if m.DeleteFunc != nil {
		return m.DeleteFunc(ctx, namespace, name)
	}
	return nil
}

func (m *MockPrometheusRuleInterface) AddRule(ctx context.Context, namespacedName types.NamespacedName, groupName string, rule monitoringv1.Rule) error {
	m.AddRuleCalls = append(m.AddRuleCalls, AddRuleCall{
		Ctx:            ctx,
		NamespacedName: namespacedName,
		GroupName:      groupName,
		Rule:           rule,
	})
	if m.AddRuleFunc != nil {
		return m.AddRuleFunc(ctx, namespacedName, groupName, rule)
	}
	return nil
}
