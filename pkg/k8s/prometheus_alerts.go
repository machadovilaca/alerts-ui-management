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
	thanosQuerierRouteNamespace = "openshift-monitoring"
	thanosQuerierRouteName      = "thanos-querier"
	thanosQuerierAPIPath        = "/v1/alerts"
)

var (
	thanosQuerierRoutePath = fmt.Sprintf("/apis/route.openshift.io/v1/namespaces/%s/routes/%s", thanosQuerierRouteNamespace, thanosQuerierRouteName)
)

type prometheusAlerts struct {
	clientset *kubernetes.Clientset
	config    *rest.Config
}

type ActiveAlert struct {
	Name        string            `json:"name"`
	Severity    string            `json:"severity"`
	Labels      map[string]string `json:"labels"`
	Annotations map[string]string `json:"annotations"`
	State       string            `json:"state"`
	ActiveAt    time.Time         `json:"activeAt"`
}

type prometheusAlertsResponse struct {
	Status string `json:"status"`
	Data   struct {
		Alerts []struct {
			Labels      map[string]string `json:"labels"`
			Annotations map[string]string `json:"annotations"`
			State       string            `json:"state"`
			ActiveAt    time.Time         `json:"activeAt"`
			Value       string            `json:"value,omitempty"`
		} `json:"alerts"`
	} `json:"data"`
}

func newPrometheusAlerts(clientset *kubernetes.Clientset, config *rest.Config) PrometheusAlertsInterface {
	return &prometheusAlerts{
		clientset: clientset,
		config:    config,
	}
}

func (pa prometheusAlerts) GetActiveAlerts(ctx context.Context) ([]ActiveAlert, error) {
	raw, err := pa.getAlertsViaProxy(ctx)
	if err != nil {
		return nil, err
	}

	var alertsResp prometheusAlertsResponse
	if err := json.Unmarshal(raw, &alertsResp); err != nil {
		return nil, fmt.Errorf("decode prometheus response: %w", err)
	}
	if alertsResp.Status != "success" {
		return nil, fmt.Errorf("prometheus returned status=%s", alertsResp.Status)
	}

	out := make([]ActiveAlert, 0, len(alertsResp.Data.Alerts))
	for _, a := range alertsResp.Data.Alerts {
		if a.State == "firing" || a.State == "pending" {
			out = append(out, ActiveAlert{
				Name:        a.Labels["alertname"],
				Severity:    a.Labels["severity"],
				Labels:      a.Labels,
				Annotations: a.Annotations,
				State:       a.State,
				ActiveAt:    a.ActiveAt,
			})
		}
	}
	return out, nil
}

func (pa prometheusAlerts) getAlertsViaProxy(ctx context.Context) ([]byte, error) {
	route, err := pa.clientset.CoreV1().RESTClient().
		Get().
		AbsPath(thanosQuerierRoutePath).
		DoRaw(ctx)

	if err != nil {
		return nil, fmt.Errorf("failed to get thanos-querier route: %w", err)
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

	url := fmt.Sprintf("https://%s%s%s", routeObj.Spec.Host, routeObj.Spec.Path, thanosQuerierAPIPath)

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
