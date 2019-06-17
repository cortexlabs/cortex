# Config

These environment variables can be modified and exported before running `cortex-installer.sh` commands. Alternatively, a configuration file may be provided to `cortex-installer.sh` via the `--config` flag (e.g. `cortex-installer.sh --config=./config.sh install operator`). Default values are shown.

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

# The name of the Kubernetes namespace Cortex will use
export CORTEX_NAMESPACE="cortex"

# Flag to enable collecting error reports and usage stats. If flag is not set to either "true" or "false", you will be prompted.
export CORTEX_ENABLE_TELEMETRY=""

# Image paths
export CORTEX_IMAGE_ARGO_CONTROLLER="cortexlabs/argo-controller:0.4.1"
export CORTEX_IMAGE_ARGO_EXECUTOR="cortexlabs/argo-executor:0.4.1"
export CORTEX_IMAGE_FLUENTD="cortexlabs/fluentd:0.4.1"
export CORTEX_IMAGE_NGINX_BACKEND="cortexlabs/nginx-backend:0.4.1"
export CORTEX_IMAGE_NGINX_CONTROLLER="cortexlabs/nginx-controller:0.4.1"
export CORTEX_IMAGE_OPERATOR="cortexlabs/operator:0.4.1"
export CORTEX_IMAGE_SPARK="cortexlabs/spark:0.4.1"
export CORTEX_IMAGE_SPARK_OPERATOR="cortexlabs/spark-operator:0.4.1"
export CORTEX_IMAGE_TF_SERVE="cortexlabs/tf-serve:0.4.1"
export CORTEX_IMAGE_TF_TRAIN="cortexlabs/tf-train:0.4.1"
export CORTEX_IMAGE_TF_API="cortexlabs/tf-api:0.4.1"
export CORTEX_IMAGE_TF_TRAIN_GPU="cortexlabs/tf-train-gpu:0.4.1"
export CORTEX_IMAGE_TF_SERVE_GPU="cortexlabs/tf-serve-gpu:0.4.1"
export CORTEX_IMAGE_PYTHON_PACKAGER="cortexlabs/python-packager:0.4.1"
```
