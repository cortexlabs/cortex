# Contributing

## Remote development

We recommend that you run your development environment on an EC2 instance due to frequent docker registry pushing. We've had a good experience using [Mutagen](https://mutagen.io/documentation/introduction) to synchronize local / remote filesystems.

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
wget https://dl.google.com/go/go1.20.4.linux-amd64.tar.gz && \
sudo tar -xvf go1.20.4.linux-amd64.tar.gz && \
sudo mv go /usr/local && \
rm go1.20.4.linux-amd64.tar.gz && \
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

And then log out and back in. Then, bootstrap a buildx builder:

```bash
docker buildx create --driver-opt image=moby/buildkit:master --name builder --platform linux/amd64,linux/arm64 --use
docker buildx inspect --bootstrap builder
```

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
export AWS_REGION="***"  # you can use any AWS region you'd like, e.g. "us-west-2"
export AWS_ACCESS_KEY_ID="***"  # alternatively, you can remove this to use the default credentials chain on your machine
export AWS_SECRET_ACCESS_KEY="***"  # alternatively, you can remove this to use the default credentials chain on your machine
export DEFAULT_USER_ARN="arn:aws:iam::<ACCOUNT_ID>:<AWS IAM ENTITY>" # (e.g. arn:aws-us-gov:iam::123456789:user/foo)
```

Create the ECR registries:

```bash
make registry-create
```

Create `dev/config/cluster.yaml`. Paste the following config, and update `region` and all registry URLs (replace `<account_id>` with your AWS account ID, and replace `<region>` with your region):

```yaml
# dev/config/cluster.yaml

cluster_name: cortex
region: <region>  # e.g. us-west-2

node_groups:
  - name: worker-ng
    instance_type: m5.large
    min_instances: 1
    max_instances: 5
```

### Building

Add this to your bash profile (e.g. `~/.bash_profile`, `~/.profile` or `~/.bashrc`), replacing the placeholders accordingly:

```bash
# set the default image registry
export CORTEX_DEV_DEFAULT_IMAGE_REGISTRY="<account_id>.dkr.ecr.<region>.amazonaws.com/cortexlabs"

# enable api server monitoring in grafana
export CORTEX_DEV_ADD_CONTROL_PLANE_DASHBOARD="true"

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
make images-all
```

## Dev workflow

Here is the typical full dev workflow which covers most cases:

1. `make cluster-up` (creates a cluster using `dev/config/cluster.yaml`)
2. `make devstart` (deletes the in-cluster operator, builds the CLI, and starts the operator locally; file changes will trigger the CLI and operator to re-build)
3. Make your changes
4. `make images-dev` (only necessary if changes were made outside of the operator and CLI)
5. Test your changes e.g. via `cortex deploy` (and repeat steps 3 and 4 as necessary)
6. `make cluster-down` (deletes your cluster)

If you want to switch back to the in-cluster operator:

1. `<ctrl+c>` to stop your local operator
2. `make operator-start` to restart the operator in your cluster

### Dev workflow optimizations

If you are only modifying the CLI, `make cli-watch` will build the CLI and re-build it when files are changed. When doing this, you can leave the operator running in the cluster instead of running it locally.

If you are only modifying the operator, `make operator-local` will build and start the operator locally, and build/restart it when files are changed.

See `Makefile` for additional dev commands.
