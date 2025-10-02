package k8s

import (
	"context"
)

// Interface defines the contract for Kubernetes client operations
type Interface interface {
	// TestConnection tests the connection to the Kubernetes cluster
	TestConnection(ctx context.Context) error

	// PrometheusRules returns the PrometheusRule interface
	PrometheusRules() PrometheusRuleInterface
}

// Ensure Client implements Interface
var _ Interface = (*Client)(nil)
