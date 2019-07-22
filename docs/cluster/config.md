# Cluster Configuration

These environment variables can be modified and exported before running `cortex.sh` commands. Alternatively, a configuration file may be provided to `cortex.sh` via the `--config` flag (e.g. `cortex.sh --config=./config.sh install`). Default values are shown.

<!-- CORTEX_VERSION_STABLE -->

```bash
# config.sh

# AWS credentials
export AWS_ACCESS_KEY_ID="***"
export AWS_SECRET_ACCESS_KEY="***"

# The name of the CloudWatch log group Cortex will use
export CORTEX_LOG_GROUP="cortex"

# The name of the S3 bucket Cortex will use
export CORTEX_BUCKET="cortex-[RANDOM_ID]"

# The AWS region Cortex will use
export CORTEX_REGION="us-west-2"

# The AWS zones Cortex will use (e.g. "us-east-1a,us-east-1b")
export CORTEX_ZONES=""

# The name of the EKS cluster Cortex will use
export CORTEX_CLUSTER="cortex"

# The AWS node type Cortex will use
export CORTEX_NODE_TYPE="t3.medium"

# Minimum number of nodes in the cluster
export CORTEX_NODES_MIN=2

# Maximum number of nodes in the cluster
export CORTEX_NODES_MAX=5

# The name of the Kubernetes namespace Cortex will use
export CORTEX_NAMESPACE="cortex"

# Image paths
export CORTEX_IMAGE_MANAGER="cortexlabs/manager:master"
export CORTEX_IMAGE_FLUENTD="cortexlabs/fluentd:master"
export CORTEX_IMAGE_NGINX_BACKEND="cortexlabs/nginx-backend:master"
export CORTEX_IMAGE_NGINX_CONTROLLER="cortexlabs/nginx-controller:master"
export CORTEX_IMAGE_OPERATOR="cortexlabs/operator:master"
export CORTEX_IMAGE_SPARK="cortexlabs/spark:master"
export CORTEX_IMAGE_SPARK_OPERATOR="cortexlabs/spark-operator:master"
export CORTEX_IMAGE_TF_SERVE="cortexlabs/tf-serve:master"
export CORTEX_IMAGE_TF_TRAIN="cortexlabs/tf-train:master"
export CORTEX_IMAGE_TF_API="cortexlabs/tf-api:master"
export CORTEX_IMAGE_TF_TRAIN_GPU="cortexlabs/tf-train-gpu:master"
export CORTEX_IMAGE_TF_SERVE_GPU="cortexlabs/tf-serve-gpu:master"
export CORTEX_IMAGE_ONNX_SERVE="cortexlabs/onnx-serve:master"
export CORTEX_IMAGE_ONNX_SERVE_GPU="cortexlabs/onnx-serve-gpu:master"
export CORTEX_IMAGE_PYTHON_PACKAGER="cortexlabs/python-packager:master"
export CORTEX_IMAGE_CLUSTER_AUTOSCALER="cortexlabs/cluster-autoscaler:master"
export CORTEX_IMAGE_NVIDIA="cortexlabs/nvidia:master"
export CORTEX_IMAGE_METRICS_SERVER="cortexlabs/metrics-server:master"

# Flag to enable collecting error reports and usage stats. If flag is not set to either "true" or "false", you will be prompted.
export CORTEX_ENABLE_TELEMETRY=""
```
