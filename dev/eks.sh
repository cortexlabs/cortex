#!/bin/bash
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")"/.. >/dev/null && pwd)"
source $ROOT/dev/config/k8s.sh

function eks_set_cluster() {
  eksctl utils write-kubeconfig --name=$K8S_NAME
  kubectl config set-context $(kubectl config current-context) --namespace="cortex"
}

if [ "$1" = "start" ]; then
  eksctl create cluster --version=1.11 --name=$K8S_NAME  --region $K8S_REGION --nodes=$K8S_NODE_COUNT --node-type=$K8S_NODE_INSTANCE_TYPE
  eks_set_cluster

elif [ "$1" = "update" ]; then
  echo "Not implemented"

elif [ "$1" = "stop" ]; then
  $ROOT/cortex.sh -c=$ROOT/dev/config/cortex.sh uninstall 2>/dev/null || true
  eksctl delete cluster --name=$K8S_NAME

elif [ "$1" = "set" ]; then
  eks_set_cluster
fi
