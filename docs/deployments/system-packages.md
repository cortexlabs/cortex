# System packages

Cortex uses Docker images to run workloads. These Docker images can be replaced with custom images based on Cortex images and augmented with your system packages and libraries. Your custom images need to be pushed to a container registry (e.g. Docker Hub, ECR, GCR) that can be accessed by your cluster.

See `Image paths` section in [cortex config](../cluster/config.md) for all images that can be customized.

The example below demonstrates how to create a custom Docker image and configure Cortex to use it.

## Create a custom image

Create a Dockerfile to build your custom image:

```bash
mkdir my-api && cd my-api && touch Dockerfile
```

Specify the base image you want to override followed by your customizations. The sample Dockerfile below inherits from Cortex's ONNX Serving image and installs the `tree` system package.

```dockerfile
# Dockerfile

FROM cortexlabs/onnx-serve

RUN apt-get update \
    && apt-get install -y tree \
    && apt-get clean && rm -rf /var/lib/apt/lists/*
```

## Build and push to a container registry

Create a repository to store your image:

```bash
# We create a repository in AWS ECR

export AWS_ACCESS_KEY_ID="***"
export AWS_SECRET_ACCESS_KEY="***"

eval $(aws ecr get-login --no-include-email --region us-west-2)

aws ecr create-repository --repository-name=org/my-api --region=us-west-2
# take note of repository url
```

Build the image based on your Dockerfile and push to its repository in AWS ECR:

```bash
docker build . -t org/my-api:latest -t <repository_url>:latest

docker push <repository_url>:latest
```

## Configure Cortex

Update your cluster configuration file to point to your image:

```yaml
# cluster.yaml

# ...
image_onnx_serve: <repository_url>:latest
# ...
```

And update the cluster for the change to take effect:

```bash
cortex cluster update --config=cluster.yaml
```

## Use system packages in workloads

Cortex will use your image to launch ONNX serving workloads. You will have access to any customizations you made:

```python
# request_handler.py

import subprocess

def pre_inference(sample, metadata):
    subprocess.run(["tree"])
    ...
```
