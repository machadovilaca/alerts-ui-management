package main

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/machadovilaca/alerts-ui-management/pkg/k8s"
	monitoringv1 "github.com/prometheus-operator/prometheus-operator/pkg/apis/monitoring/v1"
)

func main() {
	ctx := context.Background()

	// Create a new Kubernetes client (tries kubeconfig first, then in-cluster)
	client, err := k8s.NewClient(ctx, k8s.ClientOptions{})
	if err != nil {
		log.Fatalf("Failed to create Kubernetes client: %v", err)
	}

	if err := client.TestConnection(ctx); err != nil {
		log.Fatalf("Failed to connect to cluster: %v", err)
	}

	fmt.Println("Successfully connected to Kubernetes cluster!")

	callbacks := k8s.PrometheusRuleInformerCallback{
		OnAdd: func(pr *monitoringv1.PrometheusRule) {
			fmt.Printf("PrometheusRule added: %s/%s\n", pr.Namespace, pr.Name)
		},
		OnUpdate: func(pr *monitoringv1.PrometheusRule) {
			fmt.Printf("PrometheusRule updated: %s/%s\n", pr.Namespace, pr.Name)
		},
		OnDelete: func(pr *monitoringv1.PrometheusRule) {
			fmt.Printf("PrometheusRule deleted: %s/%s\n", pr.Namespace, pr.Name)
		},
	}

	go func() {
		err = client.PrometheusRuleInformer().Run(ctx, callbacks)
		if err != nil {
			log.Fatalf("Failed to run PrometheusRule informer: %v", err)
		}
	}()

	time.Sleep(10 * time.Second)
}
