# System packages

_WARNING: you are on the master branch, please refer to the docs on the branch that matches your `cortex version`_

## Bash script

Cortex looks inside the root directory of the project for a file named `dependencies.sh`. (i.e. the directory which contains `cortex.yaml`).

```text
./iris-classifier/
├── cortex.yaml
├── predictor.py
├── ...
└── dependencies.sh
```

This `dependencies.sh` gets executed during the initialization of each replica. Typical use cases include installing required system packages to be used in Predictor, building python packages from source, etc.

Sample `dependencies.sh` installing `tree` utility:
```bash
#!/bin/bash
apt-get update && apt-get install -y tree
```

The `tree` utility can now be called inside your `predictor.py`:

```python
# predictor.py
import subprocess

class PythonPredictor:
    def __init__(self, config):
        subprocess.run(["tree"])
    ...
```

## Custom Docker image

### Create a Dockerfile

Create a Dockerfile to build your custom image:

```bash
mkdir my-api && cd my-api && touch Dockerfile
```

The default Docker images used to deploy your models are listed below. Based on the Cortex Predictor and compute type specified in your API configuration, choose a Cortex image to use as the base for your custom Docker image:

<!-- CORTEX_VERSION_BRANCH_STABLE x5 -->
* Python Predictor (CPU): `cortexlabs/python-serve:master`
* Python Predictor (GPU): `cortexlabs/python-serve-gpu:master`
* TensorFlow Predictor (CPU and GPU): `cortexlabs/tensorflow-predictor:master`
* ONNX Predictor (CPU): `cortexlabs/onnx-serve:master`
* ONNX Predictor (GPU): `cortexlabs/onnx-serve-gpu:master`

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

Build the image based on your Dockerfile and push it to its repository in ECR:

```bash
docker build . -t org/my-api:latest -t <repository_url>:latest

docker push <repository_url>:latest
```

### Configure Cortex

Update your API configuration file to point to your image:

```yaml
# cortex.yaml

- name: my-api
  ...
  predictor:
    image: <repository_url>:latest
  ...
```

*Note: for [TensorFlow Predictors](#tensorflow-predictor), two containers run together serve predictions: one which runs your Predictor code (`cortexlabs/tensorflow-predictor`), and TensorFlow Serving which loads the SavedModel (`cortexlabs/tensorflow-serving-cpu[-gpu]`). There's a 2nd available field `tensorflow_serving_image` that can be used to override the TensorFlow Serving image. The default image (`cortexlabs/tensorflow-serving-cpu[-gpu]`) is based on the official TensorFlow Serving image (`tensorflow/serving`). Unless a different version of TensorFlow Serving is required, this image shouldn't have to be overridden, since it's only used to load the SavedModel and does not run your Predictor code.*

Deploy your API as usual:

```bash
cortex deploy
```
