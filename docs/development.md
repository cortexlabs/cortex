# Build from Source and Development

## Prerequisites

1. Go (>=1.11.5)
1. Docker
1. eksctl
1. kubectl

## Cortex Dev Environment

Clone the project:

```bash
git clone git@github.com:cortexlabs/cortex.git
cd cortex
```

Create the AWS Elastic Container Registry:

```bash
make registry-create
```

Note the registry URL, this will be needed shortly.

Create the following S3 buckets:

- `cortex-cluster-<your_name>`
- `cortex-kops-<your_name>` (if you'll be using KOPS)
- `cortex-cli-<your_name>` (if you'll be uploading your compiled CLI)

Make the config folder:

```bash
mkdir dev/config
```

Create `dev/config/k8s.sh`. Paste the following config, replace `K8S_KOPS_BUCKET` with your bucket name (if using KOPS), and update any other variables as desired:

```bash
# EKS and KOPS

export K8S_NAME="cortex"
export K8S_REGION="us-west-2"
export K8S_NODE_INSTANCE_TYPE="t3.medium"
export K8S_NODES_MAX_COUNT="2"
export K8S_NODES_MIN_COUNT="2"
export K8S_GPU_NODES_MIN_COUNT="0"
export K8S_GPU_NODES_MAX_COUNT="0"

# KOPS only

export K8S_KOPS_BUCKET="cortex-kops-<your_name>"
export K8S_ZONE="us-west-2a"
export K8S_MASTER_INSTANCE_TYPE="t3.micro"
export K8S_MASTER_VOLUME_SIZE="32"
export K8S_NODE_VOLUME_SIZE="32"
```

Create `dev/config/cortex.sh`. Paste the following config, and update `CORTEX_BUCKET`, `CORTEX_REGION`, `AWS_ACCESS_KEY_ID`, `AWS_SECRET_ACCESS_KEY`, and all registry URLs accordingly:

```bash
export CORTEX_LOG_GROUP="cortex"
export CORTEX_BUCKET="cortex-cluster-<your_name>"
export CORTEX_REGION="us-west-2"
export CORTEX_NAMESPACE="cortex"
export CORTEX_ENABLE_TELEMETRY="false"

export CORTEX_IMAGE_ARGO_CONTROLLER="XXXXXXXX.dkr.ecr.us-west-2.amazonaws.com/cortexlabs/argo-controller:latest"
export CORTEX_IMAGE_ARGO_EXECUTOR="XXXXXXXX.dkr.ecr.us-west-2.amazonaws.com/cortexlabs/argo-executor:latest"
export CORTEX_IMAGE_FLUENTD="XXXXXXXX.dkr.ecr.us-west-2.amazonaws.com/cortexlabs/fluentd:latest"
export CORTEX_IMAGE_NGINX_BACKEND="XXXXXXXX.dkr.ecr.us-west-2.amazonaws.com/cortexlabs/nginx-backend:latest"
export CORTEX_IMAGE_NGINX_CONTROLLER="XXXXXXXX.dkr.ecr.us-west-2.amazonaws.com/cortexlabs/nginx-controller:latest"
export CORTEX_IMAGE_OPERATOR="XXXXXXXX.dkr.ecr.us-west-2.amazonaws.com/cortexlabs/operator:latest"
export CORTEX_IMAGE_SPARK="XXXXXXXX.dkr.ecr.us-west-2.amazonaws.com/cortexlabs/spark:latest"
export CORTEX_IMAGE_SPARK_OPERATOR="XXXXXXXX.dkr.ecr.us-west-2.amazonaws.com/cortexlabs/spark-operator:latest"
export CORTEX_IMAGE_TF_SERVE="XXXXXXXX.dkr.ecr.us-west-2.amazonaws.com/cortexlabs/tf-serve:latest"
export CORTEX_IMAGE_TF_SERVE_GPU="XXXXXXXX.dkr.ecr.us-west-2.amazonaws.com/cortexlabs/tf-serve-gpu:latest"
export CORTEX_IMAGE_TF_TRAIN="XXXXXXXX.dkr.ecr.us-west-2.amazonaws.com/cortexlabs/tf-train:latest"
export CORTEX_IMAGE_TF_TRAIN_GPU="XXXXXXXX.dkr.ecr.us-west-2.amazonaws.com/cortexlabs/tf-train-gpu:latest"
export CORTEX_IMAGE_TF_TRANSFORM="XXXXXXXX.dkr.ecr.us-west-2.amazonaws.com/cortexlabs/tf-transform:latest"
export CORTEX_IMAGE_PYTHON_PACKAGER="XXXXXXXX.dkr.ecr.us-west-2.amazonaws.com/cortexlabs/python-packager:latest"

export AWS_ACCESS_KEY_ID="XXXXXX"
export AWS_SECRET_ACCESS_KEY="XXXXXX"
```

Create `dev/config/build.sh`. Paste the following config, and update `CLI_BUCKET_NAME`, `CLI_BUCKET_REGION`, `REGISTRY_URL`, and `REGISTRY_REGION` accordingly:

```bash
export VERSION="master"

export REGISTRY_URL="XXXXXXXX.dkr.ecr.us-west-2.amazonaws.com"
export REGISTRY_REGION="us-west-2"

# optional, only used for make ci-build-and-upload-cli
export CLI_BUCKET_NAME="cortex-cli-<your_name>"
export CLI_BUCKET_REGION="us-west-2"
```

Build and push all Cortex images (this will take a while)

```bash
make registry-all
```

Start Kubernetes cluster and install Cortex on it

```bash
make eks-up
# or
make kops-up
```

If you're using GPUs on EKS, run this after your GPU nodes join the cluster:

```bash
kubectl apply -f https://raw.githubusercontent.com/NVIDIA/k8s-device-plugin/v1.11/nvidia-device-plugin.yml
# check for GPUs:
kubectl get nodes "-o=custom-columns=NAME:.metadata.name,GPU:.status.allocatable.nvidia\.com/gpu"
```

Build and configure the Cortex CLI

```bash
make cli  # The binary will be placed in path/to/cortex/bin/cortex
path/to/cortex/bin/cortex configure
```

Run an example application

```bash
cd examples/iris
path/to/cortex/bin/cortex deploy
```

Tear down the cluster

```bash
make eks-down
# or
make kops-down
```

## Off-cluster Operator

If you're making changes in the operator and want faster iterations, you can run an off-cluster operator.

1. `make ostop` to stop the in-cluster operator
1. `make devstart` to run the off-cluster operator (which rebuilds the CLI and restarts the Operator when files change)
1. `path/to/cortex/bin/cortex configure` (on a separate terminal) to configure your cortex CLI to use the off-cluster operator. When prompted for operator URL, use `http://localhost:8888`.

If you want to switch back to the in-cluster operator:

1. `<ctrl+C>` to stop your off-cluster operator
1. `oinstall` to install the operator in your cluster
1. `path/to/cortex/bin/cortex configure` to configure your cortex CLI to use the in-cluster operator. When prompted for operator URL, use the URL shown when running `oinstall`.

## Dev Workflow

1. Install
1. `make devstart`
1. Make changes
1. `make registry-dev`
1. Test your changes with projects in `examples` or your own

See `Makefile` for additional dev commands
