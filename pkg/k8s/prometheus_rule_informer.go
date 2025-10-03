package k8s

import (
	"context"
	"log"

	monitoringv1 "github.com/prometheus-operator/prometheus-operator/pkg/apis/monitoring/v1"
	monitoringv1client "github.com/prometheus-operator/prometheus-operator/pkg/client/versioned"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/watch"
)

// PrometheusRuleInformerInterface defines operations for PrometheusRules informers
type PrometheusRuleInformerInterface interface {
	Run(ctx context.Context, callbacks PrometheusRuleInformerCallback) error
}

type PrometheusRuleInformerCallback struct {
	OnAdd    func(pr *monitoringv1.PrometheusRule)
	OnUpdate func(pr *monitoringv1.PrometheusRule)
	OnDelete func(pr *monitoringv1.PrometheusRule)
}

type prometheusRuleInformer struct {
	clientset *monitoringv1client.Clientset
}

func newPrometheusRuleInformer(clientset *monitoringv1client.Clientset) PrometheusRuleInformerInterface {
	return &prometheusRuleInformer{
		clientset: clientset,
	}
}

// Run sets up a watcher for PrometheusRule resources and handles events
func (pri *prometheusRuleInformer) Run(ctx context.Context, callbacks PrometheusRuleInformerCallback) error {
	options := metav1.ListOptions{
		Watch: true,
	}

	watcher, err := pri.clientset.MonitoringV1().PrometheusRules("").Watch(ctx, options)
	if err != nil {
		return err
	}

	ch := watcher.ResultChan()
	for event := range ch {
		pod, ok := event.Object.(*monitoringv1.PrometheusRule)
		if !ok {
			log.Printf("Unexpected type: %v", event.Object)
			continue
		}

		switch event.Type {
		case watch.Added:
			if callbacks.OnAdd != nil {
				callbacks.OnAdd(pod)
			}
		case watch.Modified:
			if callbacks.OnUpdate != nil {
				callbacks.OnUpdate(pod)
			}
		case watch.Deleted:
			if callbacks.OnDelete != nil {
				callbacks.OnDelete(pod)
			}
		case watch.Error:
			log.Printf("Error occurred while watching PrometheusRule: %s\n", event.Object)
		}
	}

	log.Fatalf("PrometheusRule watcher channel closed unexpectedly")
	return nil
}
