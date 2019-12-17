# TensorFlow APIs

_WARNING: you are on the master branch, please refer to the docs on the branch that matches your `cortex version`_

You can deploy TensorFlow models as web services by defining a class that implements Cortex's TensorFlow Predictor interface.

## Config

```yaml
- kind: api
  name: <string>  # API name (required)
  endpoint: <string>  # the endpoint for the API (default: /<deployment_name>/<api_name>)
  tensorflow:
    model: <string>  # S3 path to an exported model (e.g. s3://my-bucket/exported_model) (required)
    predictor: <string>  # path to a python file with a TensorFlowPredictor class definition, relative to the Cortex root (required)
    signature_key: <string>  # name of the signature def to use for prediction (required if your model has more than one signature def)
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

See [packaging TensorFlow models](../packaging-models/tensorflow.md) for how to export a TensorFlow model.

## Example

```yaml
- kind: api
  name: my-api
  tensorflow:
    model: s3://my-bucket/my-model
    predictor: predictor.py
  compute:
    gpu: 1
```

## Debugging

You can log information about each request by adding a `?debug=true` parameter to your requests. This will print:

1. The payload
2. The value after running the `predict` function

# TensorFlow Predictor

A TensorFlow Predictor is a Python class that describes how to serve your TensorFlow model to make predictions.

<!-- CORTEX_VERSION_MINOR -->
Cortex provides a `tensorflow_client` and a config object to initialize your implementation of the TensorFlow Predictor class. The `tensorflow_client` is an instance of [TensorFlowClient](https://github.com/cortexlabs/cortex/tree/master/pkg/workloads/cortex/tf_api/client.py) that manages a connection to a TensorFlow Serving container via gRPC to make predictions using your model. Once your implementation of the TensorFlow Predictor class has been initialized, the replica is available to serve requests. Upon receiving a request, your implementation's `predict()` function is called with the JSON payload and is responsible for returning a prediction or batch of predictions. Your `predict()` function should call `tensorflow_client.predict()` to make an inference against your exported TensorFlow model. Preprocessing of the JSON payload and postprocessing of predictions can be implemented in your `predict()` function as well.

## Implementation

```python
class TensorFlowPredictor:
    def __init__(self, tensorflow_client, config):
        """Called once before the API becomes available. Setup for model serving such as downloading/initializing vocabularies can be done here. Required.

        Args:
            tensorflow_client: TensorFlow client which can be used to make predictions.
            config: Dictionary passed from API configuration in cortex.yaml (if specified).
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
labels = ["iris-setosa", "iris-versicolor", "iris-virginica"]


class TensorFlowPredictor:
    def __init__(self, tensorflow_client, config):
        self.client = tensorflow_client

    def predict(self, payload):
        prediction = self.client.predict(payload)
        predicted_class_id = int(prediction["class_ids"][0])
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
tensor2tensor==1.14.1
tensorflow-hub==0.7.0
tensorflow==2.0.0
```

Learn how to install additional packages [here](../dependency-management/python-packages.md).
