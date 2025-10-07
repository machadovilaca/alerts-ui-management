package management

import (
	"context"

	"github.com/machadovilaca/alerts-ui-management/pkg/k8s"
	"github.com/machadovilaca/alerts-ui-management/pkg/management/mapper"
)

type client struct {
	k8sClient k8s.Client
	mapper    mapper.Client
}

func new(ctx context.Context, k8sClient k8s.Client) Client {
	m := mapper.New(k8sClient)
	m.WatchPrometheusRules(ctx)

	return &client{
		k8sClient: k8sClient,
		mapper:    m,
	}
}
