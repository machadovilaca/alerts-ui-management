package management

import (
	"context"

	"github.com/machadovilaca/alerts-ui-management/pkg/k8s"
)

// New creates a new management client
func New(ctx context.Context, k8sClient k8s.Client) Client {
	return new(ctx, k8sClient)
}
