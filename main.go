package main

import (
	"context"
	"fmt"
	"log"

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

	mgmClient := management.NewClient(ctx, client)

	err = mgmClient.DeleteRuleById(ctx, "2a143899b1695d627820aa4e73dbe29c22582cce5929dd131b4654bbe2a3e99a")
	if err != nil {
		log.Fatal(err)
	}
}
