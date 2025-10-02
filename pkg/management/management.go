package management

import (
	"context"

	"github.com/machadovilaca/alerts-ui-management/pkg/k8s"
	monitoringv1 "github.com/prometheus-operator/prometheus-operator/pkg/apis/monitoring/v1"
)

type Client interface {
	CreateUserDefinedAlertRule(ctx context.Context, alertRule monitoringv1.Rule, options Options) error
}

type client struct {
	k8sClient k8s.Client
}

func NewClient(_ context.Context, k8sClient k8s.Client) Client {
	return &client{
		k8sClient: k8sClient,
	}
}
