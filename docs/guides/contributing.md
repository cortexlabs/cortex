# Contributing

## Remote development

We recommend that you run your development environment on a cloud instance due to frequent docker registry pushing, e.g. an AWS EC2 instance or GCP VM. We've had a good experience using [Mutagen](https://mutagen.io/documentation/introduction) to synchronize local / remote file systems. Feel free to reach out to us on [gitter](https://gitter.im/cortexlabs/cortex) if you have any questions about this.

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

Create a config directory in the repo's root directory:

```bash
mkdir dev/config
```

Create `dev/config/env.sh` with the following information:

```bash
# dev/config/env.sh

export AWS_ACCOUNT_ID="***"  # you can find your account ID in the AWS web console; here is an example: 764403040417
export AWS_REGION="us-west-2"  # you can use any AWS region you'd like
export AWS_ACCESS_KEY_ID="***"
export AWS_SECRET_ACCESS_KEY="***"
# export NUM_BUILD_PROCS=2  # optional; can be >2 if you have enough memory
```

Create the ECR registries:

```bash
make registry-create-aws
```

Create `dev/config/cluster-aws.yaml`. Paste the following config, and update `cortex_region` and all registry URLs (replace `XXXXXXXX` with your AWS account ID, and update the region):

```yaml
# dev/config/cluster-aws.yaml

cluster_name: cortex
provider: aws
region: us-west-2
instance_type: m5.large
min_instances: 1
max_instances: 5

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
```

### Building

Add this to your bash profile (e.g. `~/.bash_profile`, `~/.profile` or `~/.bashrc`), replacing the image registry URL accordingly:

```bash
export CORTEX_DEV_DEFAULT_PREDICTOR_IMAGE_REGISTRY_AWS="XXXXXXXX.dkr.ecr.us-west-2.amazonaws.com/cortexlabs"  # set the default image for APIs
export CORTEX_TELEMETRY_SENTRY_DSN="https://c334df915c014ffa93f2076769e5b334@sentry.io/1848098"  # redirect analytics to our dev environment
export CORTEX_TELEMETRY_SEGMENT_WRITE_KEY="0WvoJyCey9z1W2EW7rYTPJUMRYat46dl"  # redirect error reporting to our dev environment

alias cortex='$HOME/bin/cortex'  # your path may be different depending on where you cloned the repo
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
make images-all-aws
```

## Dev workflow

Here is the typical full dev workflow which covers most cases:

1. `make cluster-up-aws` (creates a cluster using `dev/config/cluster-aws.yaml`)
2. `make devstart-aws` (deletes the in-cluster operator, builds the CLI, and starts the operator locally; file changes will trigger the CLI and operator to re-build)
3. Make your changes
4. `make images-dev-aws` (only necessary if API images or the manager are modified)
5. Test your changes e.g. via `cortex deploy` (and repeat steps 3 and 4 as necessary)
6. `make cluster-down-aws` (deletes your cluster)

If you want to switch back to the in-cluster operator:

1. `<ctrl+c>` to stop your local operator
2. `make cluster-configure-aws` to install the operator in your cluster

If you only want to test Cortex's local environment, here is the common workflow:

1. `make cli-watch` (builds the CLI and re-builds it when files are changed)
2. Make your changes
3. `make images-dev-local` (only necessary if API images or the manager are modified)
4. Test your changes e.g. via `cortex deploy` (and repeat steps 2 and 3 as necessary)

### Dev workflow optimizations

If you are only modifying the CLI, `make cli-watch` will build the CLI and re-build it when files are changed. When doing this, you can leave the operator running in the cluster instead of running it locally.

If you are only modifying the operator, `make operator-local-aws` will build and start the operator locally, and build/restart it when files are changed.

If you are modifying code in the API images (i.e. any of the Python serving code), `make images-dev-aws` may build more images than you need during testing. For example, if you are only testing using the `python-predictor-cpu` image, you can run `./dev/registry.sh update-single python-predictor-cpu --provider aws` (or use `--provider local` if testing locally).

See `Makefile` for additional dev commands.

Feel free to [chat with us](https://gitter.im/cortexlabs/cortex) if you have any questions.
