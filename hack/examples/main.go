package main

import (
	"context"
	"fmt"
	"log"
	"time"

	monitoringv1 "github.com/prometheus-operator/prometheus-operator/pkg/apis/monitoring/v1"

	"github.com/machadovilaca/alerts-ui-management/pkg/k8s"
	"github.com/machadovilaca/alerts-ui-management/pkg/management"
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

	// wait for 5 seconds to ensure all PrometheusRules are updated
	for i := 0; i < 5; i++ {
		fmt.Println("Waiting for PrometheusRules to be updated...")
		time.Sleep(1 * time.Second)
	}

	rules, err := mgmClient.ListRules(ctx, management.PrometheusRuleOptions{Namespace: "openshift-monitoring"}, management.AlertRuleOptions{})
	if err != nil {
		log.Fatalf("Failed to list rules: %v", err)
	}
	fmt.Println("Rules:", len(rules))
	for _, rule := range rules {
		fmt.Printf("- %s: %+v\n", rule.Alert, rule.Labels)
	}

	alertRuleId := "AlertmanagerReceiversNotConfigured/f964c895e3c8eb806c0326742d875051af437ea4c4edaf8c89306317290bf8f1"
	alertRule := monitoringv1.Rule{
		Alert: "AlertmanagerReceiversNotConfigured",
		Labels: map[string]string{
			"namespace": "openshift-monitoring",
			"severity":  "info",
		},
	}

	err = mgmClient.UpdatePlatformAlertRule(ctx, alertRuleId, alertRule)
	if err != nil {
		log.Fatalf("Failed to update platform alert rule: %v", err)
	}

	fmt.Println("Platform alert rule updated successfully")

	for {
		alerts, err := mgmClient.GetAlerts(ctx, k8s.GetAlertsRequest{})
		if err != nil {
			log.Fatalf("Failed to get alerts: %v", err)
		}
		fmt.Println("\n\nAlerts:", len(alerts))
		for _, alert := range alerts {
			fmt.Printf("- %s: %+v\n", alert.Labels["alertname"], alert.Labels)
		}
		time.Sleep(10 * time.Second)
	}
}
