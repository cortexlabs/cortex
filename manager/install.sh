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

CORTEX_VERSION=master

arg1="$1"

function ensure_eks() {
  # Cluster statuses: https://github.com/aws/aws-sdk-go/blob/master/service/eks/api.go#L2785
  set +e
  cluster_info=$(eksctl get cluster --name=$CORTEX_CLUSTER_NAME --region=$CORTEX_REGION -o json)
  cluster_info_exit_code=$?
  set -e

  # No cluster
  if [ $cluster_info_exit_code -ne 0 ]; then
    if [ "$arg1" = "--update" ]; then
      echo "error: there is no cluster named \"$CORTEX_CLUSTER_NAME\" in $CORTEX_REGION; please update your configuration to point to an existing cortex cluster or create a cortex cluster with \`cortex cluster up\`"
      exit 1
    fi

    if [ $CORTEX_MIN_INSTANCES -lt 1 ]; then
      export CORTEX_DESIRED_INSTANCES=1
    else
      export CORTEX_DESIRED_INSTANCES=$CORTEX_MIN_INSTANCES
    fi

    echo -e "￮ spinning up the cluster ... (this will take about 15 minutes)\n"

    envsubst < eks_cluster.yaml | eksctl create cluster -f -

    # https://docs.aws.amazon.com/eks/latest/userguide/cni-upgrades.html
    kubectl apply -f https://raw.githubusercontent.com/aws/amazon-vpc-cni-k8s/v1.5.5/config/v1.5/aws-k8s-cni.yaml >/dev/null

    if [[ "$CORTEX_INSTANCE_TYPE" == p* ]] || [[ "$CORTEX_INSTANCE_TYPE" == g* ]]; then
      if [ "$CORTEX_SPOT" == "True" ]; then
        envsubst < eks_gpu_spot.yaml | eksctl create nodegroup -f -
      else
        envsubst < eks_gpu_ondemand.yaml | eksctl create nodegroup -f -
      fi
    else
      if [ "$CORTEX_SPOT" == "True" ]; then
        envsubst < eks_cpu_spot.yaml | eksctl create nodegroup -f -
      else
        envsubst < eks_cpu_ondemand.yaml | eksctl create nodegroup -f -
      fi
    fi

    if [ "$CORTEX_SPOT" == "True" ]; then
      asg_info=$(aws autoscaling describe-auto-scaling-groups --region $CORTEX_REGION --query "AutoScalingGroups[?contains(Tags[?Key==\`alpha.eksctl.io/cluster-name\`].Value, \`$CORTEX_CLUSTER_NAME\`)]|[?contains(Tags[?Key==\`alpha.eksctl.io/nodegroup-name\`].Value, \`ng-cortex-worker\`)]")
      asg_name=$(echo "$asg_info" | jq -r 'first | .AutoScalingGroupName')
      aws autoscaling suspend-processes --region $CORTEX_REGION --auto-scaling-group-name $asg_name --scaling-processes AZRebalance
    fi

    echo  # cluster is ready
    return
  fi

  set +e
  cluster_status=$(echo "$cluster_info" | jq -r 'first | .Status')
  set -e

  if [[ "$cluster_status" == "ACTIVE" ]] && [[ "$arg1" != "--update" ]]; then
    echo "error: there is already a cluster named \"$CORTEX_CLUSTER_NAME\" in $CORTEX_REGION"
    exit 1
  fi

  if [ "$cluster_status" == "DELETING" ]; then
    echo "error: your cortex cluster named \"$CORTEX_CLUSTER_NAME\" in $CORTEX_REGION is currently spinning down; please try again once it is completely deleted (may take a few minutes)"
    exit 1
  fi

  if [ "$cluster_status" == "CREATING" ]; then
    echo "error: your cortex cluster named \"$CORTEX_CLUSTER_NAME\" in $CORTEX_REGION is currently spinning up; please try again once it is ready"
    exit 1
  fi

  if [ "$cluster_status" == "FAILED" ]; then
    echo "error: your cortex cluster named \"$CORTEX_CLUSTER_NAME\" in $CORTEX_REGION is failed; delete it with \`eksctl delete cluster --name=$CORTEX_CLUSTER_NAME --region=$CORTEX_REGION\` and try again"
    exit 1
  fi

  # Check for change in min/max instances
  asg_info=$(aws autoscaling describe-auto-scaling-groups --region $CORTEX_REGION --query "AutoScalingGroups[?contains(Tags[?Key==\`alpha.eksctl.io/cluster-name\`].Value, \`$CORTEX_CLUSTER_NAME\`)]|[?contains(Tags[?Key==\`alpha.eksctl.io/nodegroup-name\`].Value, \`ng-cortex-worker\`)]")
  asg_name=$(echo "$asg_info" | jq -r 'first | .AutoScalingGroupName')
  asg_min_size=$(echo "$asg_info" | jq -r 'first | .MinSize')
  asg_max_size=$(echo "$asg_info" | jq -r 'first | .MaxSize')
  if [ "$asg_min_size" != "$CORTEX_MIN_INSTANCES" ]; then
    echo -n "￮ updating min instances to $CORTEX_MIN_INSTANCES "
    aws autoscaling update-auto-scaling-group --region $CORTEX_REGION --auto-scaling-group-name $asg_name --min-size=$CORTEX_MIN_INSTANCES
    echo "✓"
  fi
  if [ "$asg_max_size" != "$CORTEX_MAX_INSTANCES" ]; then
    echo -n "￮ updating max instances to $CORTEX_MAX_INSTANCES "
    aws autoscaling update-auto-scaling-group --region $CORTEX_REGION --auto-scaling-group-name $asg_name --max-size=$CORTEX_MAX_INSTANCES
    echo "✓"
  fi
}

function main() {
  ensure_eks

  eksctl utils write-kubeconfig --cluster=$CORTEX_CLUSTER_NAME --region=$CORTEX_REGION | grep -v "saved kubeconfig as" | grep -v "using region" | grep -v "eksctl version" || true

  envsubst < manifests/namespace.yaml | kubectl apply -f - >/dev/null

  # pre-download images on cortex cluster up
  if [ "$arg1" != "--update" ]; then
    if [[ "$CORTEX_INSTANCE_TYPE" == p* ]] || [[ "$CORTEX_INSTANCE_TYPE" == g* ]]; then
      envsubst < manifests/image-downloader-gpu.yaml | kubectl apply -f - &>/dev/null
    else
      envsubst < manifests/image-downloader-cpu.yaml | kubectl apply -f - &>/dev/null
    fi
  fi

  setup_bucket
  setup_cloudwatch_logs

  echo -n "￮ updating cluster configuration "
  setup_configmap
  setup_secrets
  echo "✓"

  echo -n "￮ configuring networking "
  setup_istio
  envsubst < manifests/apis.yaml | kubectl apply -f - >/dev/null
  echo "✓"

  echo -n "￮ configuring autoscaling "
  envsubst < manifests/cluster-autoscaler.yaml | kubectl apply -f - >/dev/null
  echo "✓"

  echo -n "￮ configuring logging "
  kubectl -n=cortex delete --ignore-not-found=true daemonset fluentd >/dev/null 2>&1  # Pods in DaemonSets cannot be modified
  until [ "$(kubectl -n=cortex get pods -l app=fluentd -o json | jq -j '.items | length')" -eq "0" ]; do echo -n "."; sleep 2; done
  envsubst < manifests/fluentd.yaml | kubectl apply -f - >/dev/null
  echo "✓"

  echo -n "￮ configuring metrics "
  kubectl -n=cortex delete --ignore-not-found=true daemonset cloudwatch-agent-statsd >/dev/null 2>&1  # Pods in DaemonSets cannot be modified
  until [ "$(kubectl -n=cortex get pods -l name=cloudwatch-agent-statsd -o json | jq -j '.items | length')" -eq "0" ]; do echo -n "."; sleep 2; done
  envsubst < manifests/metrics-server.yaml | kubectl apply -f - >/dev/null
  envsubst < manifests/statsd.yaml | kubectl apply -f - >/dev/null
  echo "✓"

  if [[ "$CORTEX_INSTANCE_TYPE" == p* ]] || [[ "$CORTEX_INSTANCE_TYPE" == g* ]]; then
    echo -n "￮ configuring gpu support "
    kubectl -n=kube-system delete --ignore-not-found=true daemonset nvidia-device-plugin-daemonset >/dev/null 2>&1  # Pods in DaemonSets cannot be modified
    until [ "$(kubectl -n=kube-system get pods -l name=nvidia-device-plugin-ds -o json | jq -j '.items | length')" -eq "0" ]; do echo -n "."; sleep 2; done
    envsubst < manifests/nvidia.yaml | kubectl apply -f - >/dev/null
    echo "✓"
  fi

  echo -n "￮ starting operator "
  kubectl -n=cortex delete --ignore-not-found=true --grace-period=10 deployment operator >/dev/null 2>&1
  until [ "$(kubectl -n=cortex get pods -l workloadID=operator -o json | jq -j '.items | length')" -eq "0" ]; do echo -n "."; sleep 2; done
  envsubst < manifests/operator.yaml | kubectl apply -f - >/dev/null
  echo "✓"

  validate_cortex

  if kubectl get daemonset image-downloader -n cortex &>/dev/null; then
    echo -n "￮ downloading docker images "
    i=0
    until [ "$(kubectl get daemonset image-downloader -n cortex -o 'jsonpath={.status.numberReady}')" == "$(kubectl get daemonset image-downloader -n cortex -o 'jsonpath={.status.desiredNumberScheduled}')" ]; do
      if [ $i -eq 100 ]; then break; fi  # give up after 5 minutes
      echo -n "."
      ((i=i+1))
      sleep 3
    done
    kubectl -n=cortex delete --ignore-not-found=true daemonset image-downloader &>/dev/null
    echo "✓"
  fi

  echo -n "￮ configuring cli "
  python update_cli_config.py "/.cortex/cli.yaml" "$CORTEX_ENVIRONMENT" "$operator_endpoint" "$CORTEX_AWS_ACCESS_KEY_ID" "$CORTEX_AWS_SECRET_ACCESS_KEY"
  echo "✓"

  echo -e "\ncortex is ready!"
}

function setup_bucket() {
  if ! aws s3api head-bucket --bucket $CORTEX_BUCKET --output json 2>/dev/null; then
    if aws s3 ls "s3://$CORTEX_BUCKET" --output json 2>&1 | grep -q 'NoSuchBucket'; then
      echo -n "￮ creating s3 bucket: $CORTEX_BUCKET "
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
      echo "✓"
    else
      echo "error: a bucket named \"${CORTEX_BUCKET}\" already exists, but you do not have access to it"
      exit 1
    fi
  else
    echo "￮ using existing s3 bucket: $CORTEX_BUCKET"
  fi
}

function setup_cloudwatch_logs() {
  if ! aws logs list-tags-log-group --log-group-name $CORTEX_LOG_GROUP --region $CORTEX_REGION --output json 2>&1 | grep -q "\"tags\":"; then
    echo -n "￮ creating cloudwatch log group: $CORTEX_LOG_GROUP "
    aws logs create-log-group --log-group-name $CORTEX_LOG_GROUP --region $CORTEX_REGION
    echo "✓"
  else
    echo "￮ using existing cloudwatch log group: $CORTEX_LOG_GROUP"
  fi
}

function setup_configmap() {
  kubectl -n=cortex create configmap 'cluster-config' \
    --from-file='cluster.yaml'=$CORTEX_CLUSTER_CONFIG_FILE \
    -o yaml --dry-run | kubectl apply -f - >/dev/null

  kubectl -n=cortex create configmap 'env-vars' \
    --from-literal='CORTEX_VERSION'=$CORTEX_VERSION \
    --from-literal='CORTEX_REGION'=$CORTEX_REGION \
    --from-literal='AWS_REGION'=$CORTEX_REGION \
    --from-literal='CORTEX_BUCKET'=$CORTEX_BUCKET \
    --from-literal='CORTEX_TELEMETRY_DISABLE'=$CORTEX_TELEMETRY_DISABLE \
    --from-literal='CORTEX_TELEMETRY_SENTRY_DSN'=$CORTEX_TELEMETRY_SENTRY_DSN \
    --from-literal='CORTEX_TELEMETRY_SEGMENT_WRITE_KEY'=$CORTEX_TELEMETRY_SEGMENT_WRITE_KEY \
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
  sleep 3  # Sleep a bit longer to be safe, since there are multiple Istio initialization containers
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

  echo -n "￮ waiting for load balancers "

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
        echo -e "\n\nan error occurred when starting the cortex operator. View the logs with:"
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

  echo "✓"
}

main
