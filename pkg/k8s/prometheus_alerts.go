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
	alertmanagerRouteNamespace = "openshift-monitoring"
	alertmanagerRouteName      = "alertmanager-main"
	alertmanagerAPIPath        = "/v2/alerts"
)

var (
	alertmanagerRoutePath = fmt.Sprintf("/apis/route.openshift.io/v1/namespaces/%s/routes/%s", alertmanagerRouteNamespace, alertmanagerRouteName)
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

type alertmanagerAlert struct {
	Labels      map[string]string `json:"labels"`
	Annotations map[string]string `json:"annotations"`
	StartsAt    time.Time         `json:"startsAt"`
	EndsAt      time.Time         `json:"endsAt"`
	Status      struct {
		State       string   `json:"state"`
		SilencedBy  []string `json:"silencedBy"`
		InhibitedBy []string `json:"inhibitedBy"`
	} `json:"status"`
}

type alertmanagerAlertsResponse []alertmanagerAlert

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

	var alertsResp alertmanagerAlertsResponse
	if err := json.Unmarshal(raw, &alertsResp); err != nil {
		return nil, fmt.Errorf("decode alertmanager response: %w", err)
	}

	out := make([]ActiveAlert, 0, len(alertsResp))
	for _, a := range alertsResp {
		// Alertmanager v2 API uses "active" or "suppressed" state
		if a.Status.State == "active" {
			out = append(out, ActiveAlert{
				Name:        a.Labels["alertname"],
				Severity:    a.Labels["severity"],
				Labels:      a.Labels,
				Annotations: a.Annotations,
				State:       "firing", // Map "active" to "firing" for consistency
				ActiveAt:    a.StartsAt,
			})
		}
	}
	return out, nil
}

func (pa prometheusAlerts) getAlertsViaProxy(ctx context.Context) ([]byte, error) {
	route, err := pa.clientset.CoreV1().RESTClient().
		Get().
		AbsPath(alertmanagerRoutePath).
		DoRaw(ctx)

	if err != nil {
		return nil, fmt.Errorf("failed to get alertmanager route: %w", err)
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

	url := fmt.Sprintf("https://%s%s%s", routeObj.Spec.Host, routeObj.Spec.Path, alertmanagerAPIPath)

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
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("unexpected status %d: %s", resp.StatusCode, string(body))
	}

	return io.ReadAll(resp.Body)
}
