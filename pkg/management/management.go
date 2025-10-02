package management

import (
	"context"

	monitoringv1 "github.com/prometheus-operator/prometheus-operator/pkg/apis/monitoring/v1"

	"github.com/machadovilaca/alerts-ui-management/pkg/k8s"
)

type Client interface {
	GetAlertingRuleId(ctx context.Context, alertRule monitoringv1.Rule) (string, error)
	CreateUserDefinedAlertRule(ctx context.Context, alertRule monitoringv1.Rule, options Options) error
	DeleteRuleById(ctx context.Context, alertRuleId string) error
}

type client struct {
	k8sClient k8s.Client
}

func NewClient(_ context.Context, k8sClient k8s.Client) Client {
	return &client{
		k8sClient: k8sClient,
	}
}
