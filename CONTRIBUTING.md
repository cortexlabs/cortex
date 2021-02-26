# Contributing

## Remote development

We recommend that you run your development environment on a cloud instance due to frequent docker registry pushing, e.g. an AWS EC2 instance or GCP VM. We've had a good experience using [Mutagen](https://mutagen.io/documentation/introduction) to synchronize local / remote file systems.

## Prerequisites

### System packages

To install the necessary system packages on Ubuntu, you can run these commands:

```bash
sudo apt-get update
sudo apt install -y apt-transport-https ca-certificates software-properties-common gnupg-agent curl zip python3 python3-pip python3-dev build-essential jq tree
sudo python3 -m pip install --upgrade pip setuptools boto3
```

### Go

To install Go on linux, run:

```bash
mkdir -p ~/bin && \
wget https://dl.google.com/go/go1.14.7.linux-amd64.tar.gz && \
sudo tar -xvf go1.14.7.linux-amd64.tar.gz && \
sudo mv go /usr/local && \
rm go1.14.7.linux-amd64.tar.gz && \
echo 'export PATH="/usr/local/go/bin:$HOME/go/bin:$PATH"' >> $HOME/.bashrc
```

And then log out and back in.

### Docker

To install Docker on Ubuntu, run:

```bash
curl -fsSL https://download.docker.com/linux/ubuntu/gpg | sudo apt-key add - && \
sudo add-apt-repository "deb [arch=amd64] https://download.docker.com/linux/ubuntu $(lsb_release -cs) stable" && \
sudo apt update && \
sudo apt install -y docker-ce docker-ce-cli containerd.io && \
sudo usermod -aG docker $USER
```

And then log out and back in.

### kubectl

To install kubectl on linux, run:

```bash
curl -LO https://storage.googleapis.com/kubernetes-release/release/`curl -s https://storage.googleapis.com/kubernetes-release/release/stable.txt`/bin/linux/amd64/kubectl && \
chmod +x ./kubectl && \
sudo mv ./kubectl /usr/local/bin/kubectl
```

### eksctl

To install eksctl run:

```bash
curl --silent --location "https://github.com/weaveworks/eksctl/releases/latest/download/eksctl_$(uname -s)_amd64.tar.gz" | tar xz -C /tmp && \
sudo mv /tmp/eksctl /usr/local/bin
```

### aws-cli (v1)

Follow [these instructions](https://github.com/aws/aws-cli#installation) to install aws-cli (v1).

E.g. to install it globally, run:

```bash
sudo python3 -m pip install awscli

aws configure
```

### gcloud (v1)

Follow [these instructions](https://cloud.google.com/sdk/docs/install#deb) to install gcloud.

For example:

```bash
echo "deb [signed-by=/usr/share/keyrings/cloud.google.gpg] https://packages.cloud.google.com/apt cloud-sdk main" | sudo tee -a /etc/apt/sources.list.d/google-cloud-sdk.list && \
curl https://packages.cloud.google.com/apt/doc/apt-key.gpg | sudo apt-key --keyring /usr/share/keyrings/cloud.google.gpg add - && \
sudo apt-get update && \
sudo apt-get install -y google-cloud-sdk

gcloud init
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

### Dev tools

Install development tools by running:

```bash
make tools
```

After the dependencies are installed, there may be a diff in `go.mod` and `go.sum`, which you can revert.

Run the linter:

```bash
make lint
```

We use `gofmt` for formatting Go files, `black` for Python files (line length = 100), and the VS Code [yaml extension](https://marketplace.visualstudio.com/items?itemName=redhat.vscode-yaml) for YAML files. It is recommended to enable these in your code editor, but you can also run the Go and Python formatters from the terminal:

```bash
make format

git diff  # there should be no diff
```

### Cluster configuration

These instructions assume you'll be creating clusters on AWS and GCP. You may skip some of the steps and configuration if you'll only be developing / testing on a single cloud provider.

Create a config directory in the repo's root directory:

```bash
mkdir dev/config
```

Create `dev/config/env.sh` with the following information:

```bash
# dev/config/env.sh

export AWS_ACCOUNT_ID="***"  # you can find your account ID in the AWS web console; here is an example: 764403040417
export AWS_REGION="***"  # you can use any AWS region you'd like, e.g. "us-west-2"
export AWS_ACCESS_KEY_ID="***"
export AWS_SECRET_ACCESS_KEY="***"

export GCP_PROJECT_ID="***"
export GOOGLE_APPLICATION_CREDENTIALS="***"  # check the service account permissions here: https://docs.cortex.dev/clusters/gcp/credentials
export GCR_HOST="gcr.io"  # must be "gcr.io", "us.gcr.io", "eu.gcr.io", or "asia.gcr.io"

# export NUM_BUILD_PROCS=2  # optional; can be >2 if you have enough memory
```

Create the ECR registries:

```bash
make registry-create-aws
```

Create `dev/config/cluster-aws.yaml`. Paste the following config, and update `region` and all registry URLs (replace `<account_id>` with your AWS account ID, and replace `<region>` with your region):

```yaml
# dev/config/cluster-aws.yaml

cluster_name: cortex
provider: aws
region: <region>  # e.g. us-west-2
instance_type: m5.large
min_instances: 1
max_instances: 5

image_operator: <account_id>.dkr.ecr.<region>.amazonaws.com/cortexlabs/operator:master
image_manager: <account_id>.dkr.ecr.<region>.amazonaws.com/cortexlabs/manager:master
image_downloader: <account_id>.dkr.ecr.<region>.amazonaws.com/cortexlabs/downloader:master
image_request_monitor: <account_id>.dkr.ecr.<region>.amazonaws.com/cortexlabs/request-monitor:master
image_cluster_autoscaler: <account_id>.dkr.ecr.<region>.amazonaws.com/cortexlabs/cluster-autoscaler:master
image_metrics_server: <account_id>.dkr.ecr.<region>.amazonaws.com/cortexlabs/metrics-server:master
image_inferentia: <account_id>.dkr.ecr.<region>.amazonaws.com/cortexlabs/inferentia:master
image_neuron_rtd: <account_id>.dkr.ecr.<region>.amazonaws.com/cortexlabs/neuron-rtd:master
image_nvidia: <account_id>.dkr.ecr.<region>.amazonaws.com/cortexlabs/nvidia:master
image_fluent_bit: <account_id>.dkr.ecr.<region>.amazonaws.com/cortexlabs/fluent-bit:master
image_istio_proxy: <account_id>.dkr.ecr.<region>.amazonaws.com/cortexlabs/istio-proxy:master
image_istio_pilot: <account_id>.dkr.ecr.<region>.amazonaws.com/cortexlabs/istio-pilot:master
image_prometheus: <account_id>.dkr.ecr.<region>.amazonaws.com/cortexlabs/prometheus:master
image_prometheus_config_reloader: <account_id>.dkr.ecr.<region>.amazonaws.com/cortexlabs/prometheus-config-reloader:master
image_prometheus_operator: <account_id>.dkr.ecr.<region>.amazonaws.com/cortexlabs/prometheus-operator:master
image_prometheus_statsd_exporter: <account_id>.dkr.ecr.<region>.amazonaws.com/cortexlabs/prometheus-statsd-exporter:master
image_grafana: <account_id>.dkr.ecr.<region>.amazonaws.com/cortexlabs/grafana:master
image_event_exporter: <account_id>.dkr.ecr.<region>.amazonaws.com/cortexlabs/event-exporter:master
```

Create `dev/config/cluster-gcp.yaml`. Paste the following config, and update `project`, `zone`, and all registry URLs (replace `<project_id>` with your project ID, and update `gcr.io` if you are using a different host):

```yaml
# dev/config/cluster-gcp.yaml

project: <project_id>
zone: <zone>  # e.g. us-east1-c
cluster_name: cortex
provider: gcp
instance_type: n1-standard-2
min_instances: 1
max_instances: 5
# accelerator_type: nvidia-tesla-k80  # optional

image_operator: /cortexlabs/operator:master
image_manager: gcr.io/<project_id>/cortexlabs/manager:master
image_downloader: gcr.io/<project_id>/cortexlabs/downloader:master
image_request_monitor: gcr.io/<project_id>/cortexlabs/request-monitor:master
image_istio_proxy: gcr.io/<project_id>/cortexlabs/istio-proxy:master
image_istio_pilot: gcr.io/<project_id>/cortexlabs/istio-pilot:master
image_google_pause: gcr.io/<project_id>/cortexlabs/google-pause:master
image_prometheus: gcr.io/<project_id>/cortexlabs/prometheus:master
image_prometheus_config_reloader: gcr.io/<project_id>/cortexlabs/prometheus-config-reloader:master
image_prometheus_operator: gcr.io/<project_id>/cortexlabs/prometheus-operator:master
image_prometheus_statsd_exporter: gcr.io/<project_id>/cortexlabs/prometheus-statsd-exporter:master
image_grafana: gcr.io/<project_id>/cortexlabs/grafana:master
image_event_exporter: gcr.io/<project_id>/cortexlabs/event-exporter:master
```

### Building

Add this to your bash profile (e.g. `~/.bash_profile`, `~/.profile` or `~/.bashrc`), replacing the placeholders accordingly:

```bash
# set the default image for APIs
export CORTEX_DEV_DEFAULT_PREDICTOR_IMAGE_REGISTRY_AWS="<account_id>.dkr.ecr.<region>.amazonaws.com/cortexlabs"
export CORTEX_DEV_DEFAULT_PREDICTOR_IMAGE_REGISTRY_GCP="gcr.io/<project_id>/cortexlabs"
export CORTEX_DEV_DEFAULT_PREDICTOR_IMAGE_REGISTRY="cortexlabs"

# redirect analytics and error reporting to our dev environment
export CORTEX_TELEMETRY_SENTRY_DSN="https://c334df915c014ffa93f2076769e5b334@sentry.io/1848098"
export CORTEX_TELEMETRY_SEGMENT_WRITE_KEY="0WvoJyCey9z1W2EW7rYTPJUMRYat46dl"

# instruct the Python client to use your development CLI binary (update the path to point to your cortex repo)
export CORTEX_CLI_PATH="<cortex_repo_path>/bin/cortex"

# create a cortex alias which runs your development CLI
alias cortex="$CORTEX_CLI_PATH"
```

Refresh your bash profile:

```bash
. ~/.bash_profile  # or: `. ~/.bashrc`
```

Build the Cortex CLI:

```bash
make cli  # the binary will be placed in <path/to/cortex>/bin/cortex
cortex version  # should show "master"
```

Build and push all Cortex images:

```bash
# for AWS:
make images-all-aws

# for GCP:
make images-all-gcp
```

## Dev workflow

Here is the typical full dev workflow which covers most cases (replace `aws` with `gcp` if desired):

1. `make cluster-up-aws` (creates a cluster using `dev/config/cluster-aws.yaml`)
2. `make devstart-aws` (deletes the in-cluster operator, builds the CLI, and starts the operator locally; file changes will trigger the CLI and operator to re-build)
3. Make your changes
4. `make images-dev-aws` (only necessary if API images or the manager are modified)
5. Test your changes e.g. via `cortex deploy` (and repeat steps 3 and 4 as necessary)
6. `make cluster-down-aws` (deletes your cluster)

If you want to switch back to the in-cluster operator:

1. `<ctrl+c>` to stop your local operator
2. `make operator-start-aws` to restart the operator in your cluster

### Dev workflow optimizations

If you are only modifying the CLI, `make cli-watch` will build the CLI and re-build it when files are changed. When doing this, you can leave the operator running in the cluster instead of running it locally.

If you are only modifying the operator, `make operator-local-aws` will build and start the operator locally, and build/restart it when files are changed.

If you are modifying code in the API images (i.e. any of the Python serving code), `make images-dev-aws` may build more images than you need during testing. For example, if you are only testing using the `python-predictor-cpu` image, you can run `./dev/registry.sh update-single python-predictor-cpu --provider aws`.

See `Makefile` for additional dev commands.
