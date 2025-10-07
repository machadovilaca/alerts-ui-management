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

var _ Client = (*client)(nil)

type client struct {
	clientset             *kubernetes.Clientset
	monitoringv1clientset *monitoringv1client.Clientset
	config                *rest.Config

	prometheusRuleManager  PrometheusRuleInterface
	prometheusRuleInformer PrometheusRuleInformerInterface
}

func newClient(_ context.Context, opts ClientOptions) (Client, error) {
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
	c.prometheusRuleInformer = newPrometheusRuleInformer(monitoringv1clientset)

	return c, nil
}

func (c *client) TestConnection(_ context.Context) error {
	_, err := c.clientset.Discovery().ServerVersion()
	if err != nil {
		return fmt.Errorf("failed to connect to cluster: %w", err)
	}
	return nil
}

func (c *client) PrometheusRules() PrometheusRuleInterface {
	return c.prometheusRuleManager
}

func (c *client) PrometheusRuleInformer() PrometheusRuleInformerInterface {
	return c.prometheusRuleInformer
}
