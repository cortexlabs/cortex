# Configuration

```yaml
- name: <string>
  kind: BatchAPI
  predictor: # detailed configuration below
  compute: # detailed configuration below
  networking: # detailed configuration below
```

## Predictor

### Python Predictor

<!-- CORTEX_VERSION_BRANCH_STABLE x2 -->
```yaml
predictor:
  type: python
  path: <string>  # path to a python file with a PythonPredictor class definition, relative to the Cortex root (required)
  config: <string: value>  # arbitrary dictionary passed to the constructor of the Predictor (can be overridden by config passed in job submission) (optional)
  python_path: <string>  # path to the root of your Python folder that will be appended to PYTHONPATH (default: folder containing cortex.yaml)
  image: <string> # docker image to use for the Predictor (default: quay.io/cortexlabs/python-predictor-cpu:master or quay.io/cortexlabs/python-predictor-gpu:master-cuda10.2-cudnn8 based on compute)
  env: <string: string>  # dictionary of environment variables
  log_level: <string>  # log level that can be "debug", "info", "warning" or "error" (default: "info")
  shm_size: <string>  # size of shared memory (/dev/shm) for sharing data between multiple processes, e.g. 64Mi or 1Gi (default: Null)
  dependencies: # (optional)
    pip: <string>  # relative path to requirements.txt (default: requirements.txt)
    conda: <string>  # relative path to conda-packages.txt (default: conda-packages.txt)
    shell: <string>  # relative path to a shell script for system package installation (default: dependencies.sh)
```

### TensorFlow Predictor

<!-- CORTEX_VERSION_BRANCH_STABLE x3 -->
```yaml
predictor:
  type: tensorflow
  path: <string>  # path to a python file with a TensorFlowPredictor class definition, relative to the Cortex root (required)
  models:  # use this to serve a single model or multiple ones
    path: <string>  # S3 path to an exported model (e.g. s3://my-bucket/exported_model) (either this or 'paths' field must be provided)
    paths:  # (either this or 'path' must be provided)
      - name: <string> # unique name for the model (e.g. text-generator) (required)
        path: <string>  # S3 path to an exported model (e.g. s3://my-bucket/exported_model) (required)
        signature_key: <string>  # name of the signature def to use for prediction (required if your model has more than one signature def)
      ...
    signature_key: <string>  # name of the signature def to use for prediction (required if your model has more than one signature def)
  server_side_batching:  # (optional)
    max_batch_size: <int>  # the maximum number of requests to aggregate before running inference
    batch_interval: <duration>  # the maximum amount of time to spend waiting for additional requests before running inference on the batch of requests
  config: <string: value>  # arbitrary dictionary passed to the constructor of the Predictor (can be overridden by config passed in job submission) (optional)
  python_path: <string>  # path to the root of your Python folder that will be appended to PYTHONPATH (default: folder containing cortex.yaml)
  image: <string> # docker image to use for the Predictor (default: quay.io/cortexlabs/tensorflow-predictor:master)
  tensorflow_serving_image: <string> # docker image to use for the TensorFlow Serving container (default: quay.io/cortexlabs/tensorflow-serving-cpu:master or quay.io/cortexlabs/tensorflow-serving-gpu:master based on compute)
  env: <string: string>  # dictionary of environment variables
  log_level: <string>  # log level that can be "debug", "info", "warning" or "error" (default: "info")
  shm_size: <string>  # size of shared memory (/dev/shm) for sharing data between multiple processes, e.g. 64Mi or 1Gi (default: Null)
  dependencies: # (optional)
    pip: <string>  # relative path to requirements.txt (default: requirements.txt)
    conda: <string>  # relative path to conda-packages.txt (default: conda-packages.txt)
    shell: <string>  # relative path to a shell script for system package installation (default: dependencies.sh)
```

### ONNX Predictor

<!-- CORTEX_VERSION_BRANCH_STABLE x2 -->
```yaml
predictor:
  type: onnx
  path: <string>  # path to a python file with an ONNXPredictor class definition, relative to the Cortex root (required)
  models:  # use this to serve a single model or multiple ones
    path: <string>  # S3 path to an exported model (e.g. s3://my-bucket/exported_model) (either this or 'paths' must be provided)
    paths:  # (either this or 'path' must be provided)
      - name: <string> # unique name for the model (e.g. text-generator) (required)
        path: <string>  # S3 path to an exported model (e.g. s3://my-bucket/exported_model.onnx) (required)
      ...
  config: <string: value>  # arbitrary dictionary passed to the constructor of the Predictor (can be overridden by config passed in job submission) (optional)
  python_path: <string>  # path to the root of your Python folder that will be appended to PYTHONPATH (default: folder containing cortex.yaml)
  image: <string> # docker image to use for the Predictor (default: quay.io/cortexlabs/onnx-predictor-cpu:master or quay.io/cortexlabs/onnx-predictor-gpu:master based on compute)
  env: <string: string>  # dictionary of environment variables
  log_level: <string>  # log level that can be "debug", "info", "warning" or "error" (default: "info")
  shm_size: <string>  # size of shared memory (/dev/shm) for sharing data between multiple processes, e.g. 64Mi or 1Gi (default: Null)
  dependencies: # (optional)
    pip: <string>  # relative path to requirements.txt (default: requirements.txt)
    conda: <string>  # relative path to conda-packages.txt (default: conda-packages.txt)
    shell: <string>  # relative path to a shell script for system package installation (default: dependencies.sh)
```

## Compute

```yaml
compute:
  cpu: <string | int | float>  # CPU request per worker. One unit of CPU corresponds to one virtual CPU; fractional requests are allowed, and can be specified as a floating point number or via the "m" suffix (default: 200m)
  gpu: <int>  # GPU request per worker. One unit of GPU corresponds to one virtual GPU (default: 0)
  mem: <string>  # memory request per worker. One unit of memory is one byte and can be expressed as an integer or by using one of these suffixes: K, M, G, T (or their power-of two counterparts: Ki, Mi, Gi, Ti) (default: Null)
```

## Networking

```yaml
networking:
  endpoint: <string>  # the endpoint for the API (default: <api_name>)
```
