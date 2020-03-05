# TensorFlow APIs

You can deploy TensorFlow models as web services by defining a class that implements Cortex's TensorFlow Predictor interface.

## Config

```yaml
- name: <string>  # API name (required)
  endpoint: <string>  # the endpoint for the API (default: /<api_name>)
  predictor:
    type: tensorflow
    path: <string>  # path to a python file with a TensorFlowPredictor class definition, relative to the Cortex root (required)
    model: <string>  # S3 path to an exported model (e.g. s3://my-bucket/exported_model) (required)
    signature_key: <string>  # name of the signature def to use for prediction (required if your model has more than one signature def)
    config: <string: value>  # dictionary that can be used to configure custom values (optional)
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

See [packaging TensorFlow models](../packaging-models/tensorflow.md) for how to export a TensorFlow model.

## Example

```yaml
- name: my-api
  predictor:
    type: tensorflow
    path: predictor.py
    model: s3://my-bucket/my-model
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
Cortex provides a `tensorflow_client` and a config object to initialize your implementation of the TensorFlow Predictor class. The `tensorflow_client` is an instance of [TensorFlowClient](https://github.com/cortexlabs/cortex/tree/master/pkg/workloads/cortex/lib/client/tensorflow.py) that manages a connection to a TensorFlow Serving container via gRPC to make predictions using your model. Once your implementation of the TensorFlow Predictor class has been initialized, the replica is available to serve requests. Upon receiving a request, your implementation's `predict()` function is called with the JSON payload and is responsible for returning a prediction or batch of predictions. Your `predict()` function should call `tensorflow_client.predict()` to make an inference against your exported TensorFlow model. Preprocessing of the JSON payload and postprocessing of predictions can be implemented in your `predict()` function as well.

## Implementation

```python
class TensorFlowPredictor:
    def __init__(self, tensorflow_client, config):
        """Called once before the API becomes available. Setup for model serving such as downloading/initializing vocabularies can be done here. Required.

        Args:
            tensorflow_client: TensorFlow client which can be used to make predictions.
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
labels = ["setosa", "versicolor", "virginica"]


class TensorFlowPredictor:
    def __init__(self, tensorflow_client, config):
        self.client = tensorflow_client

    def predict(self, payload):
        prediction = self.client.predict(payload)
        predicted_class_id = int(prediction["class_ids"][0])
        return labels[predicted_class_id]
```

## Pre-installed packages

The following Python packages have been pre-installed and can be used in your implementations:

```text
boto3==1.10.45
dill==0.3.1.1
msgpack==0.6.2
numpy==1.18.0
requests==2.22.0
opencv-python==4.1.2.30
pyyaml==5.3
tensor2tensor==1.15.4
tensorflow-hub==0.7.0
tensorflow==2.1.0
```

<!-- CORTEX_VERSION_MINOR -->
The pre-installed system packages are listed in the [tf-api Dockerfile](https://github.com/cortexlabs/cortex/tree/master/images/tf-api/Dockerfile).

If your application requires additional dependencies, you can [install additional Python packages](../dependency-management/python-packages.md) or [install additional system packages](../dependency-management/system-packages.md).
