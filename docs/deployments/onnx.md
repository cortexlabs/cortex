# ONNX APIs

_WARNING: you are on the master branch, please refer to the docs on the branch that matches your `cortex version`_

You can deploy ONNX models as web services by defining a class that implements Cortex's ONNX Predictor interface. The constructor is responsible for model serving preparations such downloading vocabulary files. The `predict()` class function is called on every request and is responsible for responding with a prediction.

## Config

```yaml
- kind: api
  name: <string>  # API name (required)
  endpoint: <string>  # the endpoint for the API (default: /<deployment_name>/<api_name>)
  onnx:
    model: <string>  # S3 path to an exported model (e.g. s3://my-bucket/exported_model.onnx)
    predictor: <string>  # path to a python file with an ONNXPredictor class definition, relative to the Cortex root (required)
    config: <string: value>  # dictionary that can be used to configure custom values (optional)
    python_path: <string>  # path to the root of your Python folder that will be appended to PYTHONPATH (default: folder containing cortex.yaml)
  tracker:
    key: <string>  # the JSON key in the response to track (required if the response payload is a JSON object)
    model_type: <string>  # model type, must be "classification" or "regression" (required)
  compute:
    min_replicas: <int>  # minimum number of replicas (default: 1)
    max_replicas: <int>  # maximum number of replicas (default: 100)
    init_replicas: <int>  # initial number of replicas (default: <min_replicas>)
    target_cpu_utilization: <int>  # CPU utilization threshold (as a percentage) to trigger scaling (default: 80)
    cpu: <string | int | float>  # CPU request per replica (default: 200m)
    gpu: <int>  # GPU request per replica (default: 0)
    mem: <string>  # memory request per replica (default: Null)
```

See [packaging ONNX models](../packaging-models/onnx.md) for information about exporting ONNX models.

## Example

```yaml
- kind: api
  name: my-api
  onnx:
    model: s3://my-bucket/my-model.onnx
    predictor: predictor.py
  compute:
    gpu: 1
```

## Debugging

You can log information about each request by adding a `?debug=true` parameter to your requests. This will print:

1. The payload
2. The value after running the `predict` function

# ONNX Predictor

An ONNX Predictor is a Python class that describes how to serve your model to make predictions.

<!-- CORTEX_VERSION -->
Cortex provides an `onnx_client`, an instance of [ONNXClient](https://github.com/cortexlabs/cortex/tree/tf-onnx-api-changes/pkg/workloads/cortex/onnx_serve/client.py), and a config object to initialize your implementation of the ONNX Predictor class. Once your implementation has been initialized, the replica is available to serve requests. Upon receiving a request, your implementation's `predict()` function is called with JSON payload and is responsible for returning a prediction or batch of predictions. Your `predict()` function can call `onnx_client.predict` to make an inference and respond to the request. Preprocessing of the JSON payload, postprocessing of predictions can be implemented in your `predict()` function.


## Implementation

```python
class ONNXPredictor:
    def __init__(self, onnx_client, config):
        """Called once before the API becomes available. Setup for model serving such as downloading/initializing downloading vocabularies can be done here. Required.

        Args:
            onnx_client: ONNX client which can be used to make predictions.
            config: Dictionary passed to the constructor of a Predictor.
        """
        pass

    def predict(self, payload):
        """Called once per request. Runs inference, any preprocessing of the request payload, and postprocessing of the inference output. Required.

        Args:
            payload: The JSON request payload (parsed in Python).

        Returns:
            Prediction or a batch of predictions.
        """
```

## Example

```python
import numpy as np

labels = ["iris-setosa", "iris-versicolor", "iris-virginica"]


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
boto3==1.10.13
dill==0.3.1.1
msgpack==0.6.2
numpy==1.17.3
requests==2.22.0
onnxruntime==0.5.0
```

Learn how to install additional packages [here](../dependency-management/python-packages.md).
