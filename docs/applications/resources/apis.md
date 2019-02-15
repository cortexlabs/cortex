# APIs

Serve models at scale and use them to build smarter applications.

## Config

```yaml
- kind: api  # (required)
  name: <string>  # API name (required)
  model_name: <string>  # name of a Cortex model (required)
  compute:
    replicas: <int>  # number of replicas to launch (default: 1)
    cpu: <string>  # CPU request (default: Null)
    mem: <string>  # memory request (default: Null)
    gpu: <string>  # gpu request (default: Null)
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

APIs can be integrated into other applications or services via their JSON endpoints. The endpoint for any API follows the following format: {apis_endpoint}/{app_name}/{api_name}.

The fields in the request payload for a particular API should match the raw columns that were used to train the model that it is serving. Cortex automatically applies the same transformers that were used at training time when responding to prediction requests.

## Horizontal Scalability

APIs can be configured using `replicas` in the `compute` field. Replicas can be used to change the amount of computing resources allocated to service prediction requests for a particular API. APIs that have low request volumes should have a small number of replicas while APIs that handle large request volumes should have more replicas.

## Rolling Updates

When the model that an API is serving gets updated, Cortex will update the API with the new model without any downtime.

## GPU Support
We recommend using GPU compute requests on API resources only if you have enough nodes in your cluster to support the number GPU requests in model training and API ideally with an autoscaler to manage the training nodes. Otherwise, due to the nature of zero downtime rolling updates your model training will not behave as expected as there will always be GPU nodes consumed by APIs from a previous deployment.
