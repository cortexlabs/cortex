# System Packages

Cortex uses Docker images to run workloads. These Docker images can be replaced with custom images based on Cortex images and augmented with your system packages and libraries. Your custom images need to live in a container registry (Docker Hub, ECR, GCR) that can be accessed by your cluster.

See `Image paths` section in [cortex config](../cluster/config.md) for all images that can be customized.

The example below showcases how to create a custom Docker image and configure Cortex to use it.

## Create Custom Image

Create a Dockerfile to build your custom image:

```
mkdir my-api && cd my-api && touch Dockerfile
```

Specify the base image you want to override followed by your customizations. The sample Dockerfile below inherits from the ONNX Serving image and installs the `imagemagick` system package.

```
# Dockerfile

FROM cortexlabs/onnx-serve

RUN apt-get update \
    && apt-get install -y imagemagick \
    && apt-get clean && rm -rf /var/lib/apt/lists/*
```

## Build and Push to a Container Registry

Create a repository to store your image:

```
# We create a repository in AWS ECR

export AWS_ACCESS_KEY_ID="***"
export AWS_SECRET_ACCESS_KEY="***"

# take note of repository url
eval $(aws ecr get-login --no-include-email --region us-west-2)
aws ecr create-repository --repository-name=org/my-api --region=us-west-2
```

Build your image based on your Dockerfile and push to its repository in AWS ECR.

```
docker build . -t org/my-api:latest -t <repository url>:latest

docker push <repository url>:latest
```

## Configure Cortex

Set the environment variable of the image to your `my-api` image repository url:

```
export CORTEX_IMAGE_ONNX_SERVE="<repository url>:latest"

./cortex.sh update
```

## Use System Package in Workloads

Cortex will use your image to launch ONNX serving workloads. You will have access to any customizations you made.

```
# request_handler.py
import subprocess

def pre_inference(sample, metadata):
    subprocess.run(["magick", "identify", sample["image_1"]])
    ...
```
