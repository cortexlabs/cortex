# Python APIs

_WARNING: you are on the master branch, please refer to the docs on the branch that matches your `cortex version`_

You can deploy models from any Python framework by defining a class that implements Cortex's Python Predictor interface. The class constructor is responsible for preparing the model for serving, downloading vocabulary files, etc. The `predict()` class function is called on every request and is responsible for responding with a prediction.

In addition to supporting Python models via the Python Predictor interface, Cortex can serve the following exported model formats:

- [TensorFlow](tensorflow.md)
- [ONNX](onnx.md)

## Configuration

```yaml
- name: <string>  # API name (required)
  endpoint: <string>  # the endpoint for the API (default: /<api_name>)
  predictor:
    type: python
    path: <string>  # path to a python file with a PythonPredictor class definition, relative to the Cortex root (required)
    config: <string: value>  # dictionary passed to the constructor of a Predictor (optional)
    python_path: <string>  # path to the root of your Python folder that will be appended to PYTHONPATH (default: folder containing cortex.yaml)
    env: <string: string>  # dictionary of environment variables
  tracker:
    key: <string>  # the JSON key in the response to track (required if the response payload is a JSON object)
    model_type: <string>  # model type, must be "classification" or "regression" (required)
  compute:
    cpu: <string | int | float>  # CPU request per replica (default: 200m)
    gpu: <int>  # GPU request per replica (default: 0)
    mem: <string>  # memory request per replica (default: Null)
  autoscaling:
    min_replicas: <int>  # minimum number of replicas (default: 1)
    max_replicas: <int>  # maximum number of replicas (default: 100)
    init_replicas: <int>  # initial number of replicas (default: <min_replicas>)
    workers_per_replica: <int>  # the number of parallel serving workers to run on each replica (default: 1)
    threads_per_worker: <int>  # the number of threads per worker (default: 1)
    target_replica_concurrency: <float>  # the desired number of in-flight requests per replica, which the autoscaler tries to maintain (default: workers_per_replica * threads_per_worker)
    max_replica_concurrency: <int>  # the maximum number of in-flight requests per replica before requests are rejected with error code 503 (default: 1024)
    window: <duration>  # the time over which to average the API's concurrency (default: 60s)
    downscale_stabilization_period: <duration>  # the API will not scale below the highest recommendation made during this period (default: 5m)
    upscale_stabilization_period: <duration>  # the API will not scale above the lowest recommendation made during this period (default: 0m)
    max_downscale_factor: <float>  # the maximum factor by which to scale down the API on a single scaling event (default: 0.5)
    max_upscale_factor: <float>  # the maximum factor by which to scale up the API on a single scaling event (default: 10)
    downscale_tolerance: <float>  # any recommendation falling within this factor below the current number of replicas will not trigger a scale down event (default: 0.1)
    upscale_tolerance: <float>  # any recommendation falling within this factor above the current number of replicas will not trigger a scale up event (default: 0.1)
  update_strategy:
    max_surge: <string | int>  # maximum number of replicas that can be scheduled above the desired number of replicas during an update; can be an absolute number, e.g. 5, or a percentage of desired replicas, e.g. 10% (default: 25%)
    max_unavailable: <string | int>  # maximum number of replicas that can be unavailable during an update; can be an absolute number, e.g. 5, or a percentage of desired replicas, e.g. 10% (default: 25%)
```

### Example

```yaml
- name: my-api
  predictor:
    type: python
    path: predictor.py
  compute:
    gpu: 1
```

## Debugging

You can log information about each request by adding a `?debug=true` parameter to your requests. This will print:

1. The payload
2. The value after running the `predict` function

# Python Predictor

A Python Predictor is a Python class that describes how to initialize a model and use it to make a prediction.

The lifecycle of a replica starts with the initialization of the Python Predictor class defined in your implementation file. The constructor is responsible for downloading and initializing the model. It receives the config object, which is an arbitrary dictionary defined in the API configuration (e.g. `cortex.yaml`) that can be used to pass in the path to the exported model, vocabularies, etc. After successfully initializing an instance of the Python Predictor class, the replica is available to serve requests. Upon receiving a request, the replica calls the `predict()` function with the JSON payload. The `predict()` function is responsible for returning a prediction or a batch of predictions. Preprocessing of the JSON payload and postprocessing of predictions can be implemented in your `predict()` function.

## Implementation

```python
# initialization code and variables can be declared here in global scope

class PythonPredictor:
    def __init__(self, config):
        """Called once before the API becomes available. Setup for model serving such as downloading/initializing the model or downloading vocabulary can be done here. Required.

        Args:
            config: Dictionary passed from API configuration (if specified).
        """
        pass

    def predict(self, payload):
        """Called once per request. Runs preprocessing of the request payload, inference, and postprocessing of the inference output. Required.

        Args:
            payload: The parsed JSON request payload.

        Returns:
            Prediction or a batch of predictions.
        """
```

## Example

```python
import boto3
from my_model import IrisNet

labels = ["setosa", "versicolor", "virginica"]

class PythonPredictor:
    def __init__(self, config):
        # download the model
        bucket, key = re.match("s3://(.+?)/(.+)", config["model"]).groups()
        s3 = boto3.client("s3")
        s3.download_file(bucket, key, "model.pth")

        # initialize the model
        model = IrisNet()
        model.load_state_dict(torch.load(config['model']))
        model.eval()

        self.model = model


    def predict(self, payload):
        # Convert the request to a tensor and pass it into the model
        input_tensor = torch.FloatTensor(
            [
                [
                    payload["sepal_length"],
                    payload["sepal_width"],
                    payload["petal_length"],
                    payload["petal_width"],
                ]
            ]
        )

        # Run the prediction
        output = self.model(input_tensor)

        # Translate the model output to the corresponding label string
        return labels[torch.argmax(output[0])]
```

## Pre-installed packages

The following packages have been pre-installed and can be used in your implementations:

```text
boto3==1.10.45
cloudpickle==1.3.0
dill==0.3.1.1
joblib==0.14.1
Keras==2.3.1
msgpack==0.6.2
nltk==3.4.5
np-utils==0.5.12.1
numpy==1.18.0
pandas==0.25.3
opencv-python==4.1.2.30
Pillow==6.2.1
pyyaml==5.3
requests==2.22.0
scikit-image==0.16.2
scikit-learn==0.22
scipy==1.4.1
six==1.13.0
statsmodels==0.10.2
sympy==1.5
tensor2tensor==1.15.4
tensorflow-hub==0.7.0
tensorflow==2.1.0
torch==1.4.0
torchvision==0.5.0
xgboost==0.90
```

Learn how to install additional packages [here](../dependency-management/python-packages.md).
