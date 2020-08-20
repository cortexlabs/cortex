# System packages

_WARNING: you are on the master branch, please refer to the docs on the branch that matches your `cortex version`_

## Bash script

Cortex looks for a file named `dependencies.sh` in the top level Cortex project directory (i.e. the directory which contains `cortex.yaml`). For example:

```text
./my-classifier/
├── cortex.yaml
├── predictor.py
├── ...
└── dependencies.sh
```

`dependencies.sh` is executed with `bash` shell during the initialization of each replica (before installing Python packages in `requirements.txt` or `conda-packages.txt`). Typical use cases include installing required system packages to be used in your Predictor, building Python packages from source, etc.

Here is an example `dependencies.sh`, which installs the `tree` utility:

```bash
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

You can also build a custom Docker image for use in your APIs, e.g. to avoid installing dependencies during replica initialization.

### Create a Dockerfile

Create a Dockerfile to build your custom image:

```bash
mkdir my-api && cd my-api && touch Dockerfile
```

Cortex's base Docker images are listed below. Depending on the Cortex Predictor and compute type specified in your API configuration, choose one of these images to use as the base for your Docker image:

<!-- CORTEX_VERSION_BRANCH_STABLE x6 -->
* Python Predictor (CPU): `cortexlabs/python-predictor-cpu-slim:master`
* Python Predictor (GPU): `cortexlabs/python-predictor-gpu-slim:master-cuda10.1` (also available in cuda10.0, cuda10.2, and cuda11.0)
* Python Predictor (Inferentia): `cortexlabs/python-predictor-inf-slim:master`
* TensorFlow Predictor (CPU, GPU, Inferentia): `cortexlabs/tensorflow-predictor-slim:master`
* ONNX Predictor (CPU): `cortexlabs/onnx-predictor-cpu-slim:master`
* ONNX Predictor (GPU): `cortexlabs/onnx-predictor-gpu-slim:master`

Note: the images listed above use the `-slim` suffix; Cortex's default API images are not `-slim`, since they have additional dependencies installed to cover common use cases. If you are building your own Docker image, starting with a `-slim` Predictor image will result in a smaller image size.

The sample Dockerfile below inherits from Cortex's Python CPU serving image, and installs 3 packages. `tree` is a system package and `pandas` and `rdkit` are Python packages.

<!-- CORTEX_VERSION_BRANCH_STABLE -->
```dockerfile
# Dockerfile

FROM cortexlabs/python-predictor-cpu-slim:master

RUN apt-get update \
    && apt-get install -y tree \
    && apt-get clean && rm -rf /var/lib/apt/lists/*

RUN pip install --no-cache-dir pandas \
    && conda install -y conda-forge::rdkit \
    && conda clean -a
```

### Build and push to a container registry

Create a repository to store your image:

```bash
# We create a repository in ECR

export AWS_ACCESS_KEY_ID="***"
export AWS_SECRET_ACCESS_KEY="***"

eval $(aws ecr get-login --no-include-email --region us-east-1)

aws ecr create-repository --repository-name=org/my-api --region=us-east-1
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

*Note: for [TensorFlow Predictors](#tensorflow-predictor), two containers run together to serve predictions: one runs your Predictor code (`cortexlabs/tensorflow-predictor`), and the other is TensorFlow serving to load the SavedModel (`cortexlabs/tensorflow-serving-gpu` or `cortexlabs/tensorflow-serving-cpu`). There's a second available field `tensorflow_serving_image` that can be used to override the TensorFlow Serving image. Both of the default serving images (`cortexlabs/tensorflow-serving-gpu` and `cortexlabs/tensorflow-serving-cpu`) are based on the official TensorFlow Serving image (`tensorflow/serving`). Unless a different version of TensorFlow Serving is required, the TensorFlow Serving image shouldn't have to be overridden, since it's only used to load the SavedModel and does not run your Predictor code.*

Deploy your API as usual:

```bash
cortex deploy
```
