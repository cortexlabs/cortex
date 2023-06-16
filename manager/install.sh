#!/bin/bash

# Copyright 2022 Cortex Labs, Inc.
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
EKSCTL_CLUSTER_TIMEOUT=45m
EKSCTL_NODEGROUP_TIMEOUT=30m
mkdir /workspace

arg1="$1"

function main() {
  if [ "$arg1" = "--configure" ]; then
    cluster_configure
  else
    cluster_up
  fi
}

function cluster_up() {
  create_eks

  echo -n "￮ updating cluster configuration "
  setup_namespaces
  setup_configmap
  echo "✓"

  echo -n "￮ configuring networking (this will take a few minutes) "
  setup_ipvs
  echo "setup_ipvs done. setup_istio starts..."
  setup_istio
  echo "setup_istio done."
  echo "installing ebs csi driver..."
  install_ebs_csi_driver
  echo "installed ebs csi driver"
  python render_template.py $CORTEX_CLUSTER_CONFIG_FILE manifests/apis.yaml.j2 | kubectl apply -f - >/dev/null
  echo "✓"

  echo -n "￮ configuring autoscaling "
  python render_template.py $CORTEX_CLUSTER_CONFIG_FILE manifests/autoscaler.yaml.j2 | kubectl apply -f - >/dev/null
  python render_template.py $CORTEX_CLUSTER_CONFIG_FILE manifests/activator.yaml.j2 | kubectl apply -f - >/dev/null
  python render_template.py $CORTEX_CLUSTER_CONFIG_FILE manifests/cluster-autoscaler.yaml.j2 | kubectl apply -f - >/dev/null
  echo "✓"

  echo -n "￮ configuring async gateway "
  python render_template.py $CORTEX_CLUSTER_CONFIG_FILE manifests/async-gateway.yaml.j2 | kubectl apply -f - >/dev/null
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

  echo -n "￮ configuring gpu support (for nodegroups that may require it) "
  envsubst < manifests/nvidia.yaml | kubectl apply -f - >/dev/null
  NVIDIA_COM_GPU_VALUE=true envsubst < manifests/prometheus-dcgm-exporter.yaml | kubectl apply -f - >/dev/null
  echo "✓"

  echo -n "￮ configuring inf support (for nodegroups that may require it) "
  envsubst < manifests/inferentia.yaml | kubectl apply -f - >/dev/null
  echo "✓"

  restart_operator
  start_controller_manager

  validate_cortex

  echo -e "\ncortex is ready!"
  if [ "$CORTEX_OPERATOR_LOAD_BALANCER_SCHEME" == "internal" ]; then
    echo -e "\nnote: you will need to configure VPC Peering to connect to your cluster: https://docs.cortexlabs.com/v/${CORTEX_VERSION_MINOR}/"
  fi

  print_endpoints
}

function cluster_configure() {
  check_eks

  resize_nodegroups
  add_nodegroups
  remove_nodegroups

  update_networking

  echo -n "￮ updating cluster configuration "
  setup_configmap
  echo "✓"

  # this is necessary since max_instances may have been updated
  echo -n "￮ configuring autoscaling "
  python render_template.py $CORTEX_CLUSTER_CONFIG_FILE manifests/cluster-autoscaler.yaml.j2 | kubectl apply -f - >/dev/null
  echo "✓"

  restart_controller_manager

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
      echo "error: your cortex cluster named \"$CORTEX_CLUSTER_NAME\" in $CORTEX_REGION is failed; delete it with \`eksctl delete cluster --name=$CORTEX_CLUSTER_NAME --region=$CORTEX_REGION --disable-nodegroup-eviction\` and try again"
      exit 1
    else  # cluster exists, but is has an unknown status (unexpected)
      echo "error: there is already a cluster named \"$CORTEX_CLUSTER_NAME\" in $CORTEX_REGION (status: ${cluster_status})"
      exit 1
    fi
  fi

  echo -e "￮ spinning up the cluster (this will take about 30 minutes) ...\n"
  python generate_eks.py $CORTEX_CLUSTER_CONFIG_FILE manifests/ami.json > /workspace/eks.yaml
  eksctl create cluster --timeout=$EKSCTL_CLUSTER_TIMEOUT --install-neuron-plugin=false --install-nvidia-plugin=false -f /workspace/eks.yaml
  echo

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
    echo "error: your cortex cluster named \"$CORTEX_CLUSTER_NAME\" in $CORTEX_REGION is failed; delete it with \`eksctl delete cluster --name=$CORTEX_CLUSTER_NAME --region=$CORTEX_REGION --disable-nodegroup-eviction\` and try again"
    exit 1
  fi

  # cluster status is ACTIVE or unknown (in which case we'll assume things are ok instead of erroring)

  write_kubeconfig
}

function write_kubeconfig() {
  eksctl utils write-kubeconfig --cluster=$CORTEX_CLUSTER_NAME --region=$CORTEX_REGION --verbose=0 | (grep -v "saved kubeconfig as" || true)
  out=$(kubectl get pods 2>&1 || true); if [[ "$out" == *"must be logged in to the server"* ]]; then echo "error: your aws iam user does not have access to this cluster; to grant access, see https://docs.cortexlabs.com/v/${CORTEX_VERSION_MINOR}/"; exit 1; fi
}

function setup_namespaces() {
  # doing a patch to prevent getting the kubectl.kubernetes.io/last-applied-configuration annotation warning
  kubectl patch namespace default -p '{"metadata": {"labels": {"istio-discovery": "enabled"}}}' >/dev/null
  kubectl apply -f manifests/namespaces.yaml >/dev/null
}

function setup_configmap() {
  envsubst < manifests/default_cortex_cli_config.yaml > tmp_cli_config.yaml
  kubectl -n=default create configmap 'client-config' \
    --from-file='cli.yaml'=tmp_cli_config.yaml \
    -o yaml --dry-run=client | kubectl apply -f - >/dev/null
  rm tmp_cli_config.yaml

  kubectl -n=default create configmap 'cluster-config' \
    --from-file='cluster.yaml'=$CORTEX_CLUSTER_CONFIG_FILE \
    -o yaml --dry-run=client | kubectl apply -f - >/dev/null

  kubectl -n=default create configmap 'env-vars' \
    --from-literal='CORTEX_VERSION'=$CORTEX_VERSION \
    --from-literal='CORTEX_REGION'=$CORTEX_REGION \
    --from-literal='AWS_DEFAULT_REGION'=$CORTEX_REGION \
    --from-literal='AWS_RETRY_MODE'="standard" \
    --from-literal='AWS_MAX_ATTEMPTS'="5" \
    --from-literal='CORTEX_TELEMETRY_DISABLE'=$CORTEX_TELEMETRY_DISABLE \
    --from-literal='CORTEX_TELEMETRY_SENTRY_DSN'=$CORTEX_TELEMETRY_SENTRY_DSN \
    --from-literal='CORTEX_TELEMETRY_SEGMENT_WRITE_KEY'=$CORTEX_TELEMETRY_SEGMENT_WRITE_KEY \
    --from-literal='CORTEX_DEV_DEFAULT_IMAGE_REGISTRY'=$CORTEX_DEV_DEFAULT_IMAGE_REGISTRY \
    -o yaml --dry-run=client | kubectl apply -f - >/dev/null
}

function setup_prometheus() {
  envsubst < manifests/prometheus-operator.yaml | kubectl apply --server-side -f - >/dev/null
  envsubst < manifests/prometheus-statsd-exporter.yaml | kubectl apply -f - >/dev/null
  envsubst < manifests/prometheus-kubelet-exporter.yaml | kubectl apply -f - >/dev/null
  envsubst < manifests/prometheus-kube-state-metrics.yaml | kubectl apply -f - >/dev/null
  envsubst < manifests/prometheus-node-exporter.yaml | kubectl apply -f - >/dev/null
  envsubst < manifests/prometheus-monitoring.yaml | kubectl apply -f - >/dev/null
  python render_template.py $CORTEX_CLUSTER_CONFIG_FILE manifests/prometheus-additional-scrape-configs.yaml.j2 > prometheus-additional-scrape-configs.yaml
  if ! kubectl get secret -n prometheus additional-scrape-configs >/dev/null 2>&1; then
    kubectl create secret generic -n prometheus additional-scrape-configs --from-file=prometheus-additional-scrape-configs.yaml > /dev/null
  fi
}

function setup_grafana() {
  kubectl apply -f manifests/grafana/grafana-dashboard-realtime.yaml >/dev/null
  kubectl apply -f manifests/grafana/grafana-dashboard-async.yaml >/dev/null
  kubectl apply -f manifests/grafana/grafana-dashboard-batch.yaml >/dev/null
  kubectl apply -f manifests/grafana/grafana-dashboard-task.yaml >/dev/null
  kubectl apply -f manifests/grafana/grafana-dashboard-cluster.yaml >/dev/null
  kubectl apply -f manifests/grafana/grafana-dashboard-nodes.yaml >/dev/null
  if [ "$CORTEX_DEV_ADD_CONTROL_PLANE_DASHBOARD" = "true" ]; then
    kubectl apply -f manifests/grafana/grafana-dashboard-control-plane.yaml >/dev/null
  fi
  python render_template.py $CORTEX_CLUSTER_CONFIG_FILE manifests/grafana/grafana.yaml.j2 | kubectl apply -f - >/dev/null
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

function start_controller_manager() {
  echo -n "￮ starting controller manager "

  kustomize build config/default | kubectl delete --ignore-not-found=true -f - >/dev/null

  cd config/manager \
    && kustomize edit set image controller=${CORTEX_IMAGE_CONTROLLER_MANAGER} \
    && cd ../.. > /dev/null

  kustomize build config/default | kubectl apply -f - >/dev/null
  echo "✓"
}

function restart_controller_manager() {
  echo -n "￮ restarting controller manager "

  kubectl rollout restart deployments/operator-controller-manager >/dev/null

  echo "✓"
}

function resize_nodegroups() {
  if [ -z "$CORTEX_NODEGROUP_NAMES_TO_UPDATE" ]; then
    return
  fi

  eksctl get nodegroup --cluster=$CORTEX_CLUSTER_NAME --region=$CORTEX_REGION --verbose=0 -o json > nodegroups.json
  eks_ng_len=$(cat nodegroups.json | jq -r length)
  cfg_ng_len=$(cat $CORTEX_CLUSTER_CONFIG_FILE | yq -r .node_groups | yq -r length)

  for cfg_ng_name in $CORTEX_NODEGROUP_NAMES_TO_UPDATE; do
    has_ng="false"
    for eks_idx in $(seq 0 $(($eks_ng_len-1))); do
      stack_ng=$(cat nodegroups.json | jq -r .[$eks_idx].Name)
      if [ "$stack_ng" = "cx-operator" ]; then
        continue
      fi
      if [[ "$stack_ng" == *"$cfg_ng_name" ]]; then
        has_ng="true"
        break
      fi
    done

    if [ "$has_ng" == "false" ]; then
      echo -e "error: \"$cfg_ng_name\" nodegroup (\"cx-*-$cfg_ng_name\" on aws) couldn't be scaled because stack couldn't be found\n"
      exit 1
    fi

    for cfg_idx in $(seq 0 $(($cfg_ng_len-1))); do
      cfg_ng=$(cat $CORTEX_CLUSTER_CONFIG_FILE | yq -r .node_groups[$cfg_idx].name)
      if [ "$cfg_ng" = "$cfg_ng_name" ]; then
        break
      fi
    done

    desired=$(cat nodegroups.json | jq -r .[$eks_idx].DesiredCapacity)
    existing_min=$(cat nodegroups.json | jq -r .[$eks_idx].MinSize)
    existing_max=$(cat nodegroups.json | jq -r .[$eks_idx].MaxSize)
    updating_min=$(cat $CORTEX_CLUSTER_CONFIG_FILE | yq -r .node_groups[$cfg_idx].min_instances)
    updating_max=$(cat $CORTEX_CLUSTER_CONFIG_FILE | yq -r .node_groups[$cfg_idx].max_instances)

    if [ "$desired" -lt $updating_min ]; then
      desired=$updating_min
    fi
    if [ "$desired" -gt $updating_max ]; then
      desired=$updating_max
    fi

    if [ "$existing_min" != "$updating_min" ] && [ "$existing_max" != "$updating_max" ]; then
      echo "￮ nodegroup $cfg_ng_name: updating min instances to $updating_min and max instances to $updating_max"
      eksctl scale nodegroup --cluster=$CORTEX_CLUSTER_NAME --region=$CORTEX_REGION $stack_ng --nodes $desired --nodes-min $updating_min --nodes-max $updating_max --timeout "60m"
      echo
    elif [ "$existing_min" != "$updating_min" ]; then
      echo "￮ nodegroup $cfg_ng_name: updating min instances to $updating_min"
      eksctl scale nodegroup --cluster=$CORTEX_CLUSTER_NAME --region=$CORTEX_REGION $stack_ng --nodes $desired --nodes-min $updating_min --timeout "60m"
      echo
    elif [ "$existing_max" != "$updating_max" ]; then
      echo "￮ nodegroup $cfg_ng_name: updating max instances to $updating_max"
      eksctl scale nodegroup --cluster=$CORTEX_CLUSTER_NAME --region=$CORTEX_REGION $stack_ng --nodes $desired --nodes-max $updating_max --timeout "60m"
      echo
    fi
  done

  rm nodegroups.json
}

function add_nodegroups() {
  if [ -z "$CORTEX_NODEGROUP_NAMES_TO_ADD" ]; then
    return
  fi

  nodegroup_names="$(join_by , $CORTEX_NODEGROUP_NAMES_TO_ADD)"

  echo "￮ adding new nodegroup(s) to the cluster ..."
  python generate_eks.py $CORTEX_CLUSTER_CONFIG_FILE manifests/ami.json --add-cortex-node-groups="$nodegroup_names" > /workspace/nodegroups.yaml
  eksctl create nodegroup --timeout=$EKSCTL_NODEGROUP_TIMEOUT --install-neuron-plugin=false --install-nvidia-plugin=false --skip-outdated-addons-check -f /workspace/nodegroups.yaml
  rm /workspace/nodegroups.yaml
  echo
}

function remove_nodegroups() {
  if [ -z "$CORTEX_EKS_NODEGROUP_NAMES_TO_REMOVE" ]; then
    return
  fi

  eks_nodegroup_names="$(join_by , $CORTEX_EKS_NODEGROUP_NAMES_TO_REMOVE)"

  echo "￮ removing nodegroup(s) from the cluster ..."
  python generate_eks.py $CORTEX_CLUSTER_CONFIG_FILE manifests/ami.json --remove-eks-node-groups="$eks_nodegroup_names" > /workspace/nodegroups.yaml
  eksctl delete nodegroup --timeout=$EKSCTL_NODEGROUP_TIMEOUT --approve -f /workspace/nodegroups.yaml
  rm /workspace/nodegroups.yaml
  echo
}

function setup_ipvs() {
  # get a random kube-proxy pod
  kubectl rollout status daemonset kube-proxy -n kube-system --timeout 30m >/dev/null
  kube_proxy_pod=$(kubectl get pod -n kube-system -l k8s-app=kube-proxy -o jsonpath='{.items[*].metadata.name}' | cut -d " " -f1)

  # export kube-proxy's current config
  #kubectl exec -it -n kube-system ${kube_proxy_pod} -- cat /var/lib/kube-proxy-config/config > proxy_config.yaml
  kubectl get configmap kube-proxy-config -n kube-system -o yaml > proxy_config.yaml

  # upgrade proxy mode from the exported kube-proxy config
  python upgrade_kube_proxy_mode.py proxy_config.yaml > upgraded_proxy_config.yaml

  # update kube-proxy's configmap to include the updated configuration
  kubectl get configmap -n kube-system kube-proxy -o yaml | yq --arg replace "`cat upgraded_proxy_config.yaml`" '.data.config=$replace' | kubectl apply -f - >/dev/null

  # patch the kube-proxy daemonset
  kubectl patch ds -n kube-system kube-proxy --patch "$(cat manifests/kube-proxy.patch.yaml)" >/dev/null
  kubectl rollout status daemonset kube-proxy -n kube-system --timeout 30m >/dev/null
}

function setup_istio() {
  if ! grep -q "istio-customgateway-certs" <<< $(kubectl get secret -n istio-system); then
    WEBSITE=localhost
    openssl req -subj "/C=US/CN=$WEBSITE" -newkey rsa:2048 -nodes -keyout $WEBSITE.key -x509 -days 3650 -out $WEBSITE.crt >/dev/null 2>&1
    kubectl create -n istio-system secret tls istio-customgateway-certs --key $WEBSITE.key --cert $WEBSITE.crt >/dev/null
  fi

  python render_template.py $CORTEX_CLUSTER_CONFIG_FILE manifests/istio.yaml.j2 > /workspace/istio.yaml
  output_if_error istio-${ISTIO_VERSION}/bin/istioctl install --skip-confirmation --filename /workspace/istio.yaml
}

function install_ebs_csi_driver() {
  aws_account_id=$(aws sts get-caller-identity --query "Account" --output text)
  # assert aws_account_id is not empty
  if [ -z "$aws_account_id" ]; then
    echo "unable to get aws account id"
    exit 1
  fi
  oidc_id=$(aws eks describe-cluster --name $CORTEX_CLUSTER_NAME --region $CORTEX_REGION --query "cluster.identity.oidc.issuer" --output text | cut -d '/' -f 5)
  eksctl utils associate-iam-oidc-provider --cluster $CORTEX_CLUSTER_NAME --region $CORTEX_REGION --approve
  role_name="AmazonEKS_EBS_CSI_DriverRole_${CORTEX_CLUSTER_NAME}"
  eksctl create iamserviceaccount \
    --name ebs-csi-controller-sa \
    --namespace kube-system \
    --cluster $CORTEX_CLUSTER_NAME \
    --region $CORTEX_REGION \
    --attach-policy-arn arn:aws:iam::aws:policy/service-role/AmazonEBSCSIDriverPolicy \
    --approve \
    --role-only \
    --role-name $role_name
  eksctl create addon --name aws-ebs-csi-driver --cluster $CORTEX_CLUSTER_NAME --region $CORTEX_REGION --service-account-role-arn arn:aws:iam::$aws_account_id:role/$role_name --force
}

function update_networking() {
  prev_ssl_certificate_arn=$(kubectl get svc ingressgateway-apis -n=istio-system -o json | jq -r '.metadata.annotations."service.beta.kubernetes.io/aws-load-balancer-ssl-cert"')

  if [ "$prev_ssl_certificate_arn" = "null" ]; then
      prev_ssl_certificate_arn=""
  fi

  new_ssl_certificate_arn=$(cat $CORTEX_CLUSTER_CONFIG_FILE | yq -r .ssl_certificate_arn)

  if [ "$new_ssl_certificate_arn" = "null" ]; then
      new_ssl_certificate_arn=""
  fi

  prev_api_whitelist_ip_address=$(kubectl get svc ingressgateway-apis  -n=istio-system -o yaml | yq -r -c ".spec.loadBalancerSourceRanges")
  prev_operator_whitelist_ip_address=$(kubectl get svc ingressgateway-operator  -n=istio-system -o yaml | yq -r -c ".spec.loadBalancerSourceRanges")

  new_api_whitelist_ip_address=$(cat $CORTEX_CLUSTER_CONFIG_FILE | yq -r -c ".api_load_balancer_cidr_white_list")
  new_operator_whitelist_ip_address=$(cat $CORTEX_CLUSTER_CONFIG_FILE | yq -r -c ".operator_load_balancer_cidr_white_list")

  if [ "$prev_ssl_certificate_arn" = "$new_ssl_certificate_arn" ] && [ "$prev_api_whitelist_ip_address" = "$new_api_whitelist_ip_address" ] && [ "$prev_operator_whitelist_ip_address" = "$new_operator_whitelist_ip_address" ] ; then
      return
  fi

  echo -n "￮ updating networking configuration "

  if [ "$new_ssl_certificate_arn" != "$prev_ssl_certificate_arn" ] ; then
      # there is a bug where changing the certificate annotation will not cause the HTTPS listener in the NLB to update
      # the current workaround is to delete the HTTPS listener and have it recreated with istioctl
      if [ "$prev_ssl_certificate_arn" != "" ] ; then
        kubectl patch svc ingressgateway-apis -n=istio-system --type=json -p="[{'op': 'remove', 'path': '/metadata/annotations/service.beta.kubernetes.io~1aws-load-balancer-ssl-cert'}]" >/dev/null
      fi
      https_index=$(kubectl get svc ingressgateway-apis -n=istio-system -o json  | jq '.spec.ports | map(.name == "https") | index(true)')
      if [ "$https_index" != "null" ] ; then
        kubectl patch svc ingressgateway-apis -n=istio-system --type=json -p="[{'op': 'remove', 'path': '/spec/ports/$https_index'}]" >/dev/null
      fi
  fi

  python render_template.py $CORTEX_CLUSTER_CONFIG_FILE manifests/istio.yaml.j2 > /workspace/istio.yaml
  output_if_error istio-${ISTIO_VERSION}/bin/istioctl install --skip-confirmation --filename /workspace/istio.yaml
  python render_template.py $CORTEX_CLUSTER_CONFIG_FILE manifests/apis.yaml.j2 > /workspace/apis.yaml
  kubectl apply -f /workspace/apis.yaml >/dev/null

  echo "✓"
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
  prometheus_ready=""
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
      if [ "${prometheus_ready}" != "" ]; then
        echo "prometheus is ready: $prometheus_ready"
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

      echo "additional networking events:"
      kubectl get events -n=istio-system --field-selector involvedObject.kind=Service --sort-by=".metadata.managedFields[0].time" | tail -10
      kubectl get events -n=istio-system --field-selector involvedObject.kind=Pod --sort-by=".metadata.managedFields[0].time" | tail -10
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

    if [ "$prometheus_ready" == "" ]; then
      readyReplicas=$(kubectl get statefulset -n prometheus prometheus-prometheus -o jsonpath='{.status.readyReplicas}' 2> /dev/null)
      desiredReplicas=$(kubectl get statefulset -n prometheus prometheus-prometheus -o jsonpath='{.status.replicas}' 2> /dev/null)

      if [ "$readyReplicas" != "" ] && [ "$desiredReplicas" != "" ]; then
        if [ "$readyReplicas" == "$desiredReplicas" ]; then
          prometheus_ready="true"
        else
          prometheus_ready="false"
        fi
      fi

      if [ "$prometheus_ready" != "true" ]; then
        success_cycles=0
        continue
      fi
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

  echo "operator:          $operator_endpoint"  # before modifying this, search for this prefix
  echo "api load balancer: $api_load_balancer_endpoint"
}

function get_operator_endpoint() {
  kubectl -n=istio-system get service ingressgateway-operator -o json | tr -d '[:space:]' | sed 's/.*{\"hostname\":\"\(.*\)\".*/\1/'
}

function get_api_load_balancer_endpoint() {
  kubectl -n=istio-system get service ingressgateway-apis -o json | tr -d '[:space:]' | sed 's/.*{\"hostname\":\"\(.*\)\".*/\1/'
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

function join_by {
  local IFS="$1"
  shift
  echo "$*"
}

main
