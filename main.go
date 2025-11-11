package main

import (
	"context"
	"log"
	"net/http"

	"github.com/machadovilaca/alerts-ui-management/internal/httprouter"
	"github.com/machadovilaca/alerts-ui-management/pkg/k8s"
	"github.com/machadovilaca/alerts-ui-management/pkg/management"
)

const (
	listenAddr = ":8080"
)

func main() {
	ctx := context.Background()

	client, err := k8s.NewClient(ctx, k8s.ClientOptions{})
	if err != nil {
		log.Fatalf("Failed to create Kubernetes client: %v", err)
	}

	if err := client.TestConnection(ctx); err != nil {
		log.Fatalf("Failed to connect to cluster: %v", err)
	}

	mgmClient := management.New(ctx, client)

	r := httprouter.New(mgmClient)

	log.Println("listening on", listenAddr)
	if err := http.ListenAndServe(listenAddr, r); err != nil {
		log.Fatal(err)
	}
}
