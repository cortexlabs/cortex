# System Packages

Custom system packages can be installed by overriding the default Docker images used by Cortex. Your custom images need to live in a container registry (DockerHub, ECR, GCR) that can be accessed by your cluster.

See `Image paths` section in [cortex config](../cluster/config.md) for all images that can be customized.

The example below showcases how to override a Cortex Docker image and configure Cortex to use it.

## Create Custom Image

```
mkdir my-api && cd my-api && touch Dockerfile
```

In your Dockerfile, specify the base image you want to override followed by your customizations. In this example we install the `tree` system package.


```
FROM cortexlabs/onnx-serve

RUN apt-get update \
    && apt-get install -y tree \
    && apt-get clean && rm -rf /var/lib/apt/lists/*
```

## Build and Push to a Container Registry

We will build a new image called `my-api` and push it to a repository in a container registry.
```
# We will be pushing to AWS ECR in this example

export AWS_ACCESS_KEY_ID="***"
export AWS_SECRET_ACCESS_KEY="***"

# take note of repository url
eval $(aws ecr get-login --no-include-email --region us-west-2)
aws ecr create-repository --repository-name=org/my-api --region=us-west-2


docker build . -t org/my-api:latest -t <repository url>:latest

docker push <repository url>:latest
```

## Configure Cortex

Set the environment variable of image you want to customize to your `my-api` image url.
```
export CORTEX_IMAGE_ONNX_SERVE="<repository url>:latest"

./cortex.sh update
```

## Use System Package

Cortex will now your image, instead of the default image.
```
# request_handler.py
import subprocess

def pre_inference(sample, metadata):
    subprocess.run(["tree"])
    ...
```
