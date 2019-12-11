# Prediction monitoring

_WARNING: you are on the master branch, please refer to the docs on the branch that matches your `cortex version`_

`tracker` can be configured to collect API prediction metrics and display real-time stats in `cortex get <api_name>`. The tracker looks for scalar values in the response payload. If the response payload is a JSON object, `key` can be set to extract the desired scalar value.

```yaml
- kind: api
  name: <string>  # API name (required)
  ...
  tracker:
    key: <string>  # the JSON key in the response to track (required if the response payload is a JSON object)
    model_type: <string>  # model type, must be "classification" or "regression" (required)
  ...
```

For classification models, the tracker should be configured with `model_type: classification` to collect integer or string values and display the class distribution. For regression models, the tracker should be configured with `model_type: regression` to collect float values and display regression stats such as min, max and average.

## Example

```yaml
- kind: api
  name: iris
  predictor:
    path: predictor.py
  tracker:
    model_type: classification
```
