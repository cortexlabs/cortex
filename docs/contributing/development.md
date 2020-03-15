# Development

## Prerequisites

1. Go (>=1.13.5)
2. Docker
3. eksctl
4. kubectl
5. aws-cli
6. rerun

#### Go

For a system wide installation of Go, run the following commands:
```
wget https://dl.google.com/go/go1.13.linux-amd64.tar.gz && \
sudo tar -xvf go1.13.linux-amd64.tar.gz && \
sudo mv go /usr/local && \
rm go1.13.linux-amd64.tar.gz
```

#### Docker

Follow [these instructions](https://www.digitalocean.com/community/tutorials/how-to-install-and-use-docker-on-ubuntu-18-04) to install Docker.

#### eksctl

To install eksctl run:
```bash
curl --silent --location "https://github.com/weaveworks/eksctl/releases/latest/download/eksctl_$(uname -s)_amd64.tar.gz" | tar xz -C /tmp
sudo mv /tmp/eksctl /usr/local/bin
```

#### kubectl

Follow [these instructions](https://kubernetes.io/docs/tasks/tools/install-kubectl/#install-kubectl-binary-with-curl-on-linux) to install kubectl.

#### aws-cli

Follow [these instructions](https://github.com/aws/aws-cli) to install aws-cli.

#### rerun

Follow [these instructions](https://github.com/alexch/rerun) to install rerun. If you don't want to use `gem` to install rerun, then the apt version can be used, although it's an older version of it.
```bash
sudo apt update && sudo apt install rerun
```

#### Conda

Some users may prefer using conda instead of installing binaries system-wide. The following is an example for the Go binary which can be easily applied to other binaries as well: `eksctl`, `kubectl` or `aws-cli`.

---

Create a `go/bin` directories inside `/path/to/miniconda/env`, move the binary there and then follow [these instructions](https://docs.conda.io/projects/conda/en/latest/user-guide/tasks/manage-environments.html#macos-and-linux) to set/unset `GOPATH`, `GOBIN` and `PATH` appropriately every time `conda activate env` or `conda deactivate` is run.

This is for `activate.d/env_vars.sh`:
```bash
export GOPATH=$HOME/.miniconda3/envs/cortex-env/go
export GOBIN=$HOME/.miniconda3/envs/cortex-env/go/bin
export PATH=$PATH:$GOBIN
```
This is for `deactivate.d/env_vars.sh`:
```bash
unset GOPATH
unset GOBIN
PATH=$(echo "$PATH" | sed -e 's/:\/home\/user\/.miniconda3\/envs\/cortex-env\/go\/bin$//')
```

## Cortex dev environment

### Clone the repo

Clone the project:

```bash
git clone https://github.com/cortexlabs/cortex.git
cd cortex
```

Run the tests:

```bash
make test
```

### Image Registry

Create a config directory in the repo's root directory:
```bash
mkdir dev/config
```

Next, create `dev/config/build.sh`. Add the following content to it:
```bash
export CORTEX_VERSION="master"

export REGISTRY_REGION="us-west-2"
```

Create the AWS Elastic Container Registry:

```bash
make registry-create
```

Take note of the registry URL, this will be needed shortly.

Create the S3 buckets:

```bash
aws s3 mb s3://cortex-cluster-<your_name>
aws s3 mb s3://cortex-cli-<your_name>  # if you'll be uploading your compiled CLI
```

### Cluster

Update `dev/config/build.sh`. Paste the following config, and update `CLI_BUCKET_NAME`, `CLI_BUCKET_REGION`, `REGISTRY_URL` (the), and `REGISTRY_REGION` accordingly:

```bash
export CORTEX_VERSION="master"

export REGISTRY_REGION="us-west-2"
export REGISTRY_URL="XXXXXXXX.dkr.ecr.us-west-2.amazonaws.com"

# optional, only used for make ci-build-and-upload-cli
export CLI_BUCKET_NAME="cortex-cli-<your_name>"
export CLI_BUCKET_REGION="us-west-2"
```

Create `dev/config/cluster.yaml`. Paste the following config, and update `cortex_bucket`, `cortex_region`, `aws_access_key_id`, `aws_secret_access_key`, and all registry URLs accordingly:

```yaml
aws_access_key_id: ***
aws_secret_access_key: ***

instance_type: m5.large
min_instances: 2
max_instances: 5
bucket: cortex-cluster-<your_name>
region: us-west-2
log_group: cortex
cluster_name: cortex

image_python_serve: XXXXXXXX.dkr.ecr.us-west-2.amazonaws.com/cortexlabs/python-serve:latest
image_python_serve_gpu: XXXXXXXX.dkr.ecr.us-west-2.amazonaws.com/cortexlabs/python-serve-gpu:latest
image_tf_serve: XXXXXXXX.dkr.ecr.us-west-2.amazonaws.com/cortexlabs/tf-serve:latest
image_tf_serve_gpu: XXXXXXXX.dkr.ecr.us-west-2.amazonaws.com/cortexlabs/tf-serve-gpu:latest
image_tf_api: XXXXXXXX.dkr.ecr.us-west-2.amazonaws.com/cortexlabs/tf-api:latest
image_onnx_serve: XXXXXXXX.dkr.ecr.us-west-2.amazonaws.com/cortexlabs/onnx-serve:latest
image_onnx_serve_gpu: XXXXXXXX.dkr.ecr.us-west-2.amazonaws.com/cortexlabs/onnx-serve-gpu:latest
image_operator: XXXXXXXX.dkr.ecr.us-west-2.amazonaws.com/cortexlabs/operator:latest
image_manager: XXXXXXXX.dkr.ecr.us-west-2.amazonaws.com/cortexlabs/manager:latest
image_downloader: XXXXXXXX.dkr.ecr.us-west-2.amazonaws.com/cortexlabs/downloader:latest
image_request_monitor: XXXXXXXX.dkr.ecr.us-west-2.amazonaws.com/cortexlabs/request-monitor:latest
image_cluster_autoscaler: XXXXXXXX.dkr.ecr.us-west-2.amazonaws.com/cortexlabs/cluster-autoscaler:latest
image_metrics_server: XXXXXXXX.dkr.ecr.us-west-2.amazonaws.com/cortexlabs/metrics-server:latest
image_nvidia: XXXXXXXX.dkr.ecr.us-west-2.amazonaws.com/cortexlabs/nvidia:latest
image_fluentd: XXXXXXXX.dkr.ecr.us-west-2.amazonaws.com/cortexlabs/fluentd:latest
image_statsd: XXXXXXXX.dkr.ecr.us-west-2.amazonaws.com/cortexlabs/statsd:latest
image_istio_proxy: XXXXXXXX.dkr.ecr.us-west-2.amazonaws.com/cortexlabs/istio-proxy:latest
image_istio_pilot: XXXXXXXX.dkr.ecr.us-west-2.amazonaws.com/cortexlabs/istio-pilot:latest
image_istio_citadel: XXXXXXXX.dkr.ecr.us-west-2.amazonaws.com/cortexlabs/istio-citadel:latest
image_istio_galley: XXXXXXXX.dkr.ecr.us-west-2.amazonaws.com/cortexlabs/istio-galley:latest
```

### Building

Add this to your bash profile (e.g. `~/.bash_profile`, `~/.profile` or `~/.bashrc`):

```bash
alias cortex-dev='<path/to/cortex>/bin/cortex'  # replace <path/to/cortex> with the path to the cortex repo that you cloned
```

Refresh your bash profile:

```bash
. ~/.bash_profile  # or: `. ~/.bashrc`
```

Build and push all Cortex images:

```bash
make registry-all
```

Build the Cortex CLI:

```bash
make cli  # the binary will be placed in <path/to/cortex>/bin/cortex
cortex-dev version  # should show "master"
```

### Cortex cluster

Start Cortex:

```bash
make cluster-up
```

Tear down the Cortex cluster:

```bash
make cluster-down
```

### Deploy an example

```bash
cd examples/pytorch/iris-classifier
cortex-dev deploy
```

## Off-cluster operator

If you're making changes in the operator and want faster iterations, you can run an off-cluster operator.

1. `make operator-stop` to stop the in-cluster operator
2. `make devstart` to run the off-cluster operator (which rebuilds the CLI and restarts the Operator when files change)

If you want to switch back to the in-cluster operator:

1. `<ctrl+c>` to stop your off-cluster operator
2. `make operator-start` to install the operator in your cluster

## Dev workflow

1. `make cluster-up`
2. `make devstart`
3. Make changes
4. `make registry-dev`
5. Test your changes with projects in `examples` or your own

See `Makefile` for additional dev commands.

Feel free to [chat with us](https://gitter.im/cortexlabs/cortex) if you have any questions.
