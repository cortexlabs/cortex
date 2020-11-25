# API configuration

_WARNING: you are on the master branch, please refer to the docs on the branch that matches your `cortex version`_

Once your model is [exported](../../guides/exporting.md) and you've implemented a [Predictor](predictors.md), you can configure your API via a YAML file (typically named `cortex.yaml`).

Reference the section below which corresponds to your Predictor type: [Python](#python-predictor), [TensorFlow](#tensorflow-predictor), or [ONNX](#onnx-predictor).

## Python Predictor

<!-- CORTEX_VERSION_BRANCH_STABLE x2 -->
```yaml
- name: <string>  # API name (required)
  kind: RealtimeAPI
  predictor:
    type: python
    path: <string>  # path to a python file with a PythonPredictor class definition, relative to the Cortex root (required)
    model_path: <string>  # S3 path to an exported model directory (e.g. s3://my-bucket/exported_model/) (optional, cannot be provided along with 'models')
    models:  # use this to serve multiple models in a single API (optional, cannot be provided along with 'model_path')
      dir: <string>  # S3 path to a directory containing multiple models (e.g. s3://my-bucket/models/) (either this or 'paths' must be provided)
      paths:  # list of S3 paths to exported model directories (either this or 'dir' must be provided)
        - name: <string>  # unique name for the model (e.g. text-generator) (required)
          model_path: <string>  # S3 path to an exported model directory (e.g. s3://my-bucket/exported_model/) (required)
        ...
      cache_size: <int>  # the number models to keep in memory (optional; all models are kept in memory by default)
      disk_cache_size: <int>  # the number of models to keep on disk (optional; all models are kept on disk by default)
    processes_per_replica: <int>  # the number of parallel serving processes to run on each replica (default: 1)
    threads_per_process: <int>  # the number of threads per process (default: 1)
    config: <string: value>  # arbitrary dictionary passed to the constructor of the Predictor (optional)
    python_path: <string>  # path to the root of your Python folder that will be appended to PYTHONPATH (default: folder containing cortex.yaml)
    image: <string>  # docker image to use for the Predictor (default: quay.io/cortexlabs/python-predictor-cpu:master or quay.io/cortexlabs/python-predictor-gpu:master based on compute)
    env: <string: string>  # dictionary of environment variables
  networking:
    endpoint: <string>  # the endpoint for the API (aws only) (default: <api_name>)
    local_port: <int>  # specify the port for API (local only) (default: 8888)
    api_gateway: public | none  # whether to create a public API Gateway endpoint for this API (if not, the API will still be accessible via the load balancer) (default: public, unless disabled cluster-wide)
  compute:
    cpu: <string | int | float>  # CPU request per replica, e.g. 200m or 1 (200m is equivalent to 0.2) (default: 200m)
    gpu: <int>  # GPU request per replica (default: 0)
    inf: <int>  # Inferentia ASIC request per replica (default: 0)
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

See additional documentation for [models](models.md), [parallelism](parallelism.md), [autoscaling](autoscaling.md), [compute](../compute.md), [networking](../../aws/networking.md), [prediction monitoring](prediction-monitoring.md), and [overriding API images](../system-packages.md).

## TensorFlow Predictor

<!-- CORTEX_VERSION_BRANCH_STABLE x3 -->
```yaml
- name: <string>  # API name (required)
  kind: RealtimeAPI
  predictor:
    type: tensorflow
    path: <string>  # path to a python file with a TensorFlowPredictor class definition, relative to the Cortex root (required)
    model_path: <string>  # S3 path to an exported model directory (e.g. s3://my-bucket/exported_model/) (either this or 'models' must be provided)
    signature_key: <string>  # name of the signature def to use for prediction (required if your model has more than one signature def)
    models:  # use this to serve multiple models in a single API (either this or 'model_path' must be provided)
      dir: <string>  # S3 path to a directory containing multiple models (e.g. s3://my-bucket/models/) (either this or 'paths' must be provided)
      paths:  # list of S3 paths to exported model directories (either this or 'dir' must be provided)
        - name: <string>  # unique name for the model (e.g. text-generator) (required)
          model_path: <string>  # S3 path to an exported model directory (e.g. s3://my-bucket/exported_model/) (required)
          signature_key: <string>  # name of the signature def to use for prediction (required if your model has more than one signature def)
        ...
      signature_key:  # name of the signature def to use for prediction for 'dir'-specified models or for models specified using 'paths' that haven't had a signature key set
      cache_size: <int>  # the number models to keep in memory (optional; all models are kept in memory by default)
      disk_cache_size: <int>  # the number of models to keep on disk (optional; all models are kept on disk by default)
    server_side_batching:  # (optional)
      max_batch_size: <int>  # the maximum number of requests to aggregate before running inference
      batch_interval: <duration>  # the maximum amount of time to spend waiting for additional requests before running inference on the batch of requests
    processes_per_replica: <int>  # the number of parallel serving processes to run on each replica (default: 1)
    threads_per_process: <int>  # the number of threads per process (default: 1)
    config: <string: value>  # arbitrary dictionary passed to the constructor of the Predictor (optional)
    python_path: <string>  # path to the root of your Python folder that will be appended to PYTHONPATH (default: folder containing cortex.yaml)
    image: <string>  # docker image to use for the Predictor (default: quay.io/cortexlabs/tensorflow-predictor:master)
    tensorflow_serving_image: <string>  # docker image to use for the TensorFlow Serving container (default: quay.io/cortexlabs/tensorflow-serving-gpu:master or quay.io/cortexlabs/tensorflow-serving-cpu:master based on compute)
    env: <string: string>  # dictionary of environment variables
  networking:
    endpoint: <string>  # the endpoint for the API (aws only) (default: <api_name>)
    local_port: <int>  # specify the port for API (local only) (default: 8888)
    api_gateway: public | none  # whether to create a public API Gateway endpoint for this API (if not, the API will still be accessible via the load balancer) (default: public, unless disabled cluster-wide)
  compute:
    cpu: <string | int | float>  # CPU request per replica, e.g. 200m or 1 (200m is equivalent to 0.2) (default: 200m)
    gpu: <int>  # GPU request per replica (default: 0)
    inf: <int>  # Inferentia ASIC request per replica (default: 0)
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

See additional documentation for [models](models.md), [parallelism](parallelism.md), [autoscaling](autoscaling.md), [compute](../compute.md), [networking](../../aws/networking.md), [prediction monitoring](prediction-monitoring.md), and [overriding API images](../system-packages.md).

## ONNX Predictor

<!-- CORTEX_VERSION_BRANCH_STABLE x2 -->
```yaml
- name: <string>  # API name (required)
  kind: RealtimeAPI
  predictor:
    type: onnx
    path: <string>  # path to a python file with an ONNXPredictor class definition, relative to the Cortex root (required)
    model_path: <string>  # S3 path to an exported model directory (e.g. s3://my-bucket/exported_model/) (either this or 'models' must be provided)
    models:  # use this to serve multiple models in a single API (either this or 'model_path' must be provided)
      dir: <string>  # S3 path to a directory containing multiple models (e.g. s3://my-bucket/models/) (either this or 'paths' must be provided)
      paths:  # list of S3 paths to exported model directories (either this or 'dir' must be provided)
        - name: <string>  # unique name for the model (e.g. text-generator) (required)
          model_path: <string>  # S3 path to an exported model directory (e.g. s3://my-bucket/exported_model/) (required)
          signature_key: <string>  # name of the signature def to use for prediction (required if your model has more than one signature def)
        ...
      cache_size: <int>  # the number models to keep in memory (optional; all models are kept in memory by default)
      disk_cache_size: <int>  # the number of models to keep on disk (optional; all models are kept on disk by default)
    processes_per_replica: <int>  # the number of parallel serving processes to run on each replica (default: 1)
    threads_per_process: <int>  # the number of threads per process (default: 1)
    config: <string: value>  # arbitrary dictionary passed to the constructor of the Predictor (optional)
    python_path: <string>  # path to the root of your Python folder that will be appended to PYTHONPATH (default: folder containing cortex.yaml)
    image: <string>  # docker image to use for the Predictor (default: quay.io/cortexlabs/onnx-predictor-gpu:master or quay.io/cortexlabs/onnx-predictor-cpu:master based on compute)
    env: <string: string>  # dictionary of environment variables
  networking:
    endpoint: <string>  # the endpoint for the API (aws only) (default: <api_name>)
    local_port: <int>  # specify the port for API (local only) (default: 8888)
    api_gateway: public | none  # whether to create a public API Gateway endpoint for this API (if not, the API will still be accessible via the load balancer) (default: public, unless disabled cluster-wide)
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

See additional documentation for [models](models.md), [parallelism](parallelism.md), [autoscaling](autoscaling.md), [compute](../compute.md), [networking](../../aws/networking.md), [prediction monitoring](prediction-monitoring.md), and [overriding API images](../system-packages.md).
