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

	fmt.Println("Successfully connected to Kubernetes cluster!")

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

		fmt.Printf("Found %d alert rules:\n", len(rules))
		for _, rule := range rules {
			fmt.Printf("- %s: %s\n", rule.Alert, rule.Labels["severity"])
		}

		time.Sleep(5 * time.Second)
	}
}
