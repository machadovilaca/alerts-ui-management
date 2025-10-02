package main

import (
	"context"
	"fmt"
	"log"

	"github.com/machadovilaca/alerts-ui-management/pkg/k8s"
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

	prs, err := client.PrometheusRules().List(ctx)
	if err != nil {
		log.Fatalf("Failed to list PrometheusRules: %v", err)
	}

	fmt.Printf("Found %d PrometheusRules:\n", len(prs))
	for _, pr := range prs {
		fmt.Printf("  - %s/%s\n", pr.Namespace, pr.Name)
	}
}
