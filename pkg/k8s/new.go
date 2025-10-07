package k8s

import "context"

// NewClient creates a new Kubernetes client with the given options
func NewClient(ctx context.Context, opts ClientOptions) (Client, error) {
	return newClient(ctx, opts)
}
