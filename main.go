package main

import (
	"context"
	"fmt"
	"log"

	"github.com/machadovilaca/alerts-ui-management/pkg/k8s"
	"github.com/machadovilaca/alerts-ui-management/pkg/management"
	monitoringv1 "github.com/prometheus-operator/prometheus-operator/pkg/apis/monitoring/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
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

	mgmClient := management.NewClient(ctx, client)

	alertRule := monitoringv1.Rule{
		Alert: "HighRequestLatency",
		Expr:  intstr.FromString("job:request_latency_seconds:mean5m > 0.5"),
		Labels: map[string]string{
			"severity": "warning",
		},
		Annotations: map[string]string{
			"summary":     "High request latency",
			"description": "Request latency is above 0.5s for more than 10 minutes.",
		},
	}

	err = mgmClient.CreateUserDefinedAlertRule(ctx, alertRule, management.Options{
		PrometheusRuleName:      "custom-alert-rules",
		PrometheusRuleNamespace: "testns",
		GroupName:               "custom-group",
	})
	if err != nil {
		log.Fatal(err)
	}
}
