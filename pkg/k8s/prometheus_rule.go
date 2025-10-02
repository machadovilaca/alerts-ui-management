package k8s

import (
	"context"
	"fmt"

	monitoringv1 "github.com/prometheus-operator/prometheus-operator/pkg/apis/monitoring/v1"
	monitoringv1client "github.com/prometheus-operator/prometheus-operator/pkg/client/versioned"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// PrometheusRuleInterface defines operations for managing PrometheusRules
type PrometheusRuleInterface interface {
	List(ctx context.Context) ([]monitoringv1.PrometheusRule, error)
}

// prometheusRuleManager implements PrometheusRuleInterface
type prometheusRuleManager struct {
	clientset *monitoringv1client.Clientset
}

// newPrometheusRuleManager creates a new PrometheusRule manager
func newPrometheusRuleManagerManager(clientset *monitoringv1client.Clientset) PrometheusRuleInterface {
	return &prometheusRuleManager{
		clientset: clientset,
	}
}

// List returns a list of all PrometheusRules in the cluster
func (prm *prometheusRuleManager) List(ctx context.Context) ([]monitoringv1.PrometheusRule, error) {
	prs, err := prm.clientset.MonitoringV1().PrometheusRules("").List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to list namespaces: %w", err)
	}

	return prs.Items, nil
}
