# Prediction Monitoring

The predictions made by your API can be tracked using Cortex. 

```yaml
- kind: api
  name: <string>  # API name (required)
  ...
  tracker:
    key: <string>  # key to track (required if the response payload is a JSON object)
    model_type: <string>  # model type, must be "classification" or "regression" (required)
  ...
```

`tracker` can be configured to collect API prediction metrics and display real-time stats in `cortex get <api_name>`. The tracker looks for scalar values in the response payload (after the execution of the `post_inference` request handler, if provided). If the response payload is a JSON object, `key` can be set to extract the desired scalar value. For regression models, the tracker should be configured with `model_type: regression` to collect float values and display regression stats such as min, max and average. For classification models, the tracker should be configured with `model_type: classification` to collect integer or string values and display the class distribution.

## Example

```yaml
- kind: api
  name: iris
  python:
    inference: inference.py
  tracker:
    model_type: classification
```
