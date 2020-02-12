# ONNX APIs

_WARNING: you are on the master branch, please refer to the docs on the branch that matches your `cortex version`_

You can deploy ONNX models as web services by defining a class that implements Cortex's ONNX Predictor interface.

## Config

```yaml
- name: <string>  # API name (required)
  endpoint: <string>  # the endpoint for the API (default: /<api_name>)
  predictor:
    type: onnx
    path: <string>  # path to a python file with an ONNXPredictor class definition, relative to the Cortex root (required)
    model: <string>  # S3 path to an exported model (e.g. s3://my-bucket/exported_model.onnx) (required)
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
    workers_per_replica: <int>  # the number of parallel serving workers to run on each replica (default: 4)
    threads_per_worker: <int>  # the number of threads per worker (default: 1)
    request_backlog: <int>  # maximum number of pending connections per replica
    target_queue_length: <float>  # the desired queue length per replica (default: 0)
    window: <duration>  # the time over which to average the API's queue length (default: 60s)
    # tick: <duration>  # the time between each execution of the autoscaler  # TODO maybe don't make this configurable
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

See [packaging ONNX models](../packaging-models/onnx.md) for information about exporting ONNX models.

## Example

```yaml
- name: my-api
  predictor:
    type: onnx
    path: predictor.py
    model: s3://my-bucket/my-model.onnx
  compute:
    gpu: 1
```

## Debugging

You can log information about each request by adding a `?debug=true` parameter to your requests. This will print:

1. The payload
2. The value after running the `predict` function

# ONNX Predictor

An ONNX Predictor is a Python class that describes how to serve your ONNX model to make predictions.

<!-- CORTEX_VERSION_MINOR -->
Cortex provides an `onnx_client` and a config object to initialize your implementation of the ONNX Predictor class. The `onnx_client` is an instance of [ONNXClient](https://github.com/cortexlabs/cortex/tree/master/pkg/workloads/cortex/lib/client/onnx.py) that manages an ONNX Runtime session and helps make predictions using your model. Once your implementation of the ONNX Predictor class has been initialized, the replica is available to serve requests. Upon receiving a request, your implementation's `predict()` function is called with the JSON payload and is responsible for returning a prediction or batch of predictions. Your `predict()` function should call `onnx_client.predict()` to make an inference against your exported ONNX model. Preprocessing of the JSON payload and postprocessing of predictions can be implemented in your `predict()` function as well.

## Implementation

```python
class ONNXPredictor:
    def __init__(self, onnx_client, config):
        """Called once before the API becomes available. Setup for model serving such as downloading/initializing vocabularies can be done here. Required.

        Args:
            onnx_client: ONNX client which can be used to make predictions.
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
import numpy as np

labels = ["setosa", "versicolor", "virginica"]


class ONNXPredictor:
    def __init__(self, onnx_client, config):
        self.client = onnx_client

    def predict(self, payload):
        model_input = [
            payload["sepal_length"],
            payload["sepal_width"],
            payload["petal_length"],
            payload["petal_width"],
        ]

        prediction = self.client.predict(model_input)
        predicted_class_id = prediction[0][0]
        return labels[predicted_class_id]
```

## Pre-installed packages

The following packages have been pre-installed and can be used in your implementations:

```text
boto3==1.10.45
dill==0.3.1.1
msgpack==0.6.2
numpy==1.18.0
onnxruntime==1.1.0
requests==2.22.0
```

Learn how to install additional packages [here](../dependency-management/python-packages.md).
