# API configuration

_WARNING: you are on the master branch, please refer to the docs on the branch that matches your `cortex version`_

Once your model is [exported](exporting.md) and you've implemented a [Predictor](predictors.md), you can configure your API via a yaml file (typically named `cortex.yaml`).

Reference the section below which corresponds to your Predictor type: [Python](#python-predictor), [TensorFlow](#tensorflow-predictor), or [ONNX](#onnx-predictor).

## Python Predictor

```yaml
- name: <string>  # API name (required)
  endpoint: <string>  # the endpoint for the API (default: <api_name>)
  predictor:
    type: python
    path: <string>  # path to a python file with a PythonPredictor class definition, relative to the Cortex root (required)
    config: <string: value>  # arbitrary dictionary passed to the constructor of the Predictor (optional)
    python_path: <string>  # path to the root of your Python folder that will be appended to PYTHONPATH (default: folder containing cortex.yaml)
    image: <string> # docker image to use for the Predictor (default: cortexlabs/python-serve[-gpu])
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
    upscale_stabilization_period: <duration>  # the API will not scale above the lowest recommendation made during this period (default: 1m)
    max_downscale_factor: <float>  # the maximum factor by which to scale down the API on a single scaling event (default: 0.75)
    max_upscale_factor: <float>  # the maximum factor by which to scale up the API on a single scaling event (default: 1.5)
    downscale_tolerance: <float>  # any recommendation falling within this factor below the current number of replicas will not trigger a scale down event (default: 0.05)
    upscale_tolerance: <float>  # any recommendation falling within this factor above the current number of replicas will not trigger a scale up event (default: 0.05)
  update_strategy:
    max_surge: <string | int>  # maximum number of replicas that can be scheduled above the desired number of replicas during an update; can be an absolute number, e.g. 5, or a percentage of desired replicas, e.g. 10% (default: 25%)
    max_unavailable: <string | int>  # maximum number of replicas that can be unavailable during an update; can be an absolute number, e.g. 5, or a percentage of desired replicas, e.g. 10% (default: 25%)
```

See additional documentation for [autoscaling](autoscaling.md), [compute](compute.md), [prediction monitoring](prediction-monitoring.md), and [overriding API images](system-packages.md).

## TensorFlow Predictor

```yaml
- name: <string>  # API name (required)
  endpoint: <string>  # the endpoint for the API (default: <api_name>)
  predictor:
    type: tensorflow
    path: <string>  # path to a python file with a TensorFlowPredictor class definition, relative to the Cortex root (required)
    model: <string>  # S3 path to an exported model (e.g. s3://my-bucket/exported_model) (required)
    signature_key: <string>  # name of the signature def to use for prediction (required if your model has more than one signature def)
    config: <string: value>  # arbitrary dictionary passed to the constructor of the Predictor (optional)
    python_path: <string>  # path to the root of your Python folder that will be appended to PYTHONPATH (default: folder containing cortex.yaml)
    image: <string> # docker image to use for the Predictor (default: cortexlabs/tensorflow-predictor)
    tensorflow_serving_image: <string> # docker image to use for the TensorFlow Serving container (default: cortexlabs/tensorflow-serving[-cpu|-gpu], which is based on tensorflow/serving)
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
    upscale_stabilization_period: <duration>  # the API will not scale above the lowest recommendation made during this period (default: 1m)
    max_downscale_factor: <float>  # the maximum factor by which to scale down the API on a single scaling event (default: 0.75)
    max_upscale_factor: <float>  # the maximum factor by which to scale up the API on a single scaling event (default: 1.5)
    downscale_tolerance: <float>  # any recommendation falling within this factor below the current number of replicas will not trigger a scale down event (default: 0.05)
    upscale_tolerance: <float>  # any recommendation falling within this factor above the current number of replicas will not trigger a scale up event (default: 0.05)
  update_strategy:
    max_surge: <string | int>  # maximum number of replicas that can be scheduled above the desired number of replicas during an update; can be an absolute number, e.g. 5, or a percentage of desired replicas, e.g. 10% (default: 25%)
    max_unavailable: <string | int>  # maximum number of replicas that can be unavailable during an update; can be an absolute number, e.g. 5, or a percentage of desired replicas, e.g. 10% (default: 25%)
```

See additional documentation for [autoscaling](autoscaling.md), [compute](compute.md), [prediction monitoring](prediction-monitoring.md), and [overriding API images](system-packages.md).

## ONNX Predictor

```yaml
- name: <string>  # API name (required)
  endpoint: <string>  # the endpoint for the API (default: <api_name>)
  predictor:
    type: onnx
    path: <string>  # path to a python file with an ONNXPredictor class definition, relative to the Cortex root (required)
    model: <string>  # S3 path to an exported model (e.g. s3://my-bucket/exported_model.onnx) (required)
    config: <string: value>  # arbitrary dictionary passed to the constructor of the Predictor (optional)
    python_path: <string>  # path to the root of your Python folder that will be appended to PYTHONPATH (default: folder containing cortex.yaml)
    image: <string> # docker image to use for the Predictor (default: cortexlabs/onnx-serve[-gpu])
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
    upscale_stabilization_period: <duration>  # the API will not scale above the lowest recommendation made during this period (default: 1m)
    max_downscale_factor: <float>  # the maximum factor by which to scale down the API on a single scaling event (default: 0.75)
    max_upscale_factor: <float>  # the maximum factor by which to scale up the API on a single scaling event (default: 1.5)
    downscale_tolerance: <float>  # any recommendation falling within this factor below the current number of replicas will not trigger a scale down event (default: 0.05)
    upscale_tolerance: <float>  # any recommendation falling within this factor above the current number of replicas will not trigger a scale up event (default: 0.05)
  update_strategy:
    max_surge: <string | int>  # maximum number of replicas that can be scheduled above the desired number of replicas during an update; can be an absolute number, e.g. 5, or a percentage of desired replicas, e.g. 10% (default: 25%)
    max_unavailable: <string | int>  # maximum number of replicas that can be unavailable during an update; can be an absolute number, e.g. 5, or a percentage of desired replicas, e.g. 10% (default: 25%)
```

See additional documentation for [autoscaling](autoscaling.md), [compute](compute.md), [prediction monitoring](prediction-monitoring.md), and [overriding API images](system-packages.md).
