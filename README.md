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

## Run the Automated Demo Script

The demo script creates, updates, and deletes PrometheusRule and AlertRelabelConfig resources to demonstrate the library's capabilities:

**Expected Behavior:**

**To observe the changes:**
- Run `go run main.go` in one terminal
- Run `./hack/examples/demo.sh` in another terminal
- Watch the first terminal for real-time updates as rules are created, modified, and deleted

Each step waits 10 seconds, giving you time to observe the changes in the monitoring output.

**Initial State:**
```
Watching for alert rules in 'default' namespace every 5 seconds...

Found 0 alert rules in 'default' namespace:
```

1. **Step 1**: Creates a PrometheusRule with Alert1
   - Should show Alert1 with severity `warning`
   ```
   Found 1 alert rules in 'default' namespace:
   - Alert1: warning
   ```

2. **Step 2**: Creates an AlertRelabelConfig to drop Alert1
   - No alert rules should be found
   ```
   Found 0 alert rules in 'default' namespace:
   ```

3. **Step 3**: Deletes the AlertRelabelConfig
   - Alert1 becomes visible with severity `warning`
   ```
   Found 1 alert rules in 'default' namespace:
   - Alert1: warning
   ```

4. **Step 4**: Updates the PrometheusRule - changes Alert1 severity to critical
   - Should show Alert1 with severity `critical`
   ```
   Found 1 alert rules in 'default' namespace:
   - Alert1: critical
   ```

5. **Step 5**: Creates an AlertRelabelConfig to change Alert1 severity to warning
   - Alert1 appears with severity `warning` (due to relabeling)
   ```
   Found 1 alert rules in 'default' namespace:
   - Alert1: warning
   ```

6. **Step 6**: Deletes the AlertRelabelConfig
   - Alert1 severity changes back to `critical`
   ```
   Found 1 alert rules in 'default' namespace:
   - Alert1: critical
   ```

7. **Step 7**: Deletes the PrometheusRule
   - No alert rules should be found
   ```
   Found 0 alert rules in 'default' namespace:
   ```
