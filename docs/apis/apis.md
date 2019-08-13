# APIs

Serve models at scale.

## Config

```yaml
- kind: api
  name: <string>  # API name (required)
  model: <string>  # path to an exported model (e.g. s3://my-bucket/model.zip)
  model_format: <string>  # model format, must be "tensorflow" or "onnx" (default: "onnx" if model path ends with .onnx, "tensorflow" if model path ends with .zip)
  request_handler: <string>  # path to the request handler implementation file, relative to the cortex root
  tracker:
    key: <string> # the key to the prediction in the response payload
    model_type: <string> # model type, must be "classification" or "regression"
  compute:
    min_replicas: <int>  # minimum number of replicas (default: 1)
    max_replicas: <int>  # maximum number of replicas (default: 100)
    init_replicas: <int>  # initial number of replicas (default: <min_replicas>)
    target_cpu_utilization: <int>  # CPU utilization threshold (as a percentage) to trigger scaling (default: 80)
    cpu: <string>  # CPU request per replica (default: 400m)
    gpu: <string>  # gpu request per replica (default: 0)
    mem: <string>  # memory request per replica (default: Null)
```

See [packaging models](packaging-models.md) for how to create the zipped model.

## Example

```yaml
- kind: api
  name: my-api
  model: s3://my-bucket/my-model.zip
  request_handler: inference.py
  compute:
    min_replicas: 5
    max_replicas: 20
    cpu: "1"
```

## Custom Request Handlers

Request handlers are used to decouple the interface of an API endpoint from its model. A `pre_inference` request handler can be used to modify request payloads before they are sent to the model. A `post_inference` request handler can be used to modify model predictions in the server before they are sent to the client.

See [request handlers](request-handlers.md) for a detailed guide.
