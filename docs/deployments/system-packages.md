# System packages

_WARNING: you are on the master branch, please refer to the docs on the branch that matches your `cortex version`_

Cortex uses Docker images to deploy your models. There are 2 ways to augment your system packages and libraries:

1. **Using `script.sh`** in the root directory of the project. Support is already included in Cortex.
1. Creating a **custom Docker image**. Must push to a container registry that the cluster has access to (i.e. [Docker Hub](https://hub.docker.com) or [AWS ECR](https://aws.amazon.com/ecr)).

So, let's explore an example where we use `tree` program to list files inside the `predictory.py` module. We want to be able to run this:

```python
# predictor.py

import subprocess

class PythonPredictor:
    def __init__(self, config):
        subprocess.run(["tree"])

    def predict(self, payload):
        return None
```
The above example works with either of the following methods.

---

## Using a script

Cortex looks inside the root directory of the project for a file named `script.sh`. (i.e. the directory which contains `cortex.yaml`). This `script.sh` gets executed at run-time when an API is being deployed.

Here's what `script.sh` contains in order to install `tree` utility:
```bash
#!/bin/bash
apt-get update && apt-get install -y tree
```

*Note: The order of execution on all files is this: `script.sh` -> `.condarc` -> `environment.yaml` or `conda-packages.txt` -> `requirements.txt`.*

---

## Using a custom Docker image

Create a Dockerfile to build your custom image:

```bash
mkdir my-api && cd my-api && touch Dockerfile
```

The Docker images used to deploy your models are listed below. Based on the Cortex Predictor and compute type specified in your API configuration, choose a Cortex image to use as the base for your custom Docker image.

### Base Cortex images for model serving

<!-- CORTEX_VERSION_BRANCH_STABLE x5 -->
* Python (CPU): `cortexlabs/python-serve:master`
* Python (GPU): `cortexlabs/python-serve-gpu:master`
* TensorFlow (CPU or GPU): `cortexlabs/tf-api:master`
* ONNX (CPU): `cortexlabs/onnx-serve:master`
* ONNX (GPU): `cortexlabs/onnx-serve-gpu:master`

Note that the Docker image version must match your cluster version displayed in `cortex version`.

The sample Dockerfile below inherits from Cortex's Python CPU serving image and installs the `tree` system package.

<!-- CORTEX_VERSION_BRANCH_STABLE -->
```dockerfile
# Dockerfile

FROM cortexlabs/python-serve:master

RUN apt-get update \
    && apt-get install -y tree \
    && apt-get clean && rm -rf /var/lib/apt/lists/*
```

### Build and push to a container registry

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

### Configure Cortex

Update your cluster configuration file to point to your image:

```yaml
# cluster.yaml

# ...
image_python_serve: <repository_url>:latest
# ...
```

Update your cluster for the change to take effect:

```bash
cortex cluster update --config=cluster.yaml
```
