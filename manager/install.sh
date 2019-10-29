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

arg1="$1"

function ensure_eks() {
  if ! eksctl utils describe-stacks --name=$CORTEX_CLUSTER_NAME --region=$CORTEX_REGION >/dev/null 2>&1; then
    if [ "$arg1" = "--update" ]; then
      echo "error: there isn't a Cortex cluster named \"$CORTEX_CLUSTER_NAME\" in $CORTEX_REGION; please update your configuration to point to an existing Cortex cluster or create a Cortex cluster with \`cortex cluster up\`"
      exit 1
    fi

    echo -e "￮ Spinning up the cluster ... (this will take about 15 minutes)\n"
    envsubst < eks.yaml | eksctl create cluster -f -
    echo "\n✓ Spun up the cluster"
    return
  fi

  echo "✓ Cluster is running"

  # Check if instance type changed
  ng_info=$(eksctl get nodegroup --cluster=$CORTEX_CLUSTER_NAME --region=$CORTEX_REGION --name ng-1 -o json)
  ng_instance_type=$(echo "$ng_info" | jq -r ".[] | select( .Cluster == \"$CORTEX_CLUSTER_NAME\" ) | select( .Name == \"ng-1\" ) | .InstanceType")
  if [ "$ng_instance_type" != "$CORTEX_INSTANCE_TYPE" ]; then
    echo "error: Cortex does not currently support changing the instance type of a running cluster; please run \`cortex cluster down\` followed by \`cortex cluster up\` to create a new cluster"
    exit 1
  fi

  # Check for change in min/max instances
  asg_info=$(aws autoscaling describe-auto-scaling-groups --region $CORTEX_REGION --query 'AutoScalingGroups[?contains(Tags[?Key==`alpha.eksctl.io/nodegroup-name`].Value, `ng-1`)]')
  asg_name=$(echo "$asg_info" | jq -r 'first | .AutoScalingGroupName')
  asg_min_size=$(echo "$asg_info" | jq -r 'first | .MinSize')
  asg_max_size=$(echo "$asg_info" | jq -r 'first | .MaxSize')
  if [ "$asg_min_size" != "$CORTEX_MIN_INSTANCES" ]; then
    aws autoscaling update-auto-scaling-group --region $CORTEX_REGION --auto-scaling-group-name $asg_name --min-size=$CORTEX_MIN_INSTANCES
    echo "✓ Updated min instances to $CORTEX_MIN_INSTANCES"
  fi
  if [ "$asg_max_size" != "$CORTEX_MAX_INSTANCES" ]; then
    aws autoscaling update-auto-scaling-group --region $CORTEX_REGION --auto-scaling-group-name $asg_name --max-size=$CORTEX_MAX_INSTANCES
    echo "✓ Updated max instances to $CORTEX_MAX_INSTANCES"
  fi
}

function main() {
  ensure_eks

  eksctl utils write-kubeconfig --name=$CORTEX_CLUSTER_NAME --region=$CORTEX_REGION | grep -v "saved kubeconfig as" | grep -v "using region" | grep -v "eksctl version" || true

  setup_bucket
  setup_cloudwatch_logs

  envsubst < manifests/namespace.yaml | kubectl apply -f - >/dev/null

  setup_configmap
  setup_secrets
  echo "✓ Updated cluster configuration"

  echo -n "￮ Configuring networking "
  # https://docs.aws.amazon.com/eks/latest/userguide/cni-upgrades.html
  kubectl apply -f https://raw.githubusercontent.com/aws/amazon-vpc-cni-k8s/v1.5.4/config/v1.5/aws-k8s-cni.yaml >/dev/null
  until [ "$(kubectl get daemonset aws-node -n kube-system -o 'jsonpath={.status.updatedNumberScheduled}')" == "$(kubectl get daemonset aws-node -n kube-system -o 'jsonpath={.status.desiredNumberScheduled}')" ]; do
    echo -n "."
    sleep 3
  done
  setup_istio
  envsubst < manifests/apis.yaml | kubectl apply -f - >/dev/null
  echo -e "\n✓ Configured networking"

  envsubst < manifests/cluster-autoscaler.yaml | kubectl apply -f - >/dev/null
  echo "✓ Configured autoscaling"

  kubectl -n=cortex delete --ignore-not-found=true daemonset fluentd >/dev/null 2>&1  # Pods in DaemonSets cannot be modified
  envsubst < manifests/fluentd.yaml | kubectl apply -f - >/dev/null
  echo "✓ Configured logging"

  kubectl -n=cortex delete --ignore-not-found=true daemonset cloudwatch-agent-statsd >/dev/null 2>&1  # Pods in DaemonSets cannot be modified
  envsubst < manifests/metrics-server.yaml | kubectl apply -f - >/dev/null
  envsubst < manifests/statsd.yaml | kubectl apply -f - >/dev/null
  echo "✓ Configured metrics"

  if [[ "$CORTEX_INSTANCE_TYPE" == p* ]] || [[ "$CORTEX_INSTANCE_TYPE" == g* ]]; then
    kubectl -n=cortex delete --ignore-not-found=true daemonset nvidia-device-plugin-daemonset >/dev/null 2>&1  # Pods in DaemonSets cannot be modified
    envsubst < manifests/nvidia.yaml | kubectl apply -f - >/dev/null
    echo "✓ Configured GPU support"
  fi

  kubectl -n=cortex delete --ignore-not-found=true deployment operator >/dev/null 2>&1
  envsubst < manifests/operator.yaml | kubectl apply -f - >/dev/null
  echo "✓ Started operator"

  validate_cortex

  echo "{\"cortex_url\": \"$operator_endpoint\", \"aws_access_key_id\": \"$CORTEX_AWS_ACCESS_KEY_ID\", \"aws_secret_access_key\": \"$CORTEX_AWS_SECRET_ACCESS_KEY\"}" > /.cortex/default.json
  echo "✓ Configured CLI"

  echo -e "\n✓ Cortex is ready!"
}

function setup_bucket() {
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
  kubectl -n=cortex create configmap 'cluster-config' \
    --from-file='cluster.yaml'='/.cortex/cluster.yaml' \
    -o yaml --dry-run | kubectl apply -f - >/dev/null
}

function setup_secrets() {
  kubectl -n=cortex create secret generic 'aws-credentials' \
    --from-literal='AWS_ACCESS_KEY_ID'=$CORTEX_AWS_ACCESS_KEY_ID \
    --from-literal='AWS_SECRET_ACCESS_KEY'=$CORTEX_AWS_SECRET_ACCESS_KEY \
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
    sleep 3
  done
  echo -n "."

  helm template istio-manifests/istio-cni --name istio-cni --namespace kube-system | kubectl apply -f - >/dev/null
  until [ "$(kubectl get daemonset istio-cni-node -n kube-system -o 'jsonpath={.status.numberReady}')" == "$(kubectl get daemonset istio-cni-node -n kube-system -o 'jsonpath={.status.desiredNumberScheduled}')" ]; do
    echo -n "."
    sleep 3
  done

  envsubst < manifests/istio-values.yaml | helm template istio-manifests/istio --values - --name istio --namespace istio-system | kubectl apply -f - >/dev/null
}

function validate_cortex() {
  set +e

  echo -n "￮ Waiting for load balancers "

  operator_load_balancer="waiting"
  api_load_balancer="waiting"
  operator_endpoint_reachable="waiting"
  operator_pod_ready_cycles=0
  operator_endpoint=""

  while true; do
    echo -n "."
    sleep 3

    operator_pod_name=$(kubectl -n=cortex get pods -o=name --sort-by=.metadata.creationTimestamp | grep "^pod/operator-" | tail -1)
    if [ "$operator_pod_name" == "" ]; then
      operator_pod_ready_cycles=0
    else
      is_ready=$(kubectl -n=cortex get "$operator_pod_name" -o jsonpath='{.status.containerStatuses[0].ready}')
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
      num_restart=$(kubectl -n=cortex get "$operator_pod_name" -o jsonpath='{.status.containerStatuses[0].restartCount}')
      if [[ $num_restart -ge 2 ]]; then
        echo -e "\n\nAn error occurred when starting the Cortex operator. View the logs with:"
        echo "  kubectl logs $operator_pod_name --namespace=cortex"
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

main
