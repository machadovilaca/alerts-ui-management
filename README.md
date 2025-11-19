# Alerts UI Management

A Go library for managing Prometheus alerting rules in Kubernetes clusters.
Provides a higher-level API to create, update, delete, and list alerting rules
stored in PrometheusRule CRDs from the prometheus-operator.

## Key Features

- **Hash-based rule identification**: Alert rules are identified by SHA256
hashes computed from rule content, enabling reliable tracking across updates

- **Platform vs user-defined rules**: Protects platform-managed rules
(PrometheusRules starting with `openshift-`) from accidental modification

- **Real-time synchronization**: Uses Kubernetes informers to maintain
up-to-date mapping of rules

## Project Structure

```
alerts-ui-management/
├── pkg/
│   ├── k8s/                    # Low-level Kubernetes client with PrometheusRules, AlertRelabelConfigs and Prometheus Alerts API operations
│   └── management/             # High-level management API for alert rules
│       └── mapper/             # Hash-based rule identifier mapping
├── main.go                     # Demo application
└── hack/examples/
    ├── demo.sh                 # Automated demo script
    └── *.yaml                  # Example PrometheusRule resources
```

## HTTP API Endpoints

The library includes HTTP endpoints for accessing alert data. When running the demo application (`go run main.go`), the following endpoints are available:

### Available Endpoints

#### GET `/api/v1/alerting/health`
Health check endpoint that returns the service status.

**Example:**
```bash
curl http://localhost:8080/api/v1/alerting/health
```

**Response:**
```json
{"status":"ok"}
```

#### GET `/api/v1/alerting/alerts`
Retrieves active alerts from the cluster, with optional label-based filtering.

**Query Parameters:**
- `labels[key]=value` - Filter alerts by label key-value pairs

**Examples:**

Get all active alerts:
```bash
curl http://localhost:8080/api/v1/alerting/alerts
```

Filter alerts by severity:
```bash
curl --globoff "http://localhost:8080/api/v1/alerting/alerts?labels[severity]=warning"
```

Filter alerts by multiple labels:
```bash
curl --globoff "http://localhost:8080/api/v1/alerting/alerts?labels[severity]=warning&labels[namespace]=openshift-monitoring"
```

**Response:**
```json
{
  "alerts": [
    {
      "name": "AlertName",
      "severity": "warning",
      "labels": {
        "alertname": "AlertName",
        "severity": "warning",
        "namespace": "default"
      },
      "annotations": {
        "description": "Alert description",
        "summary": "Alert summary"
      },
      "state": "firing",
      "activeAt": "2025-11-03T10:30:00Z"
    }
  ]
}
```
