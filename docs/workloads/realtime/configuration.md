# Configuration

```yaml
- name: <string>
  kind: RealtimeAPI
  predictor: # detailed configuration below
  compute: # detailed configuration below
  autoscaling: # detailed configuration below
  update_strategy: # detailed configuration below
  networking: # detailed configuration below
```

## Predictor

### Python Predictor

<!-- CORTEX_VERSION_BRANCH_STABLE x3 -->
```yaml
predictor:
  type: python
  path: <string>  # path to a python file with a PythonPredictor class definition, relative to the Cortex root (required)
  dependencies: # (optional)
    pip: <string>  # relative path to requirements.txt (default: requirements.txt)
    conda: <string>  # relative path to conda-packages.txt (default: conda-packages.txt)
    shell: <string>  # relative path to a shell script for system package installation (default: dependencies.sh)
  multi_model_reloading:  # use this to serve one or more models with live reloading (optional)
    path: <string> # S3/GCS path to an exported model directory (e.g. s3://my-bucket/exported_model/) (either this, 'dir', or 'paths' must be provided if 'multi_model_reloading' is specified)
    paths:  # list of S3/GCS paths to exported model directories (either this, 'dir', or 'path' must be provided if 'multi_model_reloading' is specified)
      - name: <string>  # unique name for the model (e.g. text-generator) (required)
        path: <string>  # S3/GCS path to an exported model directory (e.g. s3://my-bucket/exported_model/) (required)
      ...
    dir: <string>  # S3/GCS path to a directory containing multiple models (e.g. s3://my-bucket/models/) (either this, 'path', or 'paths' must be provided if 'multi_model_reloading' is specified)
    cache_size: <int>  # the number models to keep in memory (optional; all models are kept in memory by default)
    disk_cache_size: <int>  # the number of models to keep on disk (optional; all models are kept on disk by default)
  server_side_batching:  # (optional)
    max_batch_size: <int>  # the maximum number of requests to aggregate before running inference
    batch_interval: <duration>  # the maximum amount of time to spend waiting for additional requests before running inference on the batch of requests
  processes_per_replica: <int>  # the number of parallel serving processes to run on each replica (default: 1)
  threads_per_process: <int>  # the number of threads per process (default: 1)
  config: <string: value>  # arbitrary dictionary passed to the constructor of the Predictor (optional)
  python_path: <string>  # path to the root of your Python folder that will be appended to PYTHONPATH (default: folder containing cortex.yaml)
  image: <string>  # docker image to use for the Predictor (default: quay.io/cortexlabs/python-predictor-cpu:master, quay.io/cortexlabs/python-predictor-gpu:master-cuda10.2-cudnn8, or quay.io/cortexlabs/python-predictor-inf:master based on compute)
  env: <string: string>  # dictionary of environment variables
  log_level: <string>  # log level that can be "debug", "info", "warning" or "error" (default: "info")
  shm_size: <string>  # size of shared memory (/dev/shm) for sharing data between multiple processes, e.g. 64Mi or 1Gi (default: Null)
```

### TensorFlow Predictor

<!-- CORTEX_VERSION_BRANCH_STABLE x4 -->
```yaml
predictor:
  type: tensorflow
  path: <string>  # path to a python file with a TensorFlowPredictor class definition, relative to the Cortex root (required)
  dependencies: # (optional)
    pip: <string>  # relative path to requirements.txt (default: requirements.txt)
    conda: <string>  # relative path to conda-packages.txt (default: conda-packages.txt)
    shell: <string>  # relative path to a shell script for system package installation (default: dependencies.sh)
  models:  # (required)
    path: <string> # S3/GCS path to an exported SavedModel directory (e.g. s3://my-bucket/exported_model/) (either this, 'dir', or 'paths' must be provided)
    paths:  # list of S3/GCS paths to exported SavedModel directories (either this, 'dir', or 'path' must be provided)
      - name: <string>  # unique name for the model (e.g. text-generator) (required)
        path: <string>  # S3/GCS path to an exported SavedModel directory (e.g. s3://my-bucket/exported_model/) (required)
        signature_key: <string>  # name of the signature def to use for prediction (required if your model has more than one signature def)
      ...
    dir: <string>  # S3/GCS path to a directory containing multiple SavedModel directories (e.g. s3://my-bucket/models/) (either this, 'path', or 'paths' must be provided)
    signature_key:  # name of the signature def to use for prediction (required if your model has more than one signature def)
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
  tensorflow_serving_image: <string>  # docker image to use for the TensorFlow Serving container (default: quay.io/cortexlabs/tensorflow-serving-cpu:master, quay.io/cortexlabs/tensorflow-serving-gpu:master, or quay.io/cortexlabs/tensorflow-serving-inf:master based on compute)
  env: <string: string>  # dictionary of environment variables
  log_level: <string>  # log level that can be "debug", "info", "warning" or "error" (default: "info")
  shm_size: <string>  # size of shared memory (/dev/shm) for sharing data between multiple processes, e.g. 64Mi or 1Gi (default: Null)
```

### ONNX Predictor

<!-- CORTEX_VERSION_BRANCH_STABLE x2 -->
```yaml
predictor:
  type: onnx
  path: <string>  # path to a python file with an ONNXPredictor class definition, relative to the Cortex root (required)
  dependencies: # (optional)
    pip: <string>  # relative path to requirements.txt (default: requirements.txt)
    conda: <string>  # relative path to conda-packages.txt (default: conda-packages.txt)
    shell: <string>  # relative path to a shell script for system package installation (default: dependencies.sh)
  models:  # (required)
    path: <string> # S3/GCS path to an exported model directory (e.g. s3://my-bucket/exported_model/) (either this, 'dir', or 'paths' must be provided)
    paths:  # list of S3/GCS paths to exported model directories (either this, 'dir', or 'path' must be provided)
      - name: <string>  # unique name for the model (e.g. text-generator) (required)
        path: <string>  # S3/GCS path to an exported model directory (e.g. s3://my-bucket/exported_model/) (required)
      ...
    dir: <string>  # S3/GCS path to a directory containing multiple model directories (e.g. s3://my-bucket/models/) (either this, 'path', or 'paths' must be provided)
    cache_size: <int>  # the number models to keep in memory (optional; all models are kept in memory by default)
    disk_cache_size: <int>  # the number of models to keep on disk (optional; all models are kept on disk by default)
  processes_per_replica: <int>  # the number of parallel serving processes to run on each replica (default: 1)
  threads_per_process: <int>  # the number of threads per process (default: 1)
  config: <string: value>  # arbitrary dictionary passed to the constructor of the Predictor (optional)
  python_path: <string>  # path to the root of your Python folder that will be appended to PYTHONPATH (default: folder containing cortex.yaml)
  image: <string>  # docker image to use for the Predictor (default: quay.io/cortexlabs/onnx-predictor-cpu:master or quay.io/cortexlabs/onnx-predictor-gpu:master based on compute)
  env: <string: string>  # dictionary of environment variables
  log_level: <string>  # log level that can be "debug", "info", "warning" or "error" (default: "info")
  shm_size: <string>  # size of shared memory (/dev/shm) for sharing data between multiple processes, e.g. 64Mi or 1Gi (default: Null)
```

## Compute

```yaml
compute:
  cpu: <string | int | float>  # CPU request per replica. One unit of CPU corresponds to one virtual CPU; fractional requests are allowed, and can be specified as a floating point number or via the "m" suffix (default: 200m)
  gpu: <int>  # GPU request per replica. One unit of GPU corresponds to one virtual GPU (default: 0)
  mem: <string>  # memory request per replica. One unit of memory is one byte and can be expressed as an integer or by using one of these suffixes: K, M, G, T (or their power-of two counterparts: Ki, Mi, Gi, Ti) (default: Null)
```

## Autoscaling

```yaml
autoscaling:
  min_replicas: <int>  # minimum number of replicas (default: 1)
  max_replicas: <int>  # maximum number of replicas (default: 100)
  init_replicas: <int>  # initial number of replicas (default: <min_replicas>)
  max_replica_concurrency: <int>  # the maximum number of in-flight requests per replica before requests are rejected with error code 503 (default: 1024)
  target_replica_concurrency: <float>  # the desired number of in-flight requests per replica, which the autoscaler tries to maintain (default: processes_per_replica * threads_per_process) (aws only)
  window: <duration>  # the time over which to average the API's concurrency (default: 60s) (aws only)
  downscale_stabilization_period: <duration>  # the API will not scale below the highest recommendation made during this period (default: 5m) (aws only)
  upscale_stabilization_period: <duration>  # the API will not scale above the lowest recommendation made during this period (default: 1m) (aws only)
  max_downscale_factor: <float>  # the maximum factor by which to scale down the API on a single scaling event (default: 0.75) (aws only)
  max_upscale_factor: <float>  # the maximum factor by which to scale up the API on a single scaling event (default: 1.5) (aws only)
  downscale_tolerance: <float>  # any recommendation falling within this factor below the current number of replicas will not trigger a scale down event (default: 0.05) (aws only)
  upscale_tolerance: <float>  # any recommendation falling within this factor above the current number of replicas will not trigger a scale up event (default: 0.05) (aws only)
```

## Update strategy

```yaml
update_strategy:
  max_surge: <string | int>  # maximum number of replicas that can be scheduled above the desired number of replicas during an update; can be an absolute number, e.g. 5, or a percentage of desired replicas, e.g. 10% (default: 25%) (set to 0 to disable rolling updates)
  max_unavailable: <string | int>  # maximum number of replicas that can be unavailable during an update; can be an absolute number, e.g. 5, or a percentage of desired replicas, e.g. 10% (default: 25%)
```

## Networking

```yaml
  networking:
    endpoint: <string>  # the endpoint for the API (default: <api_name>)
```
