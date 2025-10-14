#!/bin/bash

set -e

SLEEP_INTERVAL=10
STEP_I=0

echo "=========================================="
echo "Starting Informer Demo"
echo "=========================================="
echo ""

echo "Step $((STEP_I+=1)): Creating PrometheusRule with two alerts (Alert1 and Alert2)"
kubectl apply -f hack/examples/prometheus-rule.yaml
echo "Waiting ${SLEEP_INTERVAL}s..."
sleep ${SLEEP_INTERVAL}
echo ""

echo "Step $((STEP_I+=1)): Creating AlertRelabelConfig to drop Alert1"
kubectl apply -f hack/examples/alert-relabel-config.yaml
echo "Waiting ${SLEEP_INTERVAL}s..."
sleep ${SLEEP_INTERVAL}
echo ""

echo "Step $((STEP_I+=1)): Deleting AlertRelabelConfig"
kubectl delete -f hack/examples/alert-relabel-config-updated.yaml
echo "Waiting ${SLEEP_INTERVAL}s..."
sleep ${SLEEP_INTERVAL}
echo ""

echo "Step $((STEP_I+=1)): Updating PrometheusRule - changing Alert1 severity to critical"
kubectl apply -f hack/examples/prometheus-rule-updated.yaml
echo "Waiting ${SLEEP_INTERVAL}s..."
sleep ${SLEEP_INTERVAL}
echo ""

echo "Step $((STEP_I+=1)): Creating AlertRelabelConfig to change Alert1 severity to warning"
kubectl apply -f hack/examples/alert-relabel-config-updated.yaml
echo "Waiting ${SLEEP_INTERVAL}s..."
sleep ${SLEEP_INTERVAL}
echo ""

echo "Step $((STEP_I+=1)): Deleting AlertRelabelConfig"
kubectl delete -f hack/examples/alert-relabel-config-updated.yaml
echo "Waiting ${SLEEP_INTERVAL}s..."
sleep ${SLEEP_INTERVAL}
echo ""

echo "Step $((STEP_I+=1)): Deleting PrometheusRule"
kubectl delete -f hack/examples/prometheus-rule-updated.yaml
echo "Waiting ${SLEEP_INTERVAL}s..."
sleep ${SLEEP_INTERVAL}
echo ""

echo "=========================================="
echo "Demo completed!"
echo "=========================================="
