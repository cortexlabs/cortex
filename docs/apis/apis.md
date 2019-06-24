# APIs

Serve models at scale and use them to build smarter applications.

## Config

```yaml
- kind: api
  name: <string>  # API name (required)
  model: <string>  # reference to a model (e.g. s3://my-bucket/my-model.zip)
  preprocessor: <string>  # path to the implementation file, relative to the cortex root (default: <name>.py)
  postprocessor: <string>  # path to the implementation file, relative to the cortex root (default: <name>.py)
  compute:
    replicas: <int>  # number of replicas to launch (default: 1)
    cpu: <string>  # CPU request (default: Null)
    mem: <string>  # memory request (default: Null)
    gpu: <string>  # gpu request (default: Null)
```

## Example

```yaml
- kind: api
  name: my-api
  model: s3://my-bucket/my-model.zip
  preprocessor: transform_payload.py
  postprocessor: process_prediction.py
  compute:
    replicas: 3
    gpu: 2
```

## Integration

APIs can be integrated into other applications or services via their JSON endpoints. The endpoint for any API follows the following format: {apis_endpoint}/{deployment_name}/{api_name}.

The fields in the request payload for a particular API should match the raw columns that were used to train the model that it is serving. Cortex automatically applies the same transformers that were used at training time when responding to prediction requests.

## Horizontal Scalability

APIs can be configured using `replicas` in the `compute` field. Replicas can be used to change the amount of computing resources allocated to service prediction requests for a particular API. APIs that have low request volumes should have a small number of replicas while APIs that handle large request volumes should have more replicas.

## Rolling Updates

When the model that an API is serving gets updated, Cortex will update the API with the new model without any downtime.
