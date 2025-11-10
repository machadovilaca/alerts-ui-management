package k8s

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"time"

	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
)

const (
	prometheusRouteNamespace = "openshift-monitoring"
	prometheusRouteName      = "prometheus-k8s"
	prometheusAPIPath        = "/v1/alerts"
)

var (
	prometheusRoutePath = fmt.Sprintf("/apis/route.openshift.io/v1/namespaces/%s/routes/%s", prometheusRouteNamespace, prometheusRouteName)
)

type prometheusAlerts struct {
	clientset *kubernetes.Clientset
	config    *rest.Config
}

// GetAlertsRequest holds parameters for filtering alerts
type GetAlertsRequest struct {
	// Labels filters alerts by labels
	Labels map[string]string
	// State filters alerts by state: "firing", "pending", or "" for all states
	State string
}

type PrometheusAlert struct {
	Labels      map[string]string `json:"labels"`
	Annotations map[string]string `json:"annotations"`
	State       string            `json:"state"`
	ActiveAt    time.Time         `json:"activeAt"`
	Value       string            `json:"value"`
}

type prometheusAlertsResponse struct {
	Status string `json:"status"`
	Data   struct {
		Alerts []PrometheusAlert `json:"alerts"`
	} `json:"data"`
}

func newPrometheusAlerts(clientset *kubernetes.Clientset, config *rest.Config) PrometheusAlertsInterface {
	return &prometheusAlerts{
		clientset: clientset,
		config:    config,
	}
}

func (pa prometheusAlerts) GetAlerts(ctx context.Context, req GetAlertsRequest) ([]PrometheusAlert, error) {
	raw, err := pa.getAlertsViaProxy(ctx)
	if err != nil {
		return nil, err
	}

	var alertsResp prometheusAlertsResponse
	if err := json.Unmarshal(raw, &alertsResp); err != nil {
		return nil, fmt.Errorf("decode prometheus response: %w", err)
	}

	if alertsResp.Status != "success" {
		return nil, fmt.Errorf("prometheus API returned non-success status: %s", alertsResp.Status)
	}

	out := make([]PrometheusAlert, 0, len(alertsResp.Data.Alerts))
	for _, a := range alertsResp.Data.Alerts {
		// Filter alerts based on state if provided
		if req.State != "" && a.State != req.State {
			continue
		}

		// Filter alerts based on labels if provided
		if !labelsMatch(&req, &a) {
			continue
		}

		out = append(out, a)
	}
	return out, nil
}

func (pa prometheusAlerts) getAlertsViaProxy(ctx context.Context) ([]byte, error) {
	route, err := pa.clientset.CoreV1().RESTClient().
		Get().
		AbsPath(prometheusRoutePath).
		DoRaw(ctx)

	if err != nil {
		return nil, fmt.Errorf("failed to get prometheus route: %w", err)
	}

	var routeObj struct {
		Spec struct {
			Host string `json:"host"`
			Path string `json:"path"`
		} `json:"spec"`
	}
	if err := json.Unmarshal(route, &routeObj); err != nil {
		return nil, fmt.Errorf("failed to parse route: %w", err)
	}

	url := fmt.Sprintf("https://%s%s%s", routeObj.Spec.Host, routeObj.Spec.Path, prometheusAPIPath)

	transport := &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
	}
	client := &http.Client{Transport: transport}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}

	token := pa.config.BearerToken
	if token == "" && pa.config.BearerTokenFile != "" {
		tokenBytes, err := os.ReadFile(pa.config.BearerTokenFile)
		if err != nil {
			return nil, fmt.Errorf("load bearer token file: %w", err)
		}
		token = string(tokenBytes)
	}

	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}

	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("execute request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("unexpected status %d: %s", resp.StatusCode, string(body))
	}

	return io.ReadAll(resp.Body)
}

func labelsMatch(req *GetAlertsRequest, alert *PrometheusAlert) bool {
	for key, value := range req.Labels {
		if alertValue, exists := alert.Labels[key]; !exists || alertValue != value {
			return false
		}
	}

	return true
}
