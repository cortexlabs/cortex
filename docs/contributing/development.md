# Development

## Remote development

Unless your internet connection is very fast or you will only be working on the CLI, it is recommended to run your development environment on a cloud instance, e.g. an AWS EC2 instance or GCP VM (due to frequent docker registry pushing). There are a variety of ways to develop on a remote VM, feel free to reach out on our [gitter](https://gitter.im/cortexlabs/cortex) and we can point you in the right direction based on your operating system and editor preferences.

## Prerequisites

1. Go (>=1.14)
2. Docker
3. eksctl
4. kubectl
5. aws-cli

Also, please use the VS Code [yaml extension](https://marketplace.visualstudio.com/items?itemName=redhat.vscode-yaml) and enable auto-formatting for YAML files.

### Go

To install Go on linux, run:

```bash
wget https://dl.google.com/go/go1.14.2.linux-amd64.tar.gz && \
sudo tar -xvf go1.14.2.linux-amd64.tar.gz && \
sudo mv go /usr/local && \
rm go1.14.2.linux-amd64.tar.gz
```

### Docker

To install Docker on Ubuntu, run:

```bash
sudo apt install docker.io && \
sudo systemctl start docker && \
sudo systemctl enable docker && \
sudo groupadd docker && \
sudo gpasswd -a $USER docker
```

### eksctl

To install eksctl run:

```bash
curl --silent --location "https://github.com/weaveworks/eksctl/releases/latest/download/eksctl_$(uname -s)_amd64.tar.gz" | tar xz -C /tmp && \
sudo mv /tmp/eksctl /usr/local/bin
```

### kubectl

To install kubectl on linux, run:

```bash
curl -LO https://storage.googleapis.com/kubernetes-release/release/`curl -s https://storage.googleapis.com/kubernetes-release/release/stable.txt`/bin/linux/amd64/kubectl && \
chmod +x ./kubectl && \
sudo mv ./kubectl /usr/local/bin/kubectl
```

### aws-cli (v1)

Follow [these instructions](https://github.com/aws/aws-cli#installation) to install aws-cli (v1).

E.g. to install it globally, run:

```bash
sudo python -m pip install awscli
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

Next, create `dev/config/build.sh`. Add the following content to it (you may use a different region for `REGISTRY_REGION`):

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

# optional, only used for dev/build_cli.sh
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

image_operator: XXXXXXXX.dkr.ecr.us-west-2.amazonaws.com/cortexlabs/operator:latest
image_manager: XXXXXXXX.dkr.ecr.us-west-2.amazonaws.com/cortexlabs/manager:latest
image_downloader: XXXXXXXX.dkr.ecr.us-west-2.amazonaws.com/cortexlabs/downloader:latest
image_request_monitor: XXXXXXXX.dkr.ecr.us-west-2.amazonaws.com/cortexlabs/request-monitor:latest
image_cluster_autoscaler: XXXXXXXX.dkr.ecr.us-west-2.amazonaws.com/cortexlabs/cluster-autoscaler:latest
image_metrics_server: XXXXXXXX.dkr.ecr.us-west-2.amazonaws.com/cortexlabs/metrics-server:latest
image_inferentia: XXXXXXXX.dkr.ecr.us-west-2.amazonaws.com/cortexlabs/inferentia:latest
image_neuron_rtd: XXXXXXXX.dkr.ecr.us-west-2.amazonaws.com/cortexlabs/neuron-rtd:latest
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
export CORTEX_DEV_DEFAULT_PREDICTOR_IMAGE_REGISTRY="XXXXXXXX.dkr.ecr.us-west-2.amazonaws.com/cortexlabs"  # set the default image for APIs
export CORTEX_TELEMETRY_SENTRY_DSN="https://c334df915c014ffa93f2076769e5b334@sentry.io/1848098"  # redirect analytics to our dev environment
export CORTEX_TELEMETRY_SEGMENT_WRITE_KEY="0WvoJyCey9z1W2EW7rYTPJUMRYat46dl"  # redirect error reporting to our dev environment
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
cortex deploy examples/pytorch/iris-classifier --env aws
```

## Off-cluster operator

If you're making changes in the operator and want faster iterations, you can run an off-cluster operator.

1. `make tools` to install the necessary dependencies to run the operator
2. `make operator-stop` to stop the in-cluster operator
3. `make devstart` to run the off-cluster operator (which rebuilds the CLI and restarts the Operator when files change)

If you want to switch back to the in-cluster operator:

1. `<ctrl+c>` to stop your off-cluster operator
2. `make cluster-configure` to install the operator in your cluster

## Dev workflow

1. `make cluster-up`
2. `make devstart`
3. Make changes
4. `make registry-dev`
5. Test your changes with projects in `examples` or your own

See `Makefile` for additional dev commands.

Feel free to [chat with us](https://gitter.im/cortexlabs/cortex) if you have any questions.
