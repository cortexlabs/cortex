#!/bin/bash

# Copyright 2019 Cortex Labs, Inc.
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

function setup_bucket() {
  if [ "$CORTEX_BUCKET" == "" ]; then
    account_id_hash=$(aws sts get-caller-identity | jq .Account | sha256sum | cut -f1 -d" " | cut -c -10)
    CORTEX_BUCKET="cortex-${account_id_hash}"
  fi

  if ! aws s3api head-bucket --bucket $CORTEX_BUCKET --output json 2>/dev/null; then
    if aws s3 ls "s3://$CORTEX_BUCKET" --output json 2>&1 | grep -q 'NoSuchBucket'; then
      echo -e "\n✓ Creating an S3 bucket: $CORTEX_BUCKET"
      aws s3api create-bucket --bucket $CORTEX_BUCKET \
                              --region $CORTEX_REGION \
                              --create-bucket-configuration LocationConstraint=$CORTEX_REGION \
                              >/dev/null
    else
      echo -e "\nA bucket named \"${CORTEX_BUCKET}\" already exists, but you do not have access to it"
      exit 1
    fi
  else
    echo -e "\n✓ Using an existing S3 bucket: $CORTEX_BUCKET"
  fi
}

function setup_cloudwatch_logs() {
  if ! aws logs list-tags-log-group --log-group-name $CORTEX_LOG_GROUP --region $CORTEX_REGION --output json 2>&1 | grep -q "\"tags\":"; then
    echo -e "\n✓ Creating a CloudWatch log group: $CORTEX_LOG_GROUP"
    aws logs create-log-group --log-group-name $CORTEX_LOG_GROUP --region $CORTEX_REGION
  else
    echo -e "\n✓ Using an existing CloudWatch log group: $CORTEX_LOG_GROUP"
  fi
}

function setup_configmap() {
  kubectl -n=$CORTEX_NAMESPACE create configmap 'cortex-config' \
    --from-literal='LOG_GROUP'=$CORTEX_LOG_GROUP \
    --from-literal='BUCKET'=$CORTEX_BUCKET \
    --from-literal='REGION'=$CORTEX_REGION \
    --from-literal='NAMESPACE'=$CORTEX_NAMESPACE \
    --from-literal='IMAGE_OPERATOR'=$CORTEX_IMAGE_OPERATOR \
    --from-literal='IMAGE_SPARK'=$CORTEX_IMAGE_SPARK \
    --from-literal='IMAGE_TF_TRAIN'=$CORTEX_IMAGE_TF_TRAIN \
    --from-literal='IMAGE_TF_SERVE'=$CORTEX_IMAGE_TF_SERVE \
    --from-literal='IMAGE_TF_API'=$CORTEX_IMAGE_TF_API \
    --from-literal='IMAGE_PYTHON_PACKAGER'=$CORTEX_IMAGE_PYTHON_PACKAGER \
    --from-literal='IMAGE_TF_TRAIN_GPU'=$CORTEX_IMAGE_TF_TRAIN_GPU \
    --from-literal='IMAGE_TF_SERVE_GPU'=$CORTEX_IMAGE_TF_SERVE_GPU \
    --from-literal='ENABLE_TELEMETRY'=$CORTEX_ENABLE_TELEMETRY \
    -o yaml --dry-run | kubectl apply -f - >/dev/null
}

function setup_secrets() {
  kubectl -n=$CORTEX_NAMESPACE create secret generic 'aws-credentials' \
    --from-literal='AWS_ACCESS_KEY_ID'=$AWS_ACCESS_KEY_ID \
    --from-literal='AWS_SECRET_ACCESS_KEY'=$AWS_SECRET_ACCESS_KEY \
    -o yaml --dry-run | kubectl apply -f - >/dev/null
}

function validate_cortex() {
  set +e

  echo -en "\nWaiting for Cortex to be ready "

  operator_load_balancer="waiting"
  api_load_balancer="waiting"
  operator_endpoint_reachable="waiting"
  operator_pod_ready_cycles=0
  operator_endpoint=""

  while true; do
    echo -n "."
    sleep 5

    operator_pod_name=$(kubectl -n=$CORTEX_NAMESPACE get pods -o=name --sort-by=.metadata.creationTimestamp | grep "^pod/operator-" | tail -1)
    if [ "$operator_pod_name" == "" ]; then
      operator_pod_ready_cycles=0
    else
      is_ready=$(kubectl -n=$CORTEX_NAMESPACE get "$operator_pod_name" -o jsonpath='{.status.containerStatuses[0].ready}')
      if [ "$is_ready" == "true" ]; then
        ((operator_pod_ready_cycles++))
      else
        operator_pod_ready_cycles=0
      fi
    fi

    if [ "$operator_load_balancer" != "ready" ]; then
      out=$(kubectl -n=$CORTEX_NAMESPACE get service nginx-controller-operator -o json | tr -d '[:space:]')
      if [[ $out != *'"loadBalancer":{"ingress":[{"'* ]]; then
        continue
      fi
      operator_load_balancer="ready"
    fi

    if [ "$api_load_balancer" != "ready" ]; then
      out=$(kubectl -n=$CORTEX_NAMESPACE get service nginx-controller-apis -o json | tr -d '[:space:]')
      if [[ $out != *'"loadBalancer":{"ingress":[{"'* ]]; then
        continue
      fi
      api_load_balancer="ready"
    fi

    if [ "$operator_endpoint" = "" ]; then
      operator_endpoint=$(kubectl -n=$CORTEX_NAMESPACE get service nginx-controller-operator -o json | tr -d '[:space:]' | sed 's/.*{\"hostname\":\"\(.*\)\".*/\1/')
    fi

    if [ "$operator_endpoint_reachable" != "ready" ]; then
      if ! curl $operator_endpoint >/dev/null 2>&1; then
        continue
      fi
      operator_endpoint_reachable="ready"
    fi

    if [ "$operator_pod_ready_cycles" == "0" ] && [ "$operator_pod_name" != "" ]; then
      num_restart=$(kubectl -n=$CORTEX_NAMESPACE get "$operator_pod_name" -o jsonpath='{.status.containerStatuses[0].restartCount}')
      if [[ $num_restart -ge 2 ]]; then
        echo -e "\n\nAn error occurred when starting the Cortex operator. View the logs with:"
        echo "  kubectl logs $operator_pod_name --namespace=$CORTEX_NAMESPACE"
        exit 1
      fi
      continue
    fi

    if [[ $operator_pod_ready_cycles -lt 3 ]]; then
      continue
    fi

    break
  done

  echo -e "\n\n✓ Cortex is ready!"

  if command -v cortex >/dev/null; then
    echo -e "\nPlease run \`cortex configure\` to make sure your CLI is configured correctly"
  fi
}

echo
eksctl utils write-kubeconfig --name=$CORTEX_CLUSTER

echo -e "\nInstalling Cortex ..."

setup_bucket
setup_cloudwatch_logs

envsubst < manifests/namespace.yaml | kubectl apply -f - >/dev/null

setup_configmap
setup_secrets

envsubst < manifests/spark.yaml | kubectl apply -f - >/dev/null
envsubst < manifests/argo.yaml | kubectl apply -f - >/dev/null
envsubst < manifests/nginx.yaml | kubectl apply -f - >/dev/null
envsubst < manifests/fluentd.yaml | kubectl apply -f - >/dev/null
envsubst < manifests/operator.yaml | kubectl apply -f - >/dev/null

validate_cortex
