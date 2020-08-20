# API configuration

_WARNING: you are on the master branch, please refer to the docs on the branch that matches your `cortex version`_

Once your model is [exported](../../guides/exporting.md) and you've implemented a [Predictor](predictors.md), you can configure your API via a YAML file (typically named `cortex.yaml`).

Reference the section below which corresponds to your Predictor type: [Python](#python-predictor), [TensorFlow](#tensorflow-predictor), or [ONNX](#onnx-predictor).

## Python Predictor

```yaml
- name: <string>  # API name (required)
  kind: RealtimeAPI
  predictor:
    type: python
    path: <string>  # path to a python file with a PythonPredictor class definition, relative to the Cortex root (required)
    processes_per_replica: <int>  # the number of parallel serving processes to run on each replica (default: 1)
    threads_per_process: <int>  # the number of threads per process (default: 1)
    config: <string: value>  # arbitrary dictionary passed to the constructor of the Predictor (optional)
    python_path: <string>  # path to the root of your Python folder that will be appended to PYTHONPATH (default: folder containing cortex.yaml)
    image: <string> # docker image to use for the Predictor (default: cortexlabs/python-predictor-cpu or cortexlabs/python-predictor-gpu based on compute)
    env: <string: string>  # dictionary of environment variables
  networking:
    endpoint: <string>  # the endpoint for the API (aws only) (default: <api_name>)
    local_port: <int>  # specify the port for API (local only) (default: 8888)
    api_gateway: public | none  # whether to create a public API Gateway endpoint for this API (if not, the load balancer will be accessed directly) (default: public)
  compute:
    cpu: <string | int | float>  # CPU request per replica, e.g. 200m or 1 (200m is equivalent to 0.2) (default: 200m)
    gpu: <int>  # GPU request per replica (default: 0)
    inf: <int> # Inferentia ASIC request per replica (default: 0)
    mem: <string>  # memory request per replica, e.g. 200Mi or 1Gi (default: Null)
  monitoring:  # (aws only)
    model_type: <string>  # must be "classification" or "regression", so responses can be interpreted correctly (i.e. categorical vs continuous) (required)
    key: <string>  # the JSON key in the response payload of the value to monitor (required if the response payload is a JSON object)
  autoscaling:  # (aws only)
    min_replicas: <int>  # minimum number of replicas (default: 1)
    max_replicas: <int>  # maximum number of replicas (default: 100)
    init_replicas: <int>  # initial number of replicas (default: <min_replicas>)
    target_replica_concurrency: <float>  # the desired number of in-flight requests per replica, which the autoscaler tries to maintain (default: processes_per_replica * threads_per_process)
    max_replica_concurrency: <int>  # the maximum number of in-flight requests per replica before requests are rejected with error code 503 (default: 1024)
    window: <duration>  # the time over which to average the API's concurrency (default: 60s)
    downscale_stabilization_period: <duration>  # the API will not scale below the highest recommendation made during this period (default: 5m)
    upscale_stabilization_period: <duration>  # the API will not scale above the lowest recommendation made during this period (default: 1m)
    max_downscale_factor: <float>  # the maximum factor by which to scale down the API on a single scaling event (default: 0.75)
    max_upscale_factor: <float>  # the maximum factor by which to scale up the API on a single scaling event (default: 1.5)
    downscale_tolerance: <float>  # any recommendation falling within this factor below the current number of replicas will not trigger a scale down event (default: 0.05)
    upscale_tolerance: <float>  # any recommendation falling within this factor above the current number of replicas will not trigger a scale up event (default: 0.05)
  update_strategy:  # (aws only)
    max_surge: <string | int>  # maximum number of replicas that can be scheduled above the desired number of replicas during an update; can be an absolute number, e.g. 5, or a percentage of desired replicas, e.g. 10% (default: 25%) (set to 0 to disable rolling updates)
    max_unavailable: <string | int>  # maximum number of replicas that can be unavailable during an update; can be an absolute number, e.g. 5, or a percentage of desired replicas, e.g. 10% (default: 25%)
```

See additional documentation for [parallelism](parallelism.md), [autoscaling](autoscaling.md), [compute](../compute.md), [networking](../networking.md), [prediction monitoring](prediction-monitoring.md), and [overriding API images](../system-packages.md).

## TensorFlow Predictor

```yaml
- name: <string>  # API name (required)
  kind: RealtimeAPI
  predictor:
    type: tensorflow
    path: <string>  # path to a python file with a TensorFlowPredictor class definition, relative to the Cortex root (required)
    model_path: <string>  # S3 path to an exported model (e.g. s3://my-bucket/exported_model) (either this or 'models' must be provided)
    signature_key: <string>  # name of the signature def to use for prediction (required if your model has more than one signature def)
    models:  # use this when multiple models per API are desired (either this or 'model_path' must be provided)
      - name: <string> # unique name for the model (e.g. text-generator) (required)
        model_path: <string>  # S3 path to an exported model (e.g. s3://my-bucket/exported_model) (required)
        signature_key: <string>  # name of the signature def to use for prediction (required if your model has more than one signature def)
      ...
    server_side_batching:  # (optional)
      max_batch_size: <int>  # the maximum number of requests to aggregate before running inference
      batch_interval: <duration>  # the maximum amount of time to spend waiting for additional requests before running inference on the batch of requests
    processes_per_replica: <int>  # the number of parallel serving processes to run on each replica (default: 1)
    threads_per_process: <int>  # the number of threads per process (default: 1)
    config: <string: value>  # arbitrary dictionary passed to the constructor of the Predictor (optional)
    python_path: <string>  # path to the root of your Python folder that will be appended to PYTHONPATH (default: folder containing cortex.yaml)
    image: <string> # docker image to use for the Predictor (default: cortexlabs/tensorflow-predictor)
    tensorflow_serving_image: <string> # docker image to use for the TensorFlow Serving container (default: cortexlabs/tensorflow-serving-gpu or cortexlabs/tensorflow-serving-cpu based on compute)
    env: <string: string>  # dictionary of environment variables
  networking:
    endpoint: <string>  # the endpoint for the API (aws only) (default: <api_name>)
    local_port: <int>  # specify the port for API (local only) (default: 8888)
    api_gateway: public | none  # whether to create a public API Gateway endpoint for this API (if not, the load balancer will be accessed directly) (default: public)
  compute:
    cpu: <string | int | float>  # CPU request per replica, e.g. 200m or 1 (200m is equivalent to 0.2) (default: 200m)
    gpu: <int>  # GPU request per replica (default: 0)
    inf: <int> # Inferentia ASIC request per replica (default: 0)
    mem: <string>  # memory request per replica, e.g. 200Mi or 1Gi (default: Null)
  monitoring:  # (aws only)
    model_type: <string>  # must be "classification" or "regression", so responses can be interpreted correctly (i.e. categorical vs continuous) (required)
    key: <string>  # the JSON key in the response payload of the value to monitor (required if the response payload is a JSON object)
  autoscaling:  # (aws only)
    min_replicas: <int>  # minimum number of replicas (default: 1)
    max_replicas: <int>  # maximum number of replicas (default: 100)
    init_replicas: <int>  # initial number of replicas (default: <min_replicas>)
    target_replica_concurrency: <float>  # the desired number of in-flight requests per replica, which the autoscaler tries to maintain (default: processes_per_replica * threads_per_process)
    max_replica_concurrency: <int>  # the maximum number of in-flight requests per replica before requests are rejected with error code 503 (default: 1024)
    window: <duration>  # the time over which to average the API's concurrency (default: 60s)
    downscale_stabilization_period: <duration>  # the API will not scale below the highest recommendation made during this period (default: 5m)
    upscale_stabilization_period: <duration>  # the API will not scale above the lowest recommendation made during this period (default: 1m)
    max_downscale_factor: <float>  # the maximum factor by which to scale down the API on a single scaling event (default: 0.75)
    max_upscale_factor: <float>  # the maximum factor by which to scale up the API on a single scaling event (default: 1.5)
    downscale_tolerance: <float>  # any recommendation falling within this factor below the current number of replicas will not trigger a scale down event (default: 0.05)
    upscale_tolerance: <float>  # any recommendation falling within this factor above the current number of replicas will not trigger a scale up event (default: 0.05)
  update_strategy:  # (aws only)
    max_surge: <string | int>  # maximum number of replicas that can be scheduled above the desired number of replicas during an update; can be an absolute number, e.g. 5, or a percentage of desired replicas, e.g. 10% (default: 25%) (set to 0 to disable rolling updates)
    max_unavailable: <string | int>  # maximum number of replicas that can be unavailable during an update; can be an absolute number, e.g. 5, or a percentage of desired replicas, e.g. 10% (default: 25%)
```

See additional documentation for [parallelism](parallelism.md), [autoscaling](autoscaling.md), [compute](../compute.md), [networking](../networking.md), [prediction monitoring](prediction-monitoring.md), and [overriding API images](../system-packages.md).

## ONNX Predictor

```yaml
- name: <string>  # API name (required)
  kind: RealtimeAPI
  predictor:
    type: onnx
    path: <string>  # path to a python file with an ONNXPredictor class definition, relative to the Cortex root (required)
    model_path: <string>  # S3 path to an exported model (e.g. s3://my-bucket/exported_model.onnx) (either this or 'models' must be provided)
    models:  # use this when multiple models per API are desired (either this or 'model_path' must be provided)
      - name: <string> # unique name for the model (e.g. text-generator) (required)
        model_path: <string>  # S3 path to an exported model (e.g. s3://my-bucket/exported_model.onnx) (required)
        signature_key: <string>  # name of the signature def to use for prediction (required if your model has more than one signature def)
      ...
    processes_per_replica: <int>  # the number of parallel serving processes to run on each replica (default: 1)
    threads_per_process: <int>  # the number of threads per process (default: 1)
    config: <string: value>  # arbitrary dictionary passed to the constructor of the Predictor (optional)
    python_path: <string>  # path to the root of your Python folder that will be appended to PYTHONPATH (default: folder containing cortex.yaml)
    image: <string> # docker image to use for the Predictor (default: cortexlabs/onnx-predictor-gpu or cortexlabs/onnx-predictor-cpu based on compute)
    env: <string: string>  # dictionary of environment variables
  networking:
    endpoint: <string>  # the endpoint for the API (aws only) (default: <api_name>)
    local_port: <int>  # specify the port for API (local only) (default: 8888)
    api_gateway: public | none  # whether to create a public API Gateway endpoint for this API (if not, the load balancer will be accessed directly) (default: public)
  compute:
    cpu: <string | int | float>  # CPU request per replica, e.g. 200m or 1 (200m is equivalent to 0.2) (default: 200m)
    gpu: <int>  # GPU request per replica (default: 0)
    mem: <string>  # memory request per replica, e.g. 200Mi or 1Gi (default: Null)
  monitoring:  # (aws only)
    model_type: <string>  # must be "classification" or "regression", so responses can be interpreted correctly (i.e. categorical vs continuous) (required)
    key: <string>  # the JSON key in the response payload of the value to monitor (required if the response payload is a JSON object)
  autoscaling:  # (aws only)
    min_replicas: <int>  # minimum number of replicas (default: 1)
    max_replicas: <int>  # maximum number of replicas (default: 100)
    init_replicas: <int>  # initial number of replicas (default: <min_replicas>)
    target_replica_concurrency: <float>  # the desired number of in-flight requests per replica, which the autoscaler tries to maintain (default: processes_per_replica * threads_per_process)
    max_replica_concurrency: <int>  # the maximum number of in-flight requests per replica before requests are rejected with error code 503 (default: 1024)
    window: <duration>  # the time over which to average the API's concurrency (default: 60s)
    downscale_stabilization_period: <duration>  # the API will not scale below the highest recommendation made during this period (default: 5m)
    upscale_stabilization_period: <duration>  # the API will not scale above the lowest recommendation made during this period (default: 1m)
    max_downscale_factor: <float>  # the maximum factor by which to scale down the API on a single scaling event (default: 0.75)
    max_upscale_factor: <float>  # the maximum factor by which to scale up the API on a single scaling event (default: 1.5)
    downscale_tolerance: <float>  # any recommendation falling within this factor below the current number of replicas will not trigger a scale down event (default: 0.05)
    upscale_tolerance: <float>  # any recommendation falling within this factor above the current number of replicas will not trigger a scale up event (default: 0.05)
  update_strategy:  # (aws only)
    max_surge: <string | int>  # maximum number of replicas that can be scheduled above the desired number of replicas during an update; can be an absolute number, e.g. 5, or a percentage of desired replicas, e.g. 10% (default: 25%) (set to 0 to disable rolling updates)
    max_unavailable: <string | int>  # maximum number of replicas that can be unavailable during an update; can be an absolute number, e.g. 5, or a percentage of desired replicas, e.g. 10% (default: 25%)
```

See additional documentation for [parallelism](parallelism.md), [autoscaling](autoscaling.md), [compute](../compute.md), [networking](../networking.md), [prediction monitoring](prediction-monitoring.md), and [overriding API images](../system-packages.md).
