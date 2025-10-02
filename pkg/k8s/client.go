package k8s

import (
	"context"
	"fmt"
	"path/filepath"

	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/util/homedir"

	monitoringv1client "github.com/prometheus-operator/prometheus-operator/pkg/client/versioned"
)

// Ensure client implements Client
var _ Client = (*client)(nil)

// Client defines the contract for Kubernetes client operations
type Client interface {
	// TestConnection tests the connection to the Kubernetes cluster
	TestConnection(ctx context.Context) error

	// PrometheusRules returns the PrometheusRule interface
	PrometheusRules() PrometheusRuleInterface
}

type client struct {
	clientset             *kubernetes.Clientset
	monitoringv1clientset *monitoringv1client.Clientset
	config                *rest.Config

	prometheusRuleManager PrometheusRuleInterface
}

// ClientOptions holds configuration options for creating a Kubernetes client
type ClientOptions struct {
	// KubeconfigPath specifies the path to the kubeconfig file for remote connections
	// If empty, will try default locations or in-cluster config
	KubeconfigPath string
}

// NewClient creates a new Kubernetes client with the given options
func NewClient(_ context.Context, opts ClientOptions) (Client, error) {
	var config *rest.Config
	var err error

	if opts.KubeconfigPath == "" {
		// Try default location
		if home := homedir.HomeDir(); home != "" {
			opts.KubeconfigPath = filepath.Join(home, ".kube", "config")
		}
	}

	config, err = clientcmd.BuildConfigFromFlags("", opts.KubeconfigPath)
	if err != nil {
		config, err = rest.InClusterConfig()
		if err != nil {
			return nil, fmt.Errorf("failed to create config from kubeconfig or in-cluster: %w", err)
		}
	}

	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		return nil, fmt.Errorf("failed to create clientset: %w", err)
	}

	monitoringv1clientset, err := monitoringv1client.NewForConfig(config)
	if err != nil {
		return nil, fmt.Errorf("failed to create monitoringv1 clientset: %w", err)
	}

	c := &client{
		clientset:             clientset,
		monitoringv1clientset: monitoringv1clientset,
		config:                config,
	}

	c.prometheusRuleManager = newPrometheusRuleManagerManager(monitoringv1clientset)

	return c, nil
}

// TestConnection tests the connection to the Kubernetes cluster
func (c *client) TestConnection(_ context.Context) error {
	_, err := c.clientset.Discovery().ServerVersion()
	if err != nil {
		return fmt.Errorf("failed to connect to cluster: %w", err)
	}
	return nil
}

// PrometheusRules returns the PrometheusRule interface
func (c *client) PrometheusRules() PrometheusRuleInterface {
	return c.prometheusRuleManager
}
