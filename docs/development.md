# Build from Source and Development

## Prerequisites

1. Go (1.11.4<)
1. Docker
1. eksctl

## Build the CLI from Source
To build the Cortex CLI, run `make build-cli` in the project root directory.


## Build workload images from Source
To build the Cortex workload images, run `make build-images` in the project root directory.
Then, specify your AWS registry and region in a dev/config/build.sh file:

dev/config/build.sh

```bash
export REGISTRY_URL="389128963714.dkr.ecr.us-west-2.amazonaws.com"
export REGISTRY_REGION="us-west-2"
```

Then, update image paths in the cortex config[link to config]

## Cortex Dev Environemnt

Clone the project:

```bash
git clone git@github.com:cortexlabs/cortex.git ~/src/github.com/cortexlabs/cortex
cd ~/src/github.com/cortexlabs/cortex
```

Create the AWS Elastic Container Registry:

```bash
make registry-create
```

Note the registry URL, as you will use it in the upcoming steps

Create the following S3 buckets:

- cortex-cluster-<your_name>
- cortex-cli-<your_name>

Make the config folder:

```bash
mkdir -p ~/src/github.com/cortexlabs/cortex/dev/config
```

Create `~/src/github.com/cortexlabs/cortex/dev/config/k8s.sh`. Paste the following config, and update `K8S_REGION`, `K8S_ZONE`, and `K8S_KOPS_BUCKET` accordingly:

```bash
# EKS and KOPS
export K8S_NAME="cortex"
export K8S_REGION="us-west-2"
export K8S_NODE_INSTANCE_TYPE="t3.small"
export K8S_NODE_COUNT="3"
# KOPS
export K8S_KOPS_BUCKET="cortex-kops-<your_name>"
export K8S_ZONE="us-west-2a"
export K8S_MASTER_INSTANCE_TYPE="t3.micro"
export K8S_MASTER_VOLUME_SIZE="32"
export K8S_NODE_VOLUME_SIZE="32"
```

Create `~/src/github.com/cortexlabs/cortex/dev/config/cortex.sh`. Paste the following config, and update `CORTEX_BUCKET`, `CORTEX_REGION`, `AWS_ACCESS_KEY_ID`, `AWS_SECRET_ACCESS_KEY`, and all registry URLs accordingly:

```bash
export CORTEX_LOG_GROUP="cortex"
export CORTEX_BUCKET="cortex-cluster-<your_name>"
export CORTEX_REGION="us-west-2"
export CORTEX_NAMESPACE="cortex"

export CORTEX_IMAGE_ARGO_CONTROLLER="XXXXXXXX.dkr.ecr.us-west-2.amazonaws.com/cortexlabs/argo-controller:latest"
export CORTEX_IMAGE_ARGO_EXECUTOR="XXXXXXXX.dkr.ecr.us-west-2.amazonaws.com/cortexlabs/argo-executor:latest"
export CORTEX_IMAGE_FLUENTD="XXXXXXXX.dkr.ecr.us-west-2.amazonaws.com/cortexlabs/fluentd:latest"
export CORTEX_IMAGE_NGINX_BACKEND="XXXXXXXX.dkr.ecr.us-west-2.amazonaws.com/cortexlabs/nginx-backend:latest"
export CORTEX_IMAGE_NGINX_CONTROLLER="XXXXXXXX.dkr.ecr.us-west-2.amazonaws.com/cortexlabs/nginx-controller:latest"
export CORTEX_IMAGE_OPERATOR="XXXXXXXX.dkr.ecr.us-west-2.amazonaws.com/cortexlabs/operator:latest"
export CORTEX_IMAGE_SPARK="XXXXXXXX.dkr.ecr.us-west-2.amazonaws.com/cortexlabs/spark:latest"
export CORTEX_IMAGE_SPARK_OPERATOR="XXXXXXXX.dkr.ecr.us-west-2.amazonaws.com/cortexlabs/spark-operator:latest"
export CORTEX_IMAGE_TF_SERVE="XXXXXXXX.dkr.ecr.us-west-2.amazonaws.com/cortexlabs/tf-serve:latest"
export CORTEX_IMAGE_TF_TRAIN="XXXXXXXX.dkr.ecr.us-west-2.amazonaws.com/cortexlabs/tf-train:latest"
export CORTEX_IMAGE_TF_TRANSFORM="XXXXXXXX.dkr.ecr.us-west-2.amazonaws.com/cortexlabs/tf-transform:latest"

export AWS_ACCESS_KEY_ID="XXXXXX"
export AWS_SECRET_ACCESS_KEY="XXXXXX"
```

Create `~/src/github.com/cortexlabs/cortex/dev/config/build.sh`. Paste the following config, and update `CLI_BUCKET_NAME`, `CLI_BUCKET_REGION`, `REGISTRY_URL`, and `REGISTRY_REGION` accordingly:

```bash
export VERSION="latest"
export CLI_BUCKET_NAME="cortex-cli-<your_name>"
export CLI_BUCKET_REGION="us-west-2"
export REGISTRY_URL="XXXXXXXX.dkr.ecr.us-west-2.amazonaws.com"
export REGISTRY_REGION="us-west-2"
```

Build and push all Cortex images (this will take a while)

```bash
make registry-dev
```

Start Kubernetes cluster

```bash
make eks-up
```

Install Cortex on the cluster

```bash
make cinstall
```

Build and configure the Cortex CLI

```bash
make build-cli
# use the operator's URL printed at the end of make cinstall
cx configure
```
