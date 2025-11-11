package main

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/machadovilaca/alerts-ui-management/pkg/k8s"
	"github.com/machadovilaca/alerts-ui-management/pkg/management"
)

const namespace = "default"

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

	fmt.Println("Successfully connected to Kubernetes cluster!\n\n---")

	mgmClient := management.New(ctx, client)

	d, err := mgmClient.GetAlerts(ctx, k8s.GetAlertsRequest{})
	if err != nil {
		log.Fatalf("Failed to get alerts: %v", err)
	}

	fmt.Printf("Found %d alerts in the cluster:\n", len(d))
	for _, alert := range d {
		fmt.Printf("Alert: %s, Severity: %s, State: %s, ActiveAt: %s\n", alert.Labels["alertname"], alert.Labels["severity"], alert.State, alert.ActiveAt)
	}

	fmt.Printf("\n\n---\nWatching for alert rules in '%s' namespace every 5 seconds...\n\n", namespace)

	for {
		rules, err := mgmClient.ListRules(
			ctx,
			management.PrometheusRuleOptions{Namespace: namespace},
			management.AlertRuleOptions{},
		)
		if err != nil {
			log.Fatalf("Failed to list alert rules: %v", err)
		}

		fmt.Printf("Found %d alert rules in '%s' namespace:\n", len(rules), namespace)
		for _, rule := range rules {
			fmt.Printf("- %s: %s\n", rule.Alert, rule.Labels["severity"])
		}

		time.Sleep(5 * time.Second)
	}
}
