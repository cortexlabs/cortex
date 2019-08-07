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
      if [ "$CORTEX_REGION" == "us-east-1" ]; then
        aws s3api create-bucket --bucket $CORTEX_BUCKET \
                                --region $CORTEX_REGION \
                                >/dev/null
      else
        aws s3api create-bucket --bucket $CORTEX_BUCKET \
                                --region $CORTEX_REGION \
                                --create-bucket-configuration LocationConstraint=$CORTEX_REGION \
                                >/dev/null
      fi
      echo "✓ Created S3 bucket: $CORTEX_BUCKET"
    else
      echo "error: a bucket named \"${CORTEX_BUCKET}\" already exists, but you do not have access to it"
      exit 1
    fi
  else
    echo "✓ Using existing S3 bucket: $CORTEX_BUCKET"
  fi
}

function setup_cloudwatch_logs() {
  if ! aws logs list-tags-log-group --log-group-name $CORTEX_LOG_GROUP --region $CORTEX_REGION --output json 2>&1 | grep -q "\"tags\":"; then
    aws logs create-log-group --log-group-name $CORTEX_LOG_GROUP --region $CORTEX_REGION
    echo "✓ Created CloudWatch log group: $CORTEX_LOG_GROUP"
  else
    echo "✓ Using existing CloudWatch log group: $CORTEX_LOG_GROUP"
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
    --from-literal='IMAGE_ONNX_SERVE'=$CORTEX_IMAGE_ONNX_SERVE \
    --from-literal='IMAGE_ONNX_SERVE_GPU'=$CORTEX_IMAGE_ONNX_SERVE_GPU \
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

function setup_istio() {
  echo -n "."
  envsubst < manifests/istio-namespace.yaml | kubectl apply -f - >/dev/null

  if ! kubectl get secret -n istio-system | grep -q istio-customgateway-certs; then
    WEBSITE=localhost
    openssl req -subj "/C=US/CN=$WEBSITE" -newkey rsa:2048 -nodes -keyout $WEBSITE.key -x509 -days 3650 -out $WEBSITE.crt >/dev/null 2>&1
    kubectl create -n istio-system secret tls istio-customgateway-certs --key $WEBSITE.key --cert $WEBSITE.crt >/dev/null
  fi

  helm template istio-manifests/istio-init --name istio-init --namespace istio-system | kubectl apply -f - >/dev/null
  until kubectl api-resources | grep -q virtualservice; do
    echo -n "."
    sleep 5
  done
  echo -n "."
  envsubst < manifests/istio-values.yaml | helm template istio-manifests/istio --values - --name istio --namespace istio-system | kubectl apply -f - >/dev/null
  envsubst < manifests/istio-metrics.yaml | kubectl apply -f - >/dev/null
  kubectl -n=istio-system create secret generic 'aws-credentials' \
    --from-literal='AWS_ACCESS_KEY_ID'=$AWS_ACCESS_KEY_ID \
    --from-literal='AWS_SECRET_ACCESS_KEY'=$AWS_SECRET_ACCESS_KEY \
    -o yaml --dry-run | kubectl apply -f - >/dev/null
  istio_patch="[
    {\"op\": \"add\", \"path\": \"/spec/template/spec/containers/0/env/-\", \"value\": {\"name\": \"AWS_ACCESS_KEY_ID\", \"valueFrom\": {\"secretKeyRef\": {\"name\": \"aws-credentials\", \"key\": \"AWS_ACCESS_KEY_ID\"}}}},\
    {\"op\": \"add\", \"path\": \"/spec/template/spec/containers/0/env/-\", \"value\": {\"name\": \"AWS_SECRET_ACCESS_KEY\", \"valueFrom\": {\"secretKeyRef\": {\"name\": \"aws-credentials\", \"key\": \"AWS_SECRET_ACCESS_KEY\"}}}},\
  ]"
  kubectl patch deployment istio-telemetry -n istio-system --type='json' -p="$istio_patch"
}

function validate_cortex() {
  set +e

  echo -en "￮ Waiting for load balancers "

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
      out=$(kubectl -n=istio-system get service operator-ingressgateway -o json | tr -d '[:space:]')
      if [[ $out != *'"loadBalancer":{"ingress":[{"'* ]]; then
        continue
      fi
      operator_load_balancer="ready"
    fi

    if [ "$api_load_balancer" != "ready" ]; then
      out=$(kubectl -n=istio-system get service apis-ingressgateway -o json | tr -d '[:space:]')
      if [[ $out != *'"loadBalancer":{"ingress":[{"'* ]]; then
        continue
      fi
      api_load_balancer="ready"
    fi

    if [ "$operator_endpoint" = "" ]; then
      operator_endpoint=$(kubectl -n=istio-system get service operator-ingressgateway -o json | tr -d '[:space:]' | sed 's/.*{\"hostname\":\"\(.*\)\".*/\1/')
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

  echo -e "\n✓ Load balancers are ready"
}

eksctl utils write-kubeconfig --name=$CORTEX_CLUSTER --region=$CORTEX_REGION | grep -v "saved kubeconfig as" || true

setup_bucket
setup_cloudwatch_logs

envsubst < manifests/namespace.yaml | kubectl apply -f - >/dev/null

setup_configmap
setup_secrets
echo "✓ Updated cluster configuration"

echo -en "￮ Configuring networking "
setup_istio
envsubst < manifests/apis.yaml | kubectl apply -f - >/dev/null
echo -e "\n✓ Configured networking"

envsubst < manifests/cluster-autoscaler.yaml | kubectl apply -f - >/dev/null
echo "✓ Configured autoscaling"

envsubst < manifests/fluentd.yaml | kubectl apply -f - >/dev/null
echo "✓ Configured logging"

envsubst < manifests/metrics-server.yaml | kubectl apply -f - >/dev/null
echo "✓ Configured metrics"

envsubst < manifests/spark.yaml | kubectl apply -f - >/dev/null

if [[ "$CORTEX_NODE_TYPE" == p* ]] || [[ "$CORTEX_NODE_TYPE" == g* ]]; then
  envsubst < manifests/nvidia.yaml | kubectl apply -f - >/dev/null
  echo "✓ Configured GPU support"
fi

envsubst < manifests/operator.yaml | kubectl apply -f - >/dev/null
echo "✓ Started Cortex operator"

validate_cortex

echo -e "\n✓ Cortex is ready!"
