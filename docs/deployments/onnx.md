# APIs

Deploy ONNX models as web services.

## Config

```yaml
- kind: api
  name: <string>  # API name (required)
  endpoint: <string>  # the endpoint for the API (default: /<deployment_name>/<api_name>)
  python_path: <string>  # path to the root of your python folder that will be appended to PYTHONPATH (default: folder containing cortex.yaml)
  onnx:
    model: <string>  # path to an exported model (e.g. s3://my-bucket/exported_model.onnx) (required)
    request_handler: <string>  # path to the request handler implementation file, relative to the cortex root (optional)
  tracker:
    key: <string>  # key to track (required if the response payload is a JSON object)
    model_type: <string>  # model type, must be "classification" or "regression" (required)
  compute:
    min_replicas: <int>  # minimum number of replicas (default: 1)
    max_replicas: <int>  # maximum number of replicas (default: 100)
    init_replicas: <int>  # initial number of replicas (default: <min_replicas>)
    target_cpu_utilization: <int>  # CPU utilization threshold (as a percentage) to trigger scaling (default: 80)
    cpu: <string | int | float>  # CPU request per replica (default: 200m)
    gpu: <int>  # GPU request per replica (default: 0)
    mem: <string>  # memory request per replica (default: Null)
  metadata: <string: value>  # dictionary that can be used to configure custom values (optional)
```

See [packaging ONNX models](../packaging/onnx.md) for information about exporting ONNX models.

## Example

```yaml
- kind: api
  name: my-api
  onnx:
    model: s3://my-bucket/my-model.onnx
    request_handler: handler.py
  compute:
    gpu: 1
```

## Debugging

You can log information about each request by adding a `?debug=true` parameter to your requests. This will print:

1. The raw sample
2. The value after running the `pre_inference` function (if applicable)
3. The value after running inference
4. The value after running the `post_inference` function (if applicable)
