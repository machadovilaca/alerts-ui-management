#!/bin/bash

set -e

SLEEP_INTERVAL=10

echo "=========================================="
echo "Starting Informer Demo"
echo "=========================================="
echo ""

echo "Step 1: Creating PrometheusRule with Alert1 (severity: warning)"
kubectl apply -f hack/examples/prometheus-rule.yaml
echo "Waiting ${SLEEP_INTERVAL}s..."
sleep ${SLEEP_INTERVAL}
echo ""

echo "Step 2: Creating AlertRelabelConfig to drop Alert1"
kubectl apply -f hack/examples/alert-relabel-config.yaml
echo "Waiting ${SLEEP_INTERVAL}s..."
sleep ${SLEEP_INTERVAL}
echo ""

echo "Step 3: Updating PrometheusRule - changing Alert1 severity to critical"
kubectl apply -f hack/examples/prometheus-rule-updated.yaml
echo "Waiting ${SLEEP_INTERVAL}s..."
sleep ${SLEEP_INTERVAL}
echo ""

echo "Step 4: Updating AlertRelabelConfig - adding severity replacement rule"
kubectl apply -f hack/examples/alert-relabel-config-updated.yaml
echo "Waiting ${SLEEP_INTERVAL}s..."
sleep ${SLEEP_INTERVAL}
echo ""

echo "Step 5: Deleting AlertRelabelConfig"
kubectl delete -f hack/examples/alert-relabel-config-updated.yaml
echo "Waiting ${SLEEP_INTERVAL}s..."
sleep ${SLEEP_INTERVAL}
echo ""

echo "Step 6: Deleting PrometheusRule"
kubectl delete -f hack/examples/prometheus-rule-updated.yaml
echo "Waiting ${SLEEP_INTERVAL}s..."
sleep ${SLEEP_INTERVAL}
echo ""

echo "=========================================="
echo "Demo completed!"
echo "=========================================="
