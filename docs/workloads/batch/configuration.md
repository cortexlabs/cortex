# Batch API configuration

## Python Predictor

<!-- CORTEX_VERSION_BRANCH_STABLE x2 -->
```yaml
- name: <string>  # API name (required)
  kind: BatchAPI
  predictor:
    type: python
    path: <string>  # path to a python file with a PythonPredictor class definition, relative to the Cortex root (required)
    config: <string: value>  # arbitrary dictionary passed to the constructor of the Predictor (can be overridden by config passed in job submission) (optional)
    python_path: <string>  # path to the root of your Python folder that will be appended to PYTHONPATH (default: folder containing cortex.yaml)
    image: <string> # docker image to use for the Predictor (default: quay.io/cortexlabs/python-predictor-cpu:master or quay.io/cortexlabs/python-predictor-gpu:master based on compute)
    env: <string: string>  # dictionary of environment variables
    log_level: <string>  # log level that can be "debug", "info", "warning" or "error" (default: "info")
  networking:
    endpoint: <string>  # the endpoint for the API (default: <api_name>)
  compute:
    cpu: <string | int | float>  # CPU request per worker, e.g. 200m or 1 (200m is equivalent to 0.2) (default: 200m)
    gpu: <int>  # GPU request per worker (default: 0)
    inf: <int> # Inferentia ASIC request per worker (default: 0)
    mem: <string>  # memory request per worker, e.g. 200Mi or 1Gi (default: Null)
```

## TensorFlow Predictor

<!-- CORTEX_VERSION_BRANCH_STABLE x3 -->
```yaml
- name: <string>  # API name (required)
  kind: BatchAPI
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
    tensorflow_serving_image: <string> # docker image to use for the TensorFlow Serving container (default: quay.io/cortexlabs/tensorflow-serving-gpu:master or quay.io/cortexlabs/tensorflow-serving-cpu:master based on compute)
    env: <string: string>  # dictionary of environment variables
    log_level: <string>  # log level that can be "debug", "info", "warning" or "error" (default: "info")
  networking:
    endpoint: <string>  # the endpoint for the API (default: <api_name>)
  compute:
    cpu: <string | int | float>  # CPU request per worker, e.g. 200m or 1 (200m is equivalent to 0.2) (default: 200m)
    gpu: <int>  # GPU request per worker (default: 0)
    inf: <int> # Inferentia ASIC request per worker (default: 0)
    mem: <string>  # memory request per worker, e.g. 200Mi or 1Gi (default: Null)
```

## ONNX Predictor

<!-- CORTEX_VERSION_BRANCH_STABLE x2 -->
```yaml
- name: <string>  # API name (required)
  kind: BatchAPI
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
    image: <string> # docker image to use for the Predictor (default: quay.io/cortexlabs/onnx-predictor-gpu:master or quay.io/cortexlabs/onnx-predictor-cpu:master based on compute)
    env: <string: string>  # dictionary of environment variables
    log_level: <string>  # log level that can be "debug", "info", "warning" or "error" (default: "info")
  networking:
    endpoint: <string>  # the endpoint for the API (default: <api_name>)
  compute:
    cpu: <string | int | float>  # CPU request per worker, e.g. 200m or 1 (200m is equivalent to 0.2) (default: 200m)
    gpu: <int>  # GPU request per worker (default: 0)
    mem: <string>  # memory request per worker, e.g. 200Mi or 1Gi (default: Null)
```
