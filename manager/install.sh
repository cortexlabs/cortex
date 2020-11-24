#!/bin/bash

# Copyright 2020 Cortex Labs, Inc.
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
  create_eks

  start_pre_download_images

  if [ "$CORTEX_API_LOAD_BALANCER_SCHEME" == "internal" ] && [ "$CORTEX_API_GATEWAY" == "public" ]; then
    create_vpc_link
  fi

  echo -n "￮ updating cluster configuration "
  setup_configmap
  setup_secrets
  echo "✓"

  echo -n "￮ configuring networking "
  setup_istio
  python render_template.py $CORTEX_CLUSTER_CONFIG_FILE manifests/apis.yaml.j2 > /workspace/apis.yaml
  kubectl apply -f /workspace/apis.yaml >/dev/null
  echo "✓"

  echo -n "￮ configuring autoscaling "
  python render_template.py $CORTEX_CLUSTER_CONFIG_FILE manifests/cluster-autoscaler.yaml.j2 > /workspace/cluster-autoscaler.yaml
  kubectl apply -f /workspace/cluster-autoscaler.yaml >/dev/null
  echo "✓"

  echo -n "￮ configuring logging "
  envsubst < manifests/fluentd.yaml | kubectl apply -f - >/dev/null
  echo "✓"

  echo -n "￮ configuring metrics "
  envsubst < manifests/metrics-server.yaml | kubectl apply -f - >/dev/null
  envsubst < manifests/statsd.yaml | kubectl apply -f - >/dev/null
  echo "✓"

  if [[ "$CORTEX_INSTANCE_TYPE" == p* ]] || [[ "$CORTEX_INSTANCE_TYPE" == g* ]]; then
    echo -n "￮ configuring gpu support "
    envsubst < manifests/nvidia.yaml | kubectl apply -f - >/dev/null
    echo "✓"
  fi

  if [[ "$CORTEX_INSTANCE_TYPE" == inf* ]]; then
    echo -n "￮ configuring inf support "
    envsubst < manifests/inferentia.yaml | kubectl apply -f - >/dev/null
    echo "✓"
  fi

  if [ "$CORTEX_API_LOAD_BALANCER_SCHEME" == "internal" ] && [ "$CORTEX_API_GATEWAY" == "public" ]; then
    create_vpc_link_integration
  fi

  restart_operator

  validate_cortex

  await_pre_download_images

  echo -e "\ncortex is ready!"
  if [ "$CORTEX_OPERATOR_LOAD_BALANCER_SCHEME" == "internal" ]; then
    echo -e "note: you will need to configure VPC Peering to connect to your cluster: https://docs.cortex.dev/v/${CORTEX_VERSION_MINOR}/aws/vpc-peering"
  fi

  print_endpoints
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

  print_endpoints
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

  echo -e "￮ spinning up the cluster (this will take about 15 minutes) ...\n"
  python generate_eks.py $CORTEX_CLUSTER_CONFIG_FILE > /workspace/eks.yaml
  eksctl create cluster --timeout=$EKSCTL_TIMEOUT --install-neuron-plugin=false -f /workspace/eks.yaml
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
  eksctl utils write-kubeconfig --cluster=$CORTEX_CLUSTER_NAME --region=$CORTEX_REGION | grep -v "saved kubeconfig as" | grep -v "using region" | grep -v "eksctl version" || true
  out=$(kubectl get pods 2>&1 || true); if [[ "$out" == *"must be logged in to the server"* ]]; then echo "error: your aws iam user does not have access to this cluster; to grant access, see https://docs.cortex.dev/v/${CORTEX_VERSION_MINOR}/aws/security#running-cortex-cluster-commands-from-different-iam-users"; exit 1; fi
}

function setup_configmap() {
  kubectl -n=default create configmap 'cluster-config' \
    --from-file='cluster.yaml'=$CORTEX_CLUSTER_CONFIG_FILE \
    -o yaml --dry-run=client | kubectl apply -f - >/dev/null

  kubectl -n=default create configmap 'env-vars' \
    --from-literal='CORTEX_VERSION'=$CORTEX_VERSION \
    --from-literal='CORTEX_REGION'=$CORTEX_REGION \
    --from-literal='AWS_REGION'=$CORTEX_REGION \
    --from-literal='CORTEX_BUCKET'=$CORTEX_BUCKET \
    --from-literal='CORTEX_TELEMETRY_DISABLE'=$CORTEX_TELEMETRY_DISABLE \
    --from-literal='CORTEX_TELEMETRY_SENTRY_DSN'=$CORTEX_TELEMETRY_SENTRY_DSN \
    --from-literal='CORTEX_TELEMETRY_SEGMENT_WRITE_KEY'=$CORTEX_TELEMETRY_SEGMENT_WRITE_KEY \
    --from-literal='CORTEX_DEV_DEFAULT_PREDICTOR_IMAGE_REGISTRY'=$CORTEX_DEV_DEFAULT_PREDICTOR_IMAGE_REGISTRY \
    -o yaml --dry-run=client | kubectl apply -f - >/dev/null
}

function setup_secrets() {
  kubectl -n=default create secret generic 'aws-credentials' \
    --from-literal='AWS_ACCESS_KEY_ID'=$CLUSTER_AWS_ACCESS_KEY_ID \
    --from-literal='AWS_SECRET_ACCESS_KEY'=$CLUSTER_AWS_SECRET_ACCESS_KEY \
    -o yaml --dry-run=client | kubectl apply -f - >/dev/null
}

function restart_operator() {
  echo -n "￮ starting operator "
  kubectl -n=default delete --ignore-not-found=true --grace-period=10 deployment operator >/dev/null 2>&1
  printed_dot="false"
  until [ "$(kubectl -n=default get pods -l workloadID=operator -o json | jq -j '.items | length')" -eq "0" ]; do echo -n "."; printed_dot="true"; sleep 2; done
  envsubst < manifests/operator.yaml | kubectl apply -f - >/dev/null
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

function create_vpc_link() {
  # get VPC ID
  vpc_id=$(aws ec2 describe-vpcs --region $CORTEX_REGION --filters Name=tag:cortex.dev/cluster-name,Values=$CORTEX_CLUSTER_NAME | jq .Vpcs[0].VpcId | tr -d '"')
  if [ "$vpc_id" = "" ] || [ "$vpc_id" = "null" ]; then
    echo "unable to find cortex vpc"
    exit 1
  fi

  # filter all private subnets belonging to cortex cluster
  private_subnets=$(aws ec2 describe-subnets --region $CORTEX_REGION --filters Name=vpc-id,Values=$vpc_id Name=tag:Name,Values=*Private* | jq -s '.[].Subnets[].SubnetId' | tr -d '"')
  if [ "$private_subnets" = "" ] || [ "$private_subnets" = "null" ]; then
    echo "unable to find cortex private subnets"
    exit 1
  fi

  # get default security group for cortex VPC
  default_security_group=$(aws ec2 describe-security-groups --region $CORTEX_REGION --filters Name=vpc-id,Values=$vpc_id Name=group-name,Values=default | jq -c .SecurityGroups[].GroupId | tr -d '"')
  if [ "$default_security_group" = "" ] || [ "$default_security_group" = "null" ]; then
    echo "unable to find cortex default security group"
    exit 1
  fi

  # create VPC Link
  create_vpc_link_output=$(aws apigatewayv2 create-vpc-link --region $CORTEX_REGION --tags "$CORTEX_TAGS_JSON" --name $CORTEX_CLUSTER_NAME --subnet-ids $private_subnets --security-group-ids $default_security_group)
  vpc_link_id=$(echo $create_vpc_link_output | jq .VpcLinkId | tr -d '"')
  if [ "$vpc_link_id" = "" ] || [ "$vpc_link_id" = "null" ]; then
    echo -e "unable to extract vpc link ID from create-vpc-link output:\n$create_vpc_link_output"
    exit 1
  fi
}

# must be called after create_vpc_link() since $vpc_link_id is reused
function create_vpc_link_integration() {
  echo -n "￮ creating api gateway vpc link integration "
  api_id=$(python get_api_gateway_id.py)
  python create_gateway_integration.py $api_id $vpc_link_id
  echo "✓"
  echo -n "￮ waiting for api gateway vpc link integration "
  until [ "$(aws apigatewayv2 get-vpc-link --region $CORTEX_REGION --vpc-link-id $vpc_link_id | jq .VpcLinkStatus | tr -d '"')" = "AVAILABLE" ]; do echo -n "."; sleep 3; done
  echo " ✓"
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
  if [[ "$CORTEX_INSTANCE_TYPE" == p* ]] || [[ "$CORTEX_INSTANCE_TYPE" == g* ]]; then
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
      if [ $i -eq 100 ]; then break; fi  # give up after 5 minutes
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

function print_endpoints() {
  echo ""

  operator_endpoint=$(get_operator_endpoint)
  api_load_balancer_endpoint=$(get_api_load_balancer_endpoint)
  if [ "$CORTEX_API_GATEWAY" == "public" ]; then
    api_gateway_endpoint=$(get_api_gateway_endpoint)
  fi

  echo "operator:          $operator_endpoint"  # before modifying this, search for this prefix
  echo "api load balancer: $api_load_balancer_endpoint"
  if [ "$CORTEX_API_GATEWAY" == "public" ]; then
    echo "api gateway:       $api_gateway_endpoint"
  fi
}

function get_operator_endpoint() {
  kubectl -n=istio-system get service ingressgateway-operator -o json | tr -d '[:space:]' | sed 's/.*{\"hostname\":\"\(.*\)\".*/\1/'
}

function get_api_load_balancer_endpoint() {
  kubectl -n=istio-system get service ingressgateway-apis -o json | tr -d '[:space:]' | sed 's/.*{\"hostname\":\"\(.*\)\".*/\1/'
}

function get_api_gateway_endpoint() {
  python get_api_gateway_endpoint.py
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
