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

set -eo pipefail

export CORTEX_VERSION=master
export CORTEX_VERSION_MINOR=master
EKSCTL_TIMEOUT=45m
mkdir /workspace

arg1="$1"

function main() {
  if [ "$arg1" = "--update" ]; then
    cluster_configure
  else
    cluster_up
  fi
}

function cluster_up() {
    if [ "$CORTEX_PROVIDER" == "aws" ]; then
      cluster_up_aws
    elif [ "$CORTEX_PROVIDER" == "gcp" ]; then
      cluster_up_gcp
    fi
}

function cluster_up_aws() {
  create_eks

  start_pre_download_images

  echo -n "￮ updating cluster configuration "
  setup_configmap
  setup_secrets
  echo "✓"

  echo -n "￮ configuring networking (this might take a few minutes) "
  setup_istio
  python render_template.py $CORTEX_CLUSTER_CONFIG_FILE manifests/apis.yaml.j2 > /workspace/apis.yaml
  kubectl apply -f /workspace/apis.yaml >/dev/null
  echo "✓"

  echo -n "￮ configuring autoscaling "
  python render_template.py $CORTEX_CLUSTER_CONFIG_FILE manifests/cluster-autoscaler.yaml.j2 > /workspace/cluster-autoscaler.yaml
  kubectl apply -f /workspace/cluster-autoscaler.yaml >/dev/null
  echo "✓"

  echo -n "￮ configuring logging "
  python render_template.py $CORTEX_CLUSTER_CONFIG_FILE manifests/fluent-bit.yaml.j2 | kubectl apply -f - >/dev/null
  envsubst < manifests/event-exporter.yaml | kubectl apply -f - >/dev/null
  echo "✓"

  echo -n "￮ configuring metrics "
  envsubst < manifests/metrics-server.yaml | kubectl apply -f - >/dev/null
  setup_prometheus
  setup_grafana
  echo "✓"

  if [[ "$CORTEX_INSTANCE_TYPE" == p* ]] || [[ "$CORTEX_INSTANCE_TYPE" == g* ]]; then
    echo -n "￮ configuring gpu support "
    envsubst < manifests/nvidia_aws.yaml | kubectl apply -f - >/dev/null
    echo "✓"
  fi

  if [[ "$CORTEX_INSTANCE_TYPE" == inf* ]]; then
    echo -n "￮ configuring inf support "
    envsubst < manifests/inferentia.yaml | kubectl apply -f - >/dev/null
    echo "✓"
  fi

  restart_operator

  validate_cortex

  await_pre_download_images

  echo -e "\ncortex is ready!"
  if [ "$CORTEX_OPERATOR_LOAD_BALANCER_SCHEME" == "internal" ]; then
    echo -e "\nnote: you will need to configure VPC Peering to connect to your cluster: https://docs.cortex.dev/v/${CORTEX_VERSION_MINOR}/"
  fi

  print_endpoints_aws
}

function cluster_up_gcp() {
  gcloud auth activate-service-account --key-file $GOOGLE_APPLICATION_CREDENTIALS 2> /dev/stdout 1> /dev/null | (grep -v "Activated service account credentials" || true)
  gcloud container clusters get-credentials $CORTEX_CLUSTER_NAME --project $CORTEX_GCP_PROJECT --region $CORTEX_GCP_ZONE 2> /dev/stdout 1> /dev/null | (grep -v "Fetching cluster" | grep -v "kubeconfig entry generated" || true)

  start_pre_download_images

  echo -n "￮ updating cluster configuration "
  setup_configmap_gcp
  setup_secrets_gcp
  echo "✓"

  echo -n "￮ configuring networking (this will take a few minutes) "
  setup_istio
  python render_template.py $CORTEX_CLUSTER_CONFIG_FILE manifests/apis.yaml.j2 > /workspace/apis.yaml
  kubectl apply -f /workspace/apis.yaml >/dev/null
  echo "✓"

  echo -n "￮ configuring autoscaling "
  python render_template.py $CORTEX_CLUSTER_CONFIG_FILE manifests/cluster-autoscaler.yaml.j2 > /workspace/cluster-autoscaler.yaml
  kubectl apply -f /workspace/cluster-autoscaler.yaml >/dev/null
  echo "✓"

  echo -n "￮ configuring logging "
  python render_template.py $CORTEX_CLUSTER_CONFIG_FILE manifests/fluent-bit.yaml.j2 | kubectl apply -f - >/dev/null
  envsubst < manifests/event-exporter.yaml | kubectl apply -f - >/dev/null
  echo "✓"

  echo -n "￮ configuring metrics "
  setup_prometheus
  setup_grafana
  echo "✓"

  if [ -n "$CORTEX_ACCELERATOR_TYPE" ]; then
    echo -n "￮ configuring gpu support "
    envsubst < manifests/nvidia_gcp.yaml | kubectl apply -f - >/dev/null
    echo "✓"
  fi

  restart_operator

  validate_cortex_gcp

  await_pre_download_images

  echo -e "\ncortex is ready!"

  print_endpoints_gcp
}

function cluster_configure() {
  check_eks

  resize_nodegroup

  echo -n "￮ updating cluster configuration "
  setup_configmap
  setup_secrets
  echo "✓"

  # this is necessary since max_instances may have been updated
  echo -n "￮ configuring autoscaling "
  python render_template.py $CORTEX_CLUSTER_CONFIG_FILE manifests/cluster-autoscaler.yaml.j2 > /workspace/cluster-autoscaler.yaml
  kubectl apply -f /workspace/cluster-autoscaler.yaml >/dev/null
  echo "✓"

  restart_operator

  validate_cortex

  echo -e "\ncortex is ready!"

  print_endpoints_aws
}

# creates the eks cluster and configures kubectl
function create_eks() {
  set +e
  cluster_info=$(eksctl get cluster --name=$CORTEX_CLUSTER_NAME --region=$CORTEX_REGION -o json 2> /dev/null)
  cluster_info_exit_code=$?
  set -e

  # cluster already exists
  if [ $cluster_info_exit_code -eq 0 ]; then
    set +e
    # cluster statuses: https://github.com/aws/aws-sdk-go/blob/master/service/eks/api.go#L6883
    cluster_status=$(echo "$cluster_info" | jq -r 'first | .Status')
    set -e

    if [ "$cluster_status" == "ACTIVE" ]; then
      echo "error: there is already a cluster named \"$CORTEX_CLUSTER_NAME\" in $CORTEX_REGION"
      exit 1
    elif [ "$cluster_status" == "DELETING" ]; then
      echo "error: your cortex cluster named \"$CORTEX_CLUSTER_NAME\" in $CORTEX_REGION is currently spinning down; please try again once it is completely deleted (may take a few minutes)"
      exit 1
    elif [ "$cluster_status" == "CREATING" ]; then
      echo "error: your cortex cluster named \"$CORTEX_CLUSTER_NAME\" in $CORTEX_REGION is currently spinning up; please try again once it is ready"
      exit 1
    elif [ "$cluster_status" == "UPDATING" ]; then
      echo "error: your cortex cluster named \"$CORTEX_CLUSTER_NAME\" in $CORTEX_REGION is currently updating; please try again once it is ready"
      exit 1
    elif [ "$cluster_status" == "FAILED" ]; then
      echo "error: your cortex cluster named \"$CORTEX_CLUSTER_NAME\" in $CORTEX_REGION is failed; delete it with \`eksctl delete cluster --name=$CORTEX_CLUSTER_NAME --region=$CORTEX_REGION\` and try again"
      exit 1
    else  # cluster exists, but is has an unknown status (unexpected)
      echo "error: there is already a cluster named \"$CORTEX_CLUSTER_NAME\" in $CORTEX_REGION (status: ${cluster_status})"
      exit 1
    fi
  fi

  echo -e "￮ spinning up the cluster (this will take about 25 minutes) ...\n"
  python generate_eks.py $CORTEX_CLUSTER_CONFIG_FILE > /workspace/eks.yaml
  eksctl create cluster --timeout=$EKSCTL_TIMEOUT --install-neuron-plugin=false --install-nvidia-plugin=false -f /workspace/eks.yaml
  echo

  suspend_az_rebalance

  write_kubeconfig
}

# checks that the eks cluster is active and configures kubectl
function check_eks() {
  set +e
  cluster_info=$(eksctl get cluster --name=$CORTEX_CLUSTER_NAME --region=$CORTEX_REGION -o json 2> /dev/null)
  cluster_info_exit_code=$?
  set -e

  # no cluster
  if [ $cluster_info_exit_code -ne 0 ]; then
    echo "error: there is no cluster named \"$CORTEX_CLUSTER_NAME\" in $CORTEX_REGION; please update your configuration to point to an existing cortex cluster or create a cortex cluster with \`cortex cluster up\`"
    exit 1
  fi

  set +e
  # cluster statuses: https://github.com/aws/aws-sdk-go/blob/master/service/eks/api.go#L6883
  cluster_status=$(echo "$cluster_info" | jq -r 'first | .Status')
  set -e

  if [ "$cluster_status" == "DELETING" ]; then
    echo "error: your cortex cluster named \"$CORTEX_CLUSTER_NAME\" in $CORTEX_REGION is currently spinning down; please try again once it is completely deleted (may take a few minutes)"
    exit 1
  elif [ "$cluster_status" == "CREATING" ]; then
    echo "error: your cortex cluster named \"$CORTEX_CLUSTER_NAME\" in $CORTEX_REGION is currently spinning up; please try again once it is ready"
    exit 1
  elif [ "$cluster_status" == "UPDATING" ]; then
    echo "error: your cortex cluster named \"$CORTEX_CLUSTER_NAME\" in $CORTEX_REGION is currently updating; please try again once it is ready"
    exit 1
  elif [ "$cluster_status" == "FAILED" ]; then
    echo "error: your cortex cluster named \"$CORTEX_CLUSTER_NAME\" in $CORTEX_REGION is failed; delete it with \`eksctl delete cluster --name=$CORTEX_CLUSTER_NAME --region=$CORTEX_REGION\` and try again"
    exit 1
  fi

  # cluster status is ACTIVE or unknown (in which case we'll assume things are ok instead of erroring)

  write_kubeconfig
}

function write_kubeconfig() {
  eksctl utils write-kubeconfig --cluster=$CORTEX_CLUSTER_NAME --region=$CORTEX_REGION | (grep -v "saved kubeconfig as" | grep -v "using region" | grep -v "eksctl version" || true)
  out=$(kubectl get pods 2>&1 || true); if [[ "$out" == *"must be logged in to the server"* ]]; then echo "error: your aws iam user does not have access to this cluster; to grant access, see https://docs.cortex.dev/v/${CORTEX_VERSION_MINOR}/"; exit 1; fi
}

function setup_configmap() {
  kubectl -n=default create configmap 'cluster-config' \
    --from-file='cluster.yaml'=$CORTEX_CLUSTER_CONFIG_FILE \
    -o yaml --dry-run=client | kubectl apply -f - >/dev/null

  kubectl -n=default create configmap 'env-vars' \
    --from-literal='CORTEX_VERSION'=$CORTEX_VERSION \
    --from-literal='CORTEX_PROVIDER'=$CORTEX_PROVIDER \
    --from-literal='CORTEX_REGION'=$CORTEX_REGION \
    --from-literal='AWS_REGION'=$CORTEX_REGION \
    --from-literal='CORTEX_TELEMETRY_DISABLE'=$CORTEX_TELEMETRY_DISABLE \
    --from-literal='CORTEX_TELEMETRY_SENTRY_DSN'=$CORTEX_TELEMETRY_SENTRY_DSN \
    --from-literal='CORTEX_TELEMETRY_SEGMENT_WRITE_KEY'=$CORTEX_TELEMETRY_SEGMENT_WRITE_KEY \
    --from-literal='CORTEX_DEV_DEFAULT_PREDICTOR_IMAGE_REGISTRY'=$CORTEX_DEV_DEFAULT_PREDICTOR_IMAGE_REGISTRY \
    -o yaml --dry-run=client | kubectl apply -f - >/dev/null
}

function setup_configmap_gcp() {
  kubectl -n=default create configmap 'cluster-config' \
    --from-file='cluster.yaml'=$CORTEX_CLUSTER_CONFIG_FILE \
    -o yaml --dry-run=client | kubectl apply -f - >/dev/null

  kubectl -n=default create configmap 'env-vars' \
    --from-literal='CORTEX_VERSION'=$CORTEX_VERSION \
    --from-literal='CORTEX_PROVIDER'=$CORTEX_PROVIDER \
    --from-literal='CORTEX_GCP_PROJECT'=$CORTEX_GCP_PROJECT \
    --from-literal='CORTEX_GCP_ZONE'=$CORTEX_GCP_ZONE \
    --from-literal='CORTEX_TELEMETRY_DISABLE'=$CORTEX_TELEMETRY_DISABLE \
    --from-literal='CORTEX_TELEMETRY_SENTRY_DSN'=$CORTEX_TELEMETRY_SENTRY_DSN \
    --from-literal='CORTEX_TELEMETRY_SEGMENT_WRITE_KEY'=$CORTEX_TELEMETRY_SEGMENT_WRITE_KEY \
    --from-literal='CORTEX_DEV_DEFAULT_PREDICTOR_IMAGE_REGISTRY'=$CORTEX_DEV_DEFAULT_PREDICTOR_IMAGE_REGISTRY \
    --from-literal='GOOGLE_APPLICATION_CREDENTIALS'='/var/secrets/google/key.json' \
    -o yaml --dry-run=client | kubectl apply -f - >/dev/null
}

function setup_secrets() {
  kubectl -n=default create secret generic 'aws-credentials' \
    --from-literal='AWS_ACCESS_KEY_ID'=$CLUSTER_AWS_ACCESS_KEY_ID \
    --from-literal='AWS_SECRET_ACCESS_KEY'=$CLUSTER_AWS_SECRET_ACCESS_KEY \
    -o yaml --dry-run=client | kubectl apply -f - >/dev/null
}

function setup_prometheus() {
  envsubst < manifests/prometheus-operator.yaml | kubectl apply -f - >/dev/null
  envsubst < manifests/prometheus-statsd-exporter.yaml | kubectl apply -f - >/dev/null
  python render_template.py $CORTEX_CLUSTER_CONFIG_FILE manifests/prometheus-monitoring.yaml.j2 | kubectl apply -f - >/dev/null
}

function setup_grafana() {
  kubectl apply -f manifests/grafana/grafana-dashboard-realtime.yaml >/dev/null
  kubectl apply -f manifests/grafana/grafana-dashboard-batch.yaml >/dev/null
  envsubst < manifests/grafana/grafana.yaml | kubectl apply -f - >/dev/null
}

function setup_secrets_gcp() {
  kubectl create secret generic 'gcp-credentials' --from-file=key.json=$GOOGLE_APPLICATION_CREDENTIALS >/dev/null
}

function restart_operator() {
  echo -n "￮ starting operator "
  kubectl -n=default delete --ignore-not-found=true --grace-period=10 deployment operator >/dev/null 2>&1
  printed_dot="false"
  until [ "$(kubectl -n=default get pods -l workloadID=operator -o json | jq -j '.items | length')" -eq "0" ]; do echo -n "."; printed_dot="true"; sleep 2; done
  python render_template.py $CORTEX_CLUSTER_CONFIG_FILE manifests/operator.yaml.j2 > /workspace/operator.yaml
  kubectl apply -f /workspace/operator.yaml >/dev/null
  if [ "$printed_dot" == "true" ]; then echo " ✓"; else echo "✓"; fi
}

function resize_nodegroup() {
  # check for change in min/max instances
  asg_on_demand_info=$(aws autoscaling describe-auto-scaling-groups --region $CORTEX_REGION --query "AutoScalingGroups[?contains(Tags[?Key==\`alpha.eksctl.io/cluster-name\`].Value, \`$CORTEX_CLUSTER_NAME\`)]|[?contains(Tags[?Key==\`alpha.eksctl.io/nodegroup-name\`].Value, \`ng-cortex-worker-on-demand\`)]")
  asg_on_demand_length=$(echo "$asg_on_demand_info" | jq -r 'length')
  asg_on_demand_name=""
  if (( "$asg_on_demand_length" > "0" )); then
    asg_on_demand_name=$(echo "$asg_on_demand_info" | jq -r 'first | .AutoScalingGroupName')
  fi

  asg_spot_info=$(aws autoscaling describe-auto-scaling-groups --region $CORTEX_REGION --query "AutoScalingGroups[?contains(Tags[?Key==\`alpha.eksctl.io/cluster-name\`].Value, \`$CORTEX_CLUSTER_NAME\`)]|[?contains(Tags[?Key==\`alpha.eksctl.io/nodegroup-name\`].Value, \`ng-cortex-worker-spot\`)]")
  asg_spot_length=$(echo "$asg_spot_info" | jq -r 'length')
  asg_spot_name=""
  if (( "$asg_spot_length" > "0" )); then
    asg_spot_name=$(echo "$asg_spot_info" | jq -r 'first | .AutoScalingGroupName')
  fi

  if [[ -z "$asg_spot_name" ]] && [[ -z "$asg_on_demand_name" ]]; then
    echo "error: unable to find valid autoscaling groups"
    exit 1
  fi

  if [[ -z $asg_spot_name ]]; then
    asg_min_size=$(echo "$asg_on_demand_info" | jq -r 'first | .MinSize')
    asg_max_size=$(echo "$asg_on_demand_info" | jq -r 'first | .MaxSize')
    if [ "$asg_min_size" = "" ] || [ "$asg_min_size" = "null" ] || [ "$asg_max_size" = "" ] || [ "$asg_max_size" = "null" ]; then
      echo -e "unable to find on-demand autoscaling group size from info:\n$asg_on_demand_info"
      exit 1
    fi
  else
    asg_min_size=$(echo "$asg_spot_info" | jq -r 'first | .MinSize')
    asg_max_size=$(echo "$asg_spot_info" | jq -r 'first | .MaxSize')
    if [ "$asg_min_size" = "" ] || [ "$asg_min_size" = "null" ] || [ "$asg_max_size" = "" ] || [ "$asg_max_size" = "null" ]; then
      echo -e "unable to find spot autoscaling group size from info:\n$asg_spot_info"
      exit 1
    fi
  fi

  asg_on_demand_resize_flags=""
  asg_spot_resize_flags=""

  if [ "$asg_min_size" != "$CORTEX_MIN_INSTANCES" ]; then
    # only update min for on-demand nodegroup if it's not a backup
    if [[ -n $asg_on_demand_name ]] && [[ "$CORTEX_SPOT_CONFIG_ON_DEMAND_BACKUP" != "True" ]]; then
      asg_on_demand_resize_flags+=" --min-size=$CORTEX_MIN_INSTANCES"
    fi
    if [[ -n $asg_spot_name ]]; then
      asg_spot_resize_flags+=" --min-size=$CORTEX_MIN_INSTANCES"
    fi
  fi

  if [ "$asg_max_size" != "$CORTEX_MAX_INSTANCES" ]; then
    if [[ -n $asg_on_demand_name ]]; then
      asg_on_demand_resize_flags+=" --max-size=$CORTEX_MAX_INSTANCES"
    fi
    if [[ -n $asg_spot_name ]]; then
      asg_spot_resize_flags+=" --max-size=$CORTEX_MAX_INSTANCES"
    fi
  fi

  is_resizing="false"
  if [ "$asg_min_size" != "$CORTEX_MIN_INSTANCES" ] && [ "$asg_max_size" != "$CORTEX_MAX_INSTANCES" ]; then
    echo -n "￮ updating min instances to $CORTEX_MIN_INSTANCES and max instances to $CORTEX_MAX_INSTANCES "
    is_resizing="true"
  elif [ "$asg_min_size" != "$CORTEX_MIN_INSTANCES" ]; then
    echo -n "￮ updating min instances to $CORTEX_MIN_INSTANCES "
    is_resizing="true"
  elif [ "$asg_max_size" != "$CORTEX_MAX_INSTANCES" ]; then
    echo -n "￮ updating max instances to $CORTEX_MAX_INSTANCES "
    is_resizing="true"
  fi

  if [ "$asg_on_demand_resize_flags" != "" ]; then
    aws autoscaling update-auto-scaling-group --region $CORTEX_REGION --auto-scaling-group-name $asg_on_demand_name $asg_on_demand_resize_flags
  fi
  if [ "$asg_spot_resize_flags" != "" ]; then
    aws autoscaling update-auto-scaling-group --region $CORTEX_REGION --auto-scaling-group-name $asg_spot_name $asg_spot_resize_flags
  fi

  if [ "$is_resizing" == "true" ]; then
    echo "✓"
  fi
}

function suspend_az_rebalance() {
  asg_on_demand_info=$(aws autoscaling describe-auto-scaling-groups --region $CORTEX_REGION --query "AutoScalingGroups[?contains(Tags[?Key==\`alpha.eksctl.io/cluster-name\`].Value, \`$CORTEX_CLUSTER_NAME\`)]|[?contains(Tags[?Key==\`alpha.eksctl.io/nodegroup-name\`].Value, \`ng-cortex-worker-on-demand\`)]")
  asg_on_demand_length=$(echo "$asg_on_demand_info" | jq -r 'length')
  if (( "$asg_on_demand_length" > "0" )); then
    asg_on_demand_name=$(echo "$asg_on_demand_info" | jq -r 'first | .AutoScalingGroupName')
    aws autoscaling suspend-processes --region $CORTEX_REGION --auto-scaling-group-name $asg_on_demand_name --scaling-processes AZRebalance
  fi

  asg_spot_info=$(aws autoscaling describe-auto-scaling-groups --region $CORTEX_REGION --query "AutoScalingGroups[?contains(Tags[?Key==\`alpha.eksctl.io/cluster-name\`].Value, \`$CORTEX_CLUSTER_NAME\`)]|[?contains(Tags[?Key==\`alpha.eksctl.io/nodegroup-name\`].Value, \`ng-cortex-worker-spot\`)]")
  asg_spot_length=$(echo "$asg_spot_info" | jq -r 'length')
  if (( "$asg_spot_length" > "0" )); then
    asg_spot_name=$(echo "$asg_spot_info" | jq -r 'first | .AutoScalingGroupName')
    aws autoscaling suspend-processes --region $CORTEX_REGION --auto-scaling-group-name $asg_spot_name --scaling-processes AZRebalance
  fi
}

function setup_istio() {
  envsubst < manifests/istio-namespace.yaml | kubectl apply -f - >/dev/null

  if ! grep -q "istio-customgateway-certs" <<< $(kubectl get secret -n istio-system); then
    WEBSITE=localhost
    openssl req -subj "/C=US/CN=$WEBSITE" -newkey rsa:2048 -nodes -keyout $WEBSITE.key -x509 -days 3650 -out $WEBSITE.crt >/dev/null 2>&1
    kubectl create -n istio-system secret tls istio-customgateway-certs --key $WEBSITE.key --cert $WEBSITE.crt >/dev/null
  fi

  python render_template.py $CORTEX_CLUSTER_CONFIG_FILE manifests/istio.yaml.j2 > /workspace/istio.yaml
  output_if_error istio-${ISTIO_VERSION}/bin/istioctl install -f /workspace/istio.yaml
}

function start_pre_download_images() {
  registry="quay.io/cortexlabs"
  if [ -n "$CORTEX_DEV_DEFAULT_PREDICTOR_IMAGE_REGISTRY" ]; then
    registry="$CORTEX_DEV_DEFAULT_PREDICTOR_IMAGE_REGISTRY"
  fi
  export CORTEX_IMAGE_PYTHON_PREDICTOR_CPU="${registry}/python-predictor-cpu:${CORTEX_VERSION}"
  export CORTEX_IMAGE_PYTHON_PREDICTOR_GPU="${registry}/python-predictor-gpu:${CORTEX_VERSION}-cuda10.2-cudnn8"
  export CORTEX_IMAGE_PYTHON_PREDICTOR_INF="${registry}/python-predictor-inf:${CORTEX_VERSION}"
  export CORTEX_IMAGE_TENSORFLOW_SERVING_CPU="${registry}/tensorflow-serving-cpu:${CORTEX_VERSION}"
  export CORTEX_IMAGE_TENSORFLOW_SERVING_GPU="${registry}/tensorflow-serving-gpu:${CORTEX_VERSION}"
  export CORTEX_IMAGE_TENSORFLOW_SERVING_INF="${registry}/tensorflow-serving-inf:${CORTEX_VERSION}"
  export CORTEX_IMAGE_TENSORFLOW_PREDICTOR="${registry}/tensorflow-predictor:${CORTEX_VERSION}"

  if [[ "$CORTEX_INSTANCE_TYPE" == p* ]] || [[ "$CORTEX_INSTANCE_TYPE" == g* ]] || [ -n "$CORTEX_ACCELERATOR_TYPE" ]; then
    envsubst < manifests/image-downloader-gpu.yaml | kubectl apply -f - &>/dev/null
  elif [[ "$CORTEX_INSTANCE_TYPE" == inf* ]]; then
    envsubst < manifests/image-downloader-inf.yaml | kubectl apply -f - &>/dev/null
  else
    envsubst < manifests/image-downloader-cpu.yaml | kubectl apply -f - &>/dev/null
  fi
}

function await_pre_download_images() {
  if kubectl get daemonset image-downloader -n=default &>/dev/null; then
    echo -n "￮ downloading docker images "
    printed_dot="false"
    i=0
    until [ "$(kubectl get daemonset image-downloader -n=default -o 'jsonpath={.status.numberReady}')" == "$(kubectl get daemonset image-downloader -n=default -o 'jsonpath={.status.desiredNumberScheduled}')" ]; do
      if [ $i -eq 120 ]; then break; fi  # give up after 6 minutes
      echo -n "."
      printed_dot="true"
      ((i=i+1))
      sleep 3
    done
    kubectl -n=default delete --ignore-not-found=true daemonset image-downloader &>/dev/null
    if [ "$printed_dot" == "true" ]; then echo " ✓"; else echo "✓"; fi
  fi
}

function validate_cortex() {
  set +e

  validation_start_time="$(date +%s)"

  echo -n "￮ waiting for load balancers "

  operator_pod_name=""
  operator_pod_is_ready=""
  operator_pod_status=""
  operator_endpoint=""
  api_load_balancer_endpoint=""
  operator_load_balancer_state=""
  api_load_balancer_state=""
  operator_target_group_status=""
  operator_endpoint_reachable=""
  success_cycles=0

  while true; do
    # 30 minute timeout
    now="$(date +%s)"
    if [ "$now" -ge "$(($validation_start_time+1800))" ]; then
      echo -e "\n\ntimeout has occurred when validating your cortex cluster"
      echo -e "\ndebugging info:"
      if [ "$operator_pod_name" != "" ]; then
        echo "operator pod name: $operator_pod_name"
      fi
      if [ "$operator_pod_is_ready" != "" ]; then
        echo "operator pod is ready: $operator_pod_is_ready"
      fi
      if [ "$operator_pod_status" != "" ]; then
        echo "operator pod status: $operator_pod_status"
      fi
      if [ "$operator_endpoint" != "" ]; then
        echo "operator endpoint: $operator_endpoint"
      fi
      if [ "$api_load_balancer_endpoint" != "" ]; then
        echo "api load balancer endpoint: $api_load_balancer_endpoint"
      fi
      if [ "$operator_load_balancer_state" != "" ]; then
        echo "operator load balancer state: $operator_load_balancer_state"
      fi
      if [ "$api_load_balancer_state" != "" ]; then
        echo "api load balancer state: $api_load_balancer_state"
      fi
      if [ "$operator_target_group_status" != "" ]; then
        echo "operator target group status: $operator_target_group_status"
      fi
      if [ "$CORTEX_OPERATOR_LOAD_BALANCER_SCHEME" == "internet-facing" ] && [ "$operator_endpoint_reachable" != "" ]; then
        echo "operator endpoint reachable: $operator_endpoint_reachable"
      fi
      if [ "$operator_endpoint" != "" ]; then
        echo "operator curl response:"
        curl --max-time 3 "${operator_endpoint}/verifycortex"
      fi
      echo
      exit 1
    fi

    echo -n "."
    sleep 5

    operator_pod_name=$(kubectl -n=default get pods -o=name --sort-by=.metadata.creationTimestamp | (grep "^pod/operator-" || true) | tail -1)
    if [ "$operator_pod_name" == "" ]; then
      success_cycles=0
      continue
    fi

    operator_pod_is_ready=$(kubectl -n=default get "$operator_pod_name" -o jsonpath='{.status.containerStatuses[0].ready}')
    if [ "$operator_pod_is_ready" != "true" ]; then
      operator_pod_status=$(kubectl -n=default get "$operator_pod_name" -o jsonpath='{.status.containerStatuses[0]}')
      if [[ "$operator_pod_status" == *"ImagePullBackOff"* ]]; then
        echo -e "\nerror: the operator image you specified could not be pulled:"
        echo $operator_pod_status
        echo
        exit 1
      fi

      num_restarts=$(kubectl -n=default get "$operator_pod_name" -o jsonpath='{.status.containerStatuses[0].restartCount}')
      if [[ $num_restarts -ge 2 ]]; then
        echo -e "\n\nan error occurred when starting the cortex operator"
        echo -e "\noperator logs (currently running container):\n"
        kubectl -n=default logs "$operator_pod_name"
        echo -e "\noperator logs (previous container):\n"
        kubectl -n=default logs "$operator_pod_name" --previous
        echo
        exit 1
      fi

      success_cycles=0
      continue
    fi
    operator_pod_status=""  # reset operator_pod_status since now the operator is active

    if [ "$operator_endpoint" == "" ]; then
      out=$(kubectl -n=istio-system get service ingressgateway-operator -o json | tr -d '[:space:]')
      if [[ $out != *'"loadBalancer":{"ingress":[{"'* ]]; then
        success_cycles=0
        continue
      fi
      operator_endpoint=$(kubectl -n=istio-system get service ingressgateway-operator -o json | tr -d '[:space:]' | sed 's/.*{\"hostname\":\"\(.*\)\".*/\1/')
    fi

    if [ "$api_load_balancer_endpoint" == "" ]; then
      out=$(kubectl -n=istio-system get service ingressgateway-apis -o json | tr -d '[:space:]')
      if [[ $out != *'"loadBalancer":{"ingress":[{"'* ]]; then
        success_cycles=0
        continue
      fi
      api_load_balancer_endpoint=$(kubectl -n=istio-system get service ingressgateway-apis -o json | tr -d '[:space:]' | sed 's/.*{\"hostname\":\"\(.*\)\".*/\1/')
    fi

    operator_load_balancer_state="$(python get_operator_load_balancer_state.py)"  # don't cache this result
    if [ "$operator_load_balancer_state" != "active" ]; then
      success_cycles=0
      continue
    fi

    api_load_balancer_state="$(python get_api_load_balancer_state.py)"  # don't cache this result
    if [ "$api_load_balancer_state" != "active" ]; then
      success_cycles=0
      continue
    fi

    operator_target_group_status="$(python get_operator_target_group_status.py)"  # don't cache this result
    if [ "$operator_target_group_status" != "healthy" ]; then
      success_cycles=0
      continue
    fi

    if [ "$CORTEX_OPERATOR_LOAD_BALANCER_SCHEME" == "internet-facing" ]; then
      operator_endpoint_reachable="false"  # don't cache this result
      if ! curl --max-time 3 "${operator_endpoint}/verifycortex" >/dev/null 2>&1; then
        success_cycles=0
        continue
      fi
      operator_endpoint_reachable="true"
    fi

    if [[ $success_cycles -lt 5 ]]; then
      ((success_cycles++))
      continue
    fi

    break
  done

  echo " ✓"
}

function validate_cortex_gcp() {
  set +e

  validation_start_time="$(date +%s)"

  echo -n "￮ waiting for load balancers "

  operator_pod_name=""
  operator_pod_is_ready=""
  operator_pod_status=""
  operator_endpoint=""
  api_load_balancer_endpoint=""
  operator_endpoint_reachable=""
  success_cycles=0

  while true; do
    # 30 minute timeout
    now="$(date +%s)"
    if [ "$now" -ge "$(($validation_start_time+1800))" ]; then
      echo -e "\n\ntimeout has occurred when validating your cortex cluster"
      echo -e "\ndebugging info:"
      if [ "$operator_pod_name" != "" ]; then
        echo "operator pod name: $operator_pod_name"
      fi
      if [ "$operator_pod_is_ready" != "" ]; then
        echo "operator pod is ready: $operator_pod_is_ready"
      fi
      if [ "$operator_pod_status" != "" ]; then
        echo "operator pod status: $operator_pod_status"
      fi
      if [ "$operator_endpoint" != "" ]; then
        echo "operator endpoint: $operator_endpoint"
      fi
      if [ "$api_load_balancer_endpoint" != "" ]; then
        echo "api load balancer endpoint: $api_load_balancer_endpoint"
      fi
      if [ "$operator_endpoint_reachable" != "" ]; then
        echo "operator endpoint reachable: $operator_endpoint_reachable"
      fi
      if [ "$operator_endpoint" != "" ]; then
        echo "operator curl response:"
        curl --max-time 3 "${operator_endpoint}/verifycortex"
      fi
      echo
      exit 1
    fi

    echo -n "."
    sleep 5

    operator_pod_name=$(kubectl -n=default get pods -o=name --sort-by=.metadata.creationTimestamp | (grep "^pod/operator-" || true) | tail -1)
    if [ "$operator_pod_name" == "" ]; then
      success_cycles=0
      continue
    fi

    operator_pod_is_ready=$(kubectl -n=default get "$operator_pod_name" -o jsonpath='{.status.containerStatuses[0].ready}')
    if [ "$operator_pod_is_ready" != "true" ]; then
      operator_pod_status=$(kubectl -n=default get "$operator_pod_name" -o jsonpath='{.status.containerStatuses[0]}')
      if [[ "$operator_pod_status" == *"ImagePullBackOff"* ]]; then
        echo -e "\nerror: the operator image you specified could not be pulled:"
        echo $operator_pod_status
        echo
        exit 1
      fi

      num_restarts=$(kubectl -n=default get "$operator_pod_name" -o jsonpath='{.status.containerStatuses[0].restartCount}')
      if [[ $num_restarts -ge 2 ]]; then
        echo -e "\n\nan error occurred when starting the cortex operator"
        echo -e "\noperator logs (currently running container):\n"
        kubectl -n=default logs "$operator_pod_name"
        echo -e "\noperator logs (previous container):\n"
        kubectl -n=default logs "$operator_pod_name" --previous
        echo
        exit 1
      fi

      success_cycles=0
      continue
    fi
    operator_pod_status=""  # reset operator_pod_status since now the operator is active

    if [ "$operator_endpoint" == "" ]; then
      out=$(kubectl -n=istio-system get service ingressgateway-operator -o json | tr -d '[:space:]')
      if [[ $out != *'"loadBalancer":{"ingress":[{"'* ]]; then
        success_cycles=0
        continue
      fi
      operator_endpoint=$(kubectl -n=istio-system get service ingressgateway-operator -o json | tr -d '[:space:]' | sed 's/.*{\"ip\":\"\(.*\)\".*/\1/')
    fi

    if [ "$api_load_balancer_endpoint" == "" ]; then
      out=$(kubectl -n=istio-system get service ingressgateway-apis -o json | tr -d '[:space:]')
      if [[ $out != *'"loadBalancer":{"ingress":[{"'* ]]; then
        success_cycles=0
        continue
      fi
      api_load_balancer_endpoint=$(kubectl -n=istio-system get service ingressgateway-apis -o json | tr -d '[:space:]' | sed 's/.*{\"ip\":\"\(.*\)\".*/\1/')
    fi

    if [ "$CORTEX_OPERATOR_LOAD_BALANCER_SCHEME" == "internet-facing" ]; then
      operator_endpoint_reachable="false"  # don't cache this result
      if ! curl --max-time 3 "${operator_endpoint}/verifycortex" >/dev/null 2>&1; then
        success_cycles=0
        continue
      fi
      operator_endpoint_reachable="true"
    fi

    if [[ $success_cycles -lt 1 ]]; then
      ((success_cycles++))
      continue
    fi

    break
  done

  echo " ✓"
}

function print_endpoints_aws() {
  echo ""

  operator_endpoint=$(get_operator_endpoint_aws)
  api_load_balancer_endpoint=$(get_api_load_balancer_endpoint_aws)

  echo "operator:          $operator_endpoint"  # before modifying this, search for this prefix
  echo "api load balancer: $api_load_balancer_endpoint"
}

function get_operator_endpoint_aws() {
  kubectl -n=istio-system get service ingressgateway-operator -o json | tr -d '[:space:]' | sed 's/.*{\"hostname\":\"\(.*\)\".*/\1/'
}

function get_api_load_balancer_endpoint_aws() {
  kubectl -n=istio-system get service ingressgateway-apis -o json | tr -d '[:space:]' | sed 's/.*{\"hostname\":\"\(.*\)\".*/\1/'
}

function print_endpoints_gcp() {
  echo ""

  operator_endpoint=$(get_operator_endpoint_gcp)
  api_load_balancer_endpoint=$(get_api_load_balancer_endpoint_gcp)

  echo "operator:          $operator_endpoint"  # before modifying this, search for this prefix
  echo "api load balancer: $api_load_balancer_endpoint"
}

function get_operator_endpoint_gcp() {
  kubectl -n=istio-system get service ingressgateway-operator -o json | tr -d '[:space:]' | sed 's/.*{\"ip\":\"\(.*\)\".*/\1/'
}

function get_api_load_balancer_endpoint_gcp() {
  kubectl -n=istio-system get service ingressgateway-apis -o json | tr -d '[:space:]' | sed 's/.*{\"ip\":\"\(.*\)\".*/\1/'
}

function output_if_error() {
  set +e
  rm --force /tmp/suppress.out 2> /dev/null
  ${1+"$@"} > /tmp/suppress.out 2>&1
  if [ "$?" != "0" ]; then
    echo
    cat /tmp/suppress.out
    exit 1
  fi
  rm --force /tmp/suppress.out 2> /dev/null
  set -e
}

main
