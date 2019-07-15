# Development

## Prerequisites

1. Go (>=1.12.6)
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

Take note of the registry URL, this will be needed shortly.

Create the S3 buckets:

```bash
aws s3 mb s3://cortex-cluster-<your_name>
aws s3 mb s3://cortex-cli-<your_name> # (if you'll be uploading your compiled CLI)
```

### Configuration

Make the config folder:

```bash
mkdir dev/config
```

Create `dev/config/cortex.sh`. Paste the following config, and update `CORTEX_BUCKET`, `CORTEX_REGION`, `AWS_ACCESS_KEY_ID`, `AWS_SECRET_ACCESS_KEY`, and all registry URLs accordingly:

```bash
export AWS_ACCESS_KEY_ID="XXXXXX"
export AWS_SECRET_ACCESS_KEY="XXXXXX"

export CORTEX_LOG_GROUP="cortex"
export CORTEX_BUCKET="cortex-cluster-<your_name>"
export CORTEX_REGION="us-west-2"

export CORTEX_CLUSTER="cortex"
export CORTEX_NODE_TYPE="t3.medium"
export CORTEX_NODES_MIN="2"
export CORTEX_NODES_MAX="5"
export CORTEX_NAMESPACE="cortex"

export CORTEX_IMAGE_MANAGER="XXXXXXXX.dkr.ecr.us-west-2.amazonaws.com/cortexlabs/manager:latest"
export CORTEX_IMAGE_ARGO_CONTROLLER="XXXXXXXX.dkr.ecr.us-west-2.amazonaws.com/cortexlabs/argo-controller:latest"
export CORTEX_IMAGE_ARGO_EXECUTOR="XXXXXXXX.dkr.ecr.us-west-2.amazonaws.com/cortexlabs/argo-executor:latest"
export CORTEX_IMAGE_FLUENTD="XXXXXXXX.dkr.ecr.us-west-2.amazonaws.com/cortexlabs/fluentd:latest"
export CORTEX_IMAGE_NGINX_BACKEND="XXXXXXXX.dkr.ecr.us-west-2.amazonaws.com/cortexlabs/nginx-backend:latest"
export CORTEX_IMAGE_NGINX_CONTROLLER="XXXXXXXX.dkr.ecr.us-west-2.amazonaws.com/cortexlabs/nginx-controller:latest"
export CORTEX_IMAGE_ONNX_SERVE="XXXXXXXX.dkr.ecr.us-west-2.amazonaws.com/cortexlabs/onnx-serve:latest"
export CORTEX_IMAGE_ONNX_SERVE_GPU="XXXXXXXX.dkr.ecr.us-west-2.amazonaws.com/cortexlabs/onnx-serve-gpu:latest"
export CORTEX_IMAGE_OPERATOR="XXXXXXXX.dkr.ecr.us-west-2.amazonaws.com/cortexlabs/operator:latest"
export CORTEX_IMAGE_SPARK="XXXXXXXX.dkr.ecr.us-west-2.amazonaws.com/cortexlabs/spark:latest"
export CORTEX_IMAGE_SPARK_OPERATOR="XXXXXXXX.dkr.ecr.us-west-2.amazonaws.com/cortexlabs/spark-operator:latest"
export CORTEX_IMAGE_TF_SERVE="XXXXXXXX.dkr.ecr.us-west-2.amazonaws.com/cortexlabs/tf-serve:latest"
export CORTEX_IMAGE_TF_SERVE_GPU="XXXXXXXX.dkr.ecr.us-west-2.amazonaws.com/cortexlabs/tf-serve-gpu:latest"
export CORTEX_IMAGE_TF_TRAIN="XXXXXXXX.dkr.ecr.us-west-2.amazonaws.com/cortexlabs/tf-train:latest"
export CORTEX_IMAGE_TF_TRAIN_GPU="XXXXXXXX.dkr.ecr.us-west-2.amazonaws.com/cortexlabs/tf-train-gpu:latest"
export CORTEX_IMAGE_TF_API="XXXXXXXX.dkr.ecr.us-west-2.amazonaws.com/cortexlabs/tf-api:latest"
export CORTEX_IMAGE_PYTHON_PACKAGER="XXXXXXXX.dkr.ecr.us-west-2.amazonaws.com/cortexlabs/python-packager:latest"
export CORTEX_IMAGE_CLUSTER_AUTOSCALER="XXXXXXXX.dkr.ecr.us-west-2.amazonaws.com/cortexlabs/cluster-autoscaler:latest"
export CORTEX_IMAGE_NVIDIA="XXXXXXXX.dkr.ecr.us-west-2.amazonaws.com/cortexlabs/nvidia:latest"
export CORTEX_IMAGE_METRICS_SERVER="XXXXXXXX.dkr.ecr.us-west-2.amazonaws.com/cortexlabs/metrics-server:latest"

export CORTEX_ENABLE_TELEMETRY="false"
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

### Building

Build and push all Cortex images:

```bash
make registry-all
```

Build and configure the Cortex CLI:

```bash
make cli  # The binary will be placed in path/to/cortex/bin/cortex
path/to/cortex/bin/cortex configure
```

### Cortex Cluster

Start Cortex:

```bash
make cortex-up
```

Tear down the Cortex cluster:

```bash
make cortex-down
```

### Deployment an Example

```bash
cd examples/iris
path/to/cortex/bin/cortex deploy
```

## Off-cluster Operator

If you're making changes in the operator and want faster iterations, you can run an off-cluster operator.

1. `make operator-stop` to stop the in-cluster operator
1. `make devstart` to run the off-cluster operator (which rebuilds the CLI and restarts the Operator when files change)
1. `path/to/cortex/bin/cortex configure` (on a separate terminal) to configure your cortex CLI to use the off-cluster operator. When prompted for operator URL, use `http://localhost:8888`

Note: `make cortex-up-dev` will start Cortex without installing the operator.

If you want to switch back to the in-cluster operator:

1. `<ctrl+C>` to stop your off-cluster operator
1. `make operator-start` to install the operator in your cluster
1. `path/to/cortex/bin/cortex configure` to configure your cortex CLI to use the in-cluster operator. When prompted for operator URL, use the URL shown when running `make cortex-info`

## Dev Workflow

1. `make cortex-up-dev`
1. `make devstart`
1. Make changes
1. `make registry-dev`
1. Test your changes with projects in `examples` or your own

See `Makefile` for additional dev commands
