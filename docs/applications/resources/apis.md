# APIs

Serve models at scale and use them to build smarter applications.

## Config

```yaml
- kind: api  # (required)
  name: <string>  # API name (required)
  model_name: <string>  # reference to a Cortex model (required)
  compute:
    replicas: <int>  # number of replicas to launch (default: 1)
    cpu: <string>  # CPU request (default: Null)
    mem: <string>  # memory request (default: Null)
  tags:
    <string>: <scalar>  # arbitrary key/value pairs to attach to the resource (optional)
    ...
```

## Example

```yaml
- kind: api
  name: classifier
  model_name: dnn
  compute:
    replicas: 3
```

## Integration

APIs can be integrated into other applications or services via their JSON endpoints. The endpoint for any API follows the following format: {APIs_endpoint}/{app_name}/{API_name}.

The request payload for a particular API should match the raw features that were used to train the model that it is serving. Cortex automatically applies the same transformers that were used at training time when responding to prediction requests.

## Horizontal Scalability

APIs can be configured using `replicas` in the `compute` field. Replicas can be used to change the amount of computing resources allocated to service prediction requests for a particular API. APIs that have low request volumes should have a small number of replicas while APIs that handle large request volumes should have more replicas.

## Rolling Updates

When the model that an API is serving gets updated, Cortex will update the API with the new model without any downtime.
