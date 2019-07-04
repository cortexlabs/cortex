# APIs

Serve models at scale and use them to build smarter applications.

## Config

```yaml
- kind: api
  name: <string>  # API name (required)
  model: <string>  # path to a zipped model dir (e.g. s3://my-bucket/model.zip)
  model_type: <string>  # framework of model, currently support tensorflow and onnx
  request_handler: <string>  # path to the implementation file, relative to the cortex root
  compute:
    min_replicas: <int>  # minimum number of replicas (default: 1)
    max_replicas: <int>  # maximum number of replicas (default: 100)
    init_replicas: <int>  # initial number of replicas (default: <min_replicas>)
    target_cpu_utilization: <int>  # CPU utilization threshold (as a percentage) to trigger scaling (default: 80)
    cpu: <string>  # CPU request per replica (default: 200m)
    gpu: <string>  # gpu request per replica (default: 0)
    mem: <string>  # memory request per replica (default: Null)
```

See [packaging models](packaging-models.md) for how to create the zipped model.

## Example

```yaml
- kind: api
  name: my-api
  model: s3://my-bucket/my-model.zip
  compute:
    min_replicas: 5
    max_replicas: 20
    cpu: "1"
```

## Custom Request Handlers

API endpoints can be customized by providing a request handler. Request handlers can be used to prepare request payloads before being passed to model for inference and modify model predictions before they are served.

```python
def preinference(sample, metadata):
    """Prepare a sample before it is passed into the model.

    Args:
        sample: A single sample in the request payload converted from JSON to Python object.

        metadata: Describes the expected shape and type of inputs to the model.
            If API model_type is tensorflow the object is a map<string, SignatureDef>
                https://github.com/tensorflow/tensorflow/blob/master/tensorflow/core/protobuf/meta_graph.proto
            If API model_type is onnx the object is a list of [onnxruntime.NodeArg]
                https://microsoft.github.io/onnxruntime/api_summary.html#onnxruntime.NodeArg

    Returns:
        If model only has one 1 input, return a python list or numpy array of expected type  and shape. If model has more than 1 input, return a dictionary mapping input names to python list or numpy array of expected type and shape.
    """
    pass

def postinference(prediction, metadata):
    """Modify prediction from model before adding it to response payload.

    Args:
        sample: A single sample in the request payload converted from JSON to Python object

        metadata: Describes the output shape and type of outputs from the model.
            If API model_type is tensorflow the object is a map<string, SignatureDef>
                https://github.com/tensorflow/tensorflow/blob/master/tensorflow/core/protobuf/meta_graph.proto
            If API model_type is onnx the object is a list of [onnxruntime.NodeArg]
                https://microsoft.github.io/onnxruntime/api_summary.html#onnxruntime.NodeArg

    Returns:
        Python object that can be marshalled to JSON.
    """
```

## Integration

APIs can be integrated into other applications or services via their JSON endpoints. The endpoint for any API follows the following format: {apis_endpoint}/{deployment_name}/{api_name}.

The fields in the request payload for a particular API should match the raw columns that were used to train the model that it is serving. Cortex automatically applies the same transformers that were used at training time when responding to prediction requests.

## Horizontal Scalability

APIs can be configured using `replicas` in the `compute` field. Replicas can be used to change the amount of computing resources allocated to service prediction requests for a particular API. APIs that have low request volumes should have a small number of replicas while APIs that handle large request volumes should have more replicas.

## Rolling Updates

When the model that an API is serving gets updated, Cortex will update the API with the new model without any downtime.
