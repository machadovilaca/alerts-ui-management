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

	namespaces, err := client.Namespaces().GetNamespaces(ctx)
	if err != nil {
		log.Fatalf("Failed to list namespaces: %v", err)
	}

	fmt.Printf("Found %d namespaces:\n", len(namespaces))
	for _, ns := range namespaces {
		fmt.Printf("  - %s\n", ns)
	}
}
