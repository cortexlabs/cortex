#!/usr/bin/env bash

CLUSTER_CONFIG=$1

portForwardCMD="kubectl port-forward -n default prometheus-prometheus-0 9090"
kill $(pgrep -f "${portForwardCMD}") >/dev/null 2>&1 || true

echo "Port-forwarding Prometheus to localhost:9090"
eval "${portForwardCMD}" >/dev/null 2>&1 &

CORTEX_PROMETHEUS_URL="http://localhost:9090" go run ./main.go -config "${CLUSTER_CONFIG}"
