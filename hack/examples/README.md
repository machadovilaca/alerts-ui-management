# Demo Script

This directory contains example YAML files and an automated demo script that demonstrates the library's capabilities by creating, updating, and deleting PrometheusRule and AlertRelabelConfig resources.

## Files

- `demo.sh` - Automated demo script
- `prometheus-rule.yaml` - Initial PrometheusRule with Alert1 and Alert2
- `prometheus-rule-updated.yaml` - Updated PrometheusRule changing Alert1 severity to critical
- `alert-relabel-config.yaml` - AlertRelabelConfig to drop Alert1
- `alert-relabel-config-updated.yaml` - AlertRelabelConfig to change Alert1 severity to warning
- `main.go` - Alternative demo application

## Running the Demo

**Prerequisites:**
- Kubernetes cluster with prometheus-operator installed
- `kubectl` configured to access the cluster
- Go installed (if running the monitoring application)

**To observe the changes:**
1. Run `go run main.go` in one terminal (from project root)
2. Run `./hack/examples/demo.sh` in another terminal
3. Watch the first terminal for real-time updates as rules are created, modified, and deleted

Each step waits 10 seconds, giving you time to observe the changes in the monitoring output.

## Expected Behavior

**Initial State:**
```
Watching for alert rules in 'default' namespace every 5 seconds...

Found 0 alert rules in 'default' namespace:
```

### Step 1: Create PrometheusRule
Creates a PrometheusRule with Alert1 - should show Alert1 with severity `warning`
```
Found 1 alert rules in 'default' namespace:
- Alert1: warning
```

### Step 2: Create AlertRelabelConfig to Drop Alert1
No alert rules should be found (Alert1 is dropped by the relabel config)
```
Found 0 alert rules in 'default' namespace:
```

### Step 3: Delete AlertRelabelConfig
Alert1 becomes visible again with severity `warning`
```
Found 1 alert rules in 'default' namespace:
- Alert1: warning
```

### Step 4: Update PrometheusRule
Changes Alert1 severity to critical - should show Alert1 with severity `critical`
```
Found 1 alert rules in 'default' namespace:
- Alert1: critical
```

### Step 5: Create AlertRelabelConfig to Change Severity
Alert1 appears with severity `warning` (due to relabeling)
```
Found 1 alert rules in 'default' namespace:
- Alert1: warning
```

### Step 6: Delete AlertRelabelConfig
Alert1 severity changes back to `critical`
```
Found 1 alert rules in 'default' namespace:
- Alert1: critical
```

### Step 7: Delete PrometheusRule
No alert rules should be found
```
Found 0 alert rules in 'default' namespace:
```

## What the Demo Shows

This demo demonstrates:

1. **Real-time synchronization**: Changes to PrometheusRule and AlertRelabelConfig resources are immediately reflected in the monitoring output
2. **Hash-based tracking**: The library tracks rules by content hash, enabling reliable identification across updates
3. **Alert relabeling**: How AlertRelabelConfig resources can modify or drop alerts before they reach Alertmanager
4. **Platform protection**: The library distinguishes between platform and user-defined rules (though this demo focuses on user-defined rules in the `default` namespace)
