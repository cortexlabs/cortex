# Docker images

Cortex includes a default set of Docker images with pre-installed Python and system packages but you can build custom images for use in your APIs. Common reasons to do this are to avoid installing dependencies during replica initialization, to have smaller images, and/or to mirror images to your cloud's container registry (for speed and reliability).

## Create a Dockerfile

```bash
mkdir my-api && cd my-api && touch Dockerfile
```

Cortex's base Docker images are listed below. Depending on the Cortex Predictor and compute type specified in your API configuration, choose one of these images to use as the base for your Docker image:

<!-- CORTEX_VERSION_BRANCH_STABLE x12 -->
* Python Predictor (CPU): `quay.io/cortexlabs/python-predictor-cpu:0.31.1`
* Python Predictor (GPU): choose one of the following:
  * `quay.io/cortexlabs/python-predictor-gpu:0.31.1-cuda10.0-cudnn7`
  * `quay.io/cortexlabs/python-predictor-gpu:0.31.1-cuda10.1-cudnn7`
  * `quay.io/cortexlabs/python-predictor-gpu:0.31.1-cuda10.1-cudnn8`
  * `quay.io/cortexlabs/python-predictor-gpu:0.31.1-cuda10.2-cudnn7`
  * `quay.io/cortexlabs/python-predictor-gpu:0.31.1-cuda10.2-cudnn8`
  * `quay.io/cortexlabs/python-predictor-gpu:0.31.1-cuda11.0-cudnn8`
  * `quay.io/cortexlabs/python-predictor-gpu:0.31.1-cuda11.1-cudnn8`
* Python Predictor (Inferentia): `quay.io/cortexlabs/python-predictor-inf:0.31.1`
* TensorFlow Predictor (CPU, GPU, Inferentia): `quay.io/cortexlabs/tensorflow-predictor:0.31.1`
* ONNX Predictor (CPU): `quay.io/cortexlabs/onnx-predictor-cpu:0.31.1`
* ONNX Predictor (GPU): `quay.io/cortexlabs/onnx-predictor-gpu:0.31.1`

The sample `Dockerfile` below inherits from Cortex's Python CPU serving image, and installs 3 packages. `tree` is a system package and `pandas` and `rdkit` are Python packages.

<!-- CORTEX_VERSION_BRANCH_STABLE -->
```dockerfile
# Dockerfile

FROM quay.io/cortexlabs/python-predictor-cpu:0.31.1

RUN apt-get update \
    && apt-get install -y tree \
    && apt-get clean && rm -rf /var/lib/apt/lists/*

RUN pip install --no-cache-dir pandas \
    && conda install -y conda-forge::rdkit \
    && conda clean -a
```

If you need to upgrade the Python Runtime version on your image, you can follow this procedure:

<!-- CORTEX_VERSION_BRANCH_STABLE -->

```Dockerfile
# Dockerfile

FROM quay.io/cortexlabs/python-predictor-cpu:0.31.1

# upgrade python runtime version
RUN conda update -n base -c defaults conda
RUN conda install -n env python=3.8.5

# re-install cortex core dependencies
RUN /usr/local/cortex/install-core-dependencies.sh

# ...
```

## Build your image

```bash
docker build . -t org/my-api:latest
```

## Push your image to a container registry

You can push your built Docker image to a public registry of your choice (e.g. Docker Hub), or to a private registry on ECR or Docker Hub.

For example, to use ECR, first create a repository to store your image:

```bash
# We create a repository in ECR

export AWS_REGION="***"
export AWS_ACCESS_KEY_ID="***"
export AWS_SECRET_ACCESS_KEY="***"
export REGISTRY_URL="***"  # this will be in the format "<aws_account_id>.dkr.ecr.<aws_region>.amazonaws.com"

aws ecr get-login-password --region $AWS_REGION | docker login --username AWS --password-stdin $REGISTRY_URL

aws ecr create-repository --repository-name=org/my-api --region=$AWS_REGION
# take note of repository url
```

Build and tag your image, and push it to your ECR repository:

```bash
docker build . -t org/my-api:latest -t <repository_url>:latest

docker push <repository_url>:latest
```

## Configure your API

```yaml
# cortex.yaml

- name: my-api
  ...
  predictor:
    image: <repository_url>:latest
  ...
```

Note: for TensorFlow Predictors, two containers run together to serve predictions: one runs your Predictor code (`quay.io/cortexlabs/tensorflow-predictor`), and the other is TensorFlow serving to load the SavedModel (`quay.io/cortexlabs/tensorflow-serving-gpu` or `quay.io/cortexlabs/tensorflow-serving-cpu`). There's a second available field `tensorflow_serving_image` that can be used to override the TensorFlow Serving image. Both of the default serving images (`quay.io/cortexlabs/tensorflow-serving-gpu` and `quay.io/cortexlabs/tensorflow-serving-cpu`) are based on the official TensorFlow Serving image (`tensorflow/serving`). Unless a different version of TensorFlow Serving is required, the TensorFlow Serving image shouldn't have to be overridden, since it's only used to load the SavedModel and does not run your Predictor code.
