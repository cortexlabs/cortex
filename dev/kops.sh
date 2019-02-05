#!/bin/bash
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")"/.. >/dev/null && pwd)"

source $ROOT/dev/config/k8s.sh
export K8S_NAME="${K8S_NAME}.k8s.local"

function kops_set_cluster() {
  kops export kubecfg --name=$K8S_NAME --state="s3://$K8S_KOPS_BUCKET"
  kubectl config set-context $(kubectl config current-context) --namespace="cortex"
}

function create_kops_config() {
  cat > $ROOT/dev/config/.k8s_kops_config.yaml << EOM
apiVersion: kops/v1alpha2
kind: Cluster
metadata:
  creationTimestamp: null
  name: ${K8S_NAME}
spec:
  api:
    loadBalancer:
      type: Public
  authorization:
    rbac: {}
  channel: stable
  cloudProvider: aws
  configBase: s3://${K8S_KOPS_BUCKET}/kops/${K8S_NAME}
  etcdClusters:
  - etcdMembers:
    - instanceGroup: master-${K8S_ZONE}
      name: a
    name: main
  - etcdMembers:
    - instanceGroup: master-${K8S_ZONE}
      name: a
    name: events
  iam:
    allowContainerRegistry: true
    legacy: false
  kubernetesApiAccess:
  - 0.0.0.0/0
  kubernetesVersion: 1.11.6
  masterPublicName: api.${K8S_NAME}
  networkCIDR: 172.20.0.0/16
  networking:
    kubenet: {}
  nonMasqueradeCIDR: 100.64.0.0/10
  sshAccess:
  - 0.0.0.0/0
  subnets:
  - cidr: 172.20.32.0/19
    name: ${K8S_ZONE}
    type: Public
    zone: ${K8S_ZONE}
  topology:
    dns:
      type: Public
    masters: public
    nodes: public
  additionalPolicies:
    node: |
      [
        {
          "Effect": "Allow",
          "Action": "s3:*",
          "Resource": "*"
        },
        {
          "Effect": "Allow",
          "Action": "logs:DescribeLogGroups",
          "Resource": "*"
        },
        {
          "Effect": "Allow",
          "Action": "logs:*",
          "Resource": [
            "arn:aws:logs:${K8S_REGION}:*:log-group:*:*:*"
          ]
        }
      ]

---

apiVersion: kops/v1alpha2
kind: InstanceGroup
metadata:
  creationTimestamp: null
  labels:
    kops.k8s.io/cluster: ${K8S_NAME}
  name: master-${K8S_ZONE}
spec:
  image: kope.io/k8s-1.11-debian-stretch-amd64-hvm-ebs-2018-08-17
  machineType: ${K8S_MASTER_INSTANCE_TYPE}
  rootVolumeSize: ${K8S_MASTER_VOLUME_SIZE}
  maxSize: 1
  minSize: 1
  nodeLabels:
    kops.k8s.io/instancegroup: master-${K8S_ZONE}
  role: Master
  subnets:
  - ${K8S_ZONE}

---

apiVersion: kops/v1alpha2
kind: InstanceGroup
metadata:
  creationTimestamp: null
  labels:
    kops.k8s.io/cluster: ${K8S_NAME}
  name: nodes
spec:
  image: kope.io/k8s-1.11-debian-stretch-amd64-hvm-ebs-2018-08-17
  machineType: ${K8S_NODE_INSTANCE_TYPE}
  rootVolumeSize: ${K8S_NODE_VOLUME_SIZE}
  maxSize: ${K8S_NODES_MAX_COUNT}
  minSize: ${K8S_NODES_MIN_COUNT}
  nodeLabels:
    kops.k8s.io/instancegroup: nodes
  role: Node
  subnets:
  - ${K8S_ZONE}

---

apiVersion: kops/v1alpha2
kind: InstanceGroup
metadata:
  creationTimestamp: null
  labels:
    kops.k8s.io/cluster: ${K8S_NAME}
  name: gpu-nodes
spec:
  image: ${K8S_GPU_NODE_IMAGE}
  machineType: ${K8S_GPU_NODE_INSTANCE_TYPE}
  rootVolumeSize: ${K8S_NODE_VOLUME_SIZE}
  maxSize: ${K8S_GPU_NODES_MAX_COUNT}
  minSize: ${K8S_GPU_NODES_MIN_COUNT}
  nodeLabels:
    kops.k8s.io/instancegroup: nodes
  role: Node
  subnets:
  - ${K8S_ZONE}
EOM
}

if [ "$1" = "start" ]; then
  create_kops_config
  kops create -f $ROOT/dev/config/.k8s_kops_config.yaml --state=s3://$K8S_KOPS_BUCKET
  kops create secret --state=s3://$K8S_KOPS_BUCKET --name=$K8S_NAME sshpublickey admin -i ~/.ssh/kops.pub
  kops update cluster --yes --name=$K8S_NAME --state=s3://$K8S_KOPS_BUCKET
  kops_set_cluster
  until kops validate cluster --name=$K8S_NAME --state=s3://$K8S_KOPS_BUCKET; do
    sleep 10
  done

elif [ "$1" = "update" ]; then
  echo "Not implemented"

elif [ "$1" = "stop" ]; then
  # This doesn't seem to be necessary
  # $ROOT/cortex.sh -c=$ROOT/dev/config/cortex.sh uninstall 2>/dev/null || true

  kops delete cluster --yes --name=$K8S_NAME --state=s3://$K8S_KOPS_BUCKET
  aws s3 rm s3://$K8S_KOPS_BUCKET/$K8S_NAME --recursive

elif [ "$1" = "set" ]; then
  kops_set_cluster
fi
