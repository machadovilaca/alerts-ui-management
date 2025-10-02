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

// Client wraps the Kubernetes clientset and provides methods for cluster interaction
type Client struct {
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
func NewClient(_ context.Context, opts ClientOptions) (Interface, error) {
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

	client := &Client{
		clientset:             clientset,
		monitoringv1clientset: monitoringv1clientset,
		config:                config,
	}

	client.prometheusRuleManager = newPrometheusRuleManagerManager(monitoringv1clientset)

	return client, nil
}

// TestConnection tests the connection to the Kubernetes cluster
func (c *Client) TestConnection(_ context.Context) error {
	_, err := c.clientset.Discovery().ServerVersion()
	if err != nil {
		return fmt.Errorf("failed to connect to cluster: %w", err)
	}
	return nil
}

// PrometheusRules returns the PrometheusRule interface
func (c *Client) PrometheusRules() PrometheusRuleInterface {
	return c.prometheusRuleManager
}
