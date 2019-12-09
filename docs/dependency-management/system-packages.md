# System packages

Cortex uses Docker images to deploy your models. These images can be replaced with custom images that you can augment with your system packages and libraries. You will need to push your custom images to a container registry that your cluster has access to (e.g. [Docker Hub](https://hub.docker.com) or [AWS ECR](https://aws.amazon.com/ecr)).

See the `image paths` section in [cluster configuration](../cluster-management/config.md) for a complete list of customizable images.

## Create a custom image

Create a Dockerfile to build your custom image:

```bash
mkdir my-api && cd my-api && touch Dockerfile
```

Specify the base image you want to override followed by your customizations. The sample Dockerfile below inherits from Cortex's Python serving image and installs the `tree` system package.

```dockerfile
# Dockerfile

FROM cortexlabs/predictor-serve

RUN apt-get update \
    && apt-get install -y tree \
    && apt-get clean && rm -rf /var/lib/apt/lists/*
```

## Build and push to a container registry

Create a repository to store your image:

```bash
# We create a repository in ECR

export AWS_ACCESS_KEY_ID="***"
export AWS_SECRET_ACCESS_KEY="***"

eval $(aws ecr get-login --no-include-email --region us-west-2)

aws ecr create-repository --repository-name=org/my-api --region=us-west-2
# take note of repository url
```

Build the image based on your Dockerfile and push to its repository in ECR:

```bash
docker build . -t org/my-api:latest -t <repository_url>:latest

docker push <repository_url>:latest
```

## Configure Cortex

Update your cluster configuration file to point to your image:

```yaml
# cluster.yaml

# ...
image_predictor_serve: <repository_url>:latest
# ...
```

Update your cluster for the change to take effect:

```bash
cortex cluster update --config=cluster.yaml
```

## Use system packages in workloads

Cortex will use your image to launch Python serving workloads and you will have access to any packages you added:

```python
# predictor.py

import subprocess

class Predictor:
    def init(self, config):
        subprocess.run(["tree"])
        ...
```
