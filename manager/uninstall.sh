#!/bin/bash

# Copyright 2021 Cortex Labs, Inc.
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#     http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

set -e

EKSCTL_TIMEOUT=45m

arg1="$1"

function main() {
  if [ "$CORTEX_PROVIDER" == "aws" ]; then
    uninstall_aws
  elif [ "$CORTEX_PROVIDER" == "gcp" ]; then
    uninstall_gcp
  fi
}

function uninstall_gcp() {
  gcloud auth activate-service-account --key-file $GOOGLE_APPLICATION_CREDENTIALS 2> /dev/stdout 1> /dev/null | (grep -v "Activated service account credentials" || true)
  gcloud container clusters get-credentials $CORTEX_CLUSTER_NAME --project $CORTEX_GCP_PROJECT --region $CORTEX_GCP_ZONE 2> /dev/stdout 1> /dev/null | (grep -v "Fetching cluster" | grep -v "kubeconfig entry generated" || true)

  if [ "$arg1" != "--keep-volumes" ]; then
    uninstall_prometheus
    uninstall_grafana
  fi
}

function uninstall_aws() {
  echo
  aws eks --region $CORTEX_REGION update-kubeconfig --name $CORTEX_CLUSTER_NAME >/dev/null
  eksctl delete cluster --wait --name=$CORTEX_CLUSTER_NAME --region=$CORTEX_REGION --timeout=$EKSCTL_TIMEOUT
  echo -e "\n✓ done spinning down the cluster"
}

function uninstall_prometheus() {
  kubectl get configmap cluster-config -o jsonpath='{.data.cluster\.yaml}' > ./cluster.yaml

  # delete resources to detach disk
  python render_template.py ./cluster.yaml manifests/prometheus-monitoring.yaml.j2 | kubectl delete -f - >/dev/null
  kubectl delete pvc --namespace default prometheus-prometheus-db-prometheus-prometheus-0 >/dev/null
}

function uninstall_grafana() {
  kubectl delete statefulset --namespace default grafana >/dev/null
  kubectl delete pvc --namespace default grafana-storage >/dev/null
}

main
