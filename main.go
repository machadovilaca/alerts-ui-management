package main

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/machadovilaca/alerts-ui-management/pkg/k8s"
	"github.com/machadovilaca/alerts-ui-management/pkg/management"
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

	fmt.Println("Successfully connected to Kubernetes cluster!\n\n---")

	d, err := client.PrometheusAlerts().GetActiveAlerts(ctx)
	if err != nil {
		log.Fatalf("Failed to get active alerts: %v", err)
	}

	fmt.Printf("Found %d active alerts in the cluster:\n", len(d))
	for _, alert := range d {
		fmt.Printf("Alert: %s, Severity: %s, State: %s, ActiveAt: %s\n", alert.Name, alert.Severity, alert.State, alert.ActiveAt)
	}

	fmt.Printf("\n\n---\nWatching for alert rules in 'default' namespace every 5 seconds...\n\n")
	mgmClient := management.New(ctx, client)

	for {
		rules, err := mgmClient.ListRules(
			ctx,
			management.PrometheusRuleOptions{Namespace: "default"},
			management.AlertRuleOptions{},
		)
		if err != nil {
			log.Fatalf("Failed to list alert rules: %v", err)
		}

		fmt.Printf("Found %d alert rules in 'default' namespace:\n", len(rules))
		for _, rule := range rules {
			fmt.Printf("- %s: %s\n", rule.Alert, rule.Labels["severity"])
		}

		time.Sleep(5 * time.Second)
	}
}
