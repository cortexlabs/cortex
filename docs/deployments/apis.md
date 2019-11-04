# APIs

You can deploy models from any Python modeling framework by implementing Cortex's Predictor interface. The interface consists of an `init()` function and a `predict()` function. The `init()` function is responsible for preparing the model for serving, downloading vocabulary files, etc. The `predict()` function is called on every request and is responsible for responding with a prediction. See [predictor](./predictor.md) for more details.

In addition to supporting Python models via the Predictor interface, Cortex can serve the following exported model formats:

- [TensorFlow](tensorflow.md)
- [ONNX](onnx.md)

## Configuration

```yaml
- kind: api
  name: <string>  # API name (required)
  endpoint: <string>  # the endpoint for the API (default: /<deployment_name>/<api_name>)
  predictor:
    path: <string>  # path to the predictor Python file, relative to the Cortex root (required)
    model: <string>  # S3 path to an exported model (e.g. s3://my-bucket/exported_model) (optional)
    python_path: <string>  # path to the root of your Python folder that will be appended to PYTHONPATH (default: folder containing cortex.yaml)
    metadata: <string: value>  # dictionary that can be used to configure custom values (optional)
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

### Example

```yaml
- kind: api
  name: my-api
  predictor:
    path: predictor.py
  compute:
    gpu: 1
```

## Debugging

You can log information about each request by adding a `?debug=true` parameter to your requests. This will print:

1. The raw sample
2. The value after running the `predict` function
