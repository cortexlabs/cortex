# APIs

Serve models at scale.

## Config

```yaml
- kind: api
  name: <string>  # API name (required)
  model: <string>  # path to an exported model (e.g. s3://my-bucket/exported_model)
  model_format: <string>  # model format, must be "tensorflow" or "onnx" (default: "onnx" if model path ends with .onnx, "tensorflow" if model path ends with .zip or is a directory)
  request_handler: <string>  # path to the request handler implementation file, relative to the cortex root
  tf_signature_key: <string> # The name of the signature def to use for prediction (optional if your model has only one signature def)
  tracker:
    key: <string> # json key to track if the response payload is a dictionary
    model_type: <string> # model type, must be "classification" or "regression"
  compute:
    min_replicas: <int>  # minimum number of replicas (default: 1)
    max_replicas: <int>  # maximum number of replicas (default: 100)
    init_replicas: <int>  # initial number of replicas (default: <min_replicas>)
    target_cpu_utilization: <int>  # CPU utilization threshold (as a percentage) to trigger scaling (default: 80)
    cpu: <string | int | float>  # CPU request per replica (default: 200m)
    gpu: <int>  # gpu request per replica (default: 0)
    mem: <string>  # memory request per replica (default: Null)
```

See [packaging models](packaging-models.md) for how to export the model.

## Example

```yaml
- kind: api
  name: my-api
  model: s3://my-bucket/my-model.onnx
  request_handler: inference.py
  compute:
    min_replicas: 5
    max_replicas: 20
    gpu: 1
```

## Custom Request Handlers

Request handlers are used to decouple the interface of an API endpoint from its model. A `pre_inference` request handler can be used to modify request payloads before they are sent to the model. A `post_inference` request handler can be used to modify model predictions in the server before they are sent to the client.

See [request handlers](request-handlers.md) for a detailed guide.

## Debugging

You can log more information about each request by adding a `?debug=true` parameter to your requests. This will print:

1. The raw sample
2. The value after running the `pre_inference` function (if applicable)
3. The value after running inference
4. The value after running the `post_inference` function (if applicable)
