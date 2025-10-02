package k8s

import (
	"context"
	"fmt"
	"path/filepath"

	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/util/homedir"
)

// Client wraps the Kubernetes clientset and provides methods for cluster interaction
type Client struct {
	clientset        *kubernetes.Clientset
	config           *rest.Config
	namespaceManager NamespaceInterface
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

	client := &Client{
		clientset: clientset,
		config:    config,
	}

	client.namespaceManager = newNamespaceManager(clientset)

	return client, nil
}

// TestConnection tests the connection to the Kubernetes cluster
func (c *Client) TestConnection(ctx context.Context) error {
	_, err := c.clientset.Discovery().ServerVersion()
	if err != nil {
		return fmt.Errorf("failed to connect to cluster: %w", err)
	}
	return nil
}

// Namespaces returns the namespace interface
func (c *Client) Namespaces() NamespaceInterface {
	return c.namespaceManager
}
