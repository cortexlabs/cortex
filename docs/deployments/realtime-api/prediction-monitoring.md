# Prediction monitoring

_WARNING: you are on the master branch, please refer to the docs on the branch that matches your `cortex version`_

You can configure your API to collect prediction metrics and display real-time stats in `cortex get <api_name>`. Cortex looks for scalar values in the response payload. If the response payload is a JSON object, `key` must be used to extract the desired scalar value.

```yaml
- name: my-api
  ...
  monitoring:
    model_type: <string>  # must be "classification" or "regression", so responses can be interpreted correctly (i.e. categorical vs continuous) (required)
    key: <string>  # the JSON key in the response payload of the value to monitor (required if the response payload is a JSON object)
  ...
```

For classification models, `monitoring` should be configured with `model_type: classification` to collect integer or string values and display the class distribution. For regression models, `monitoring` should be configured with `model_type: regression` to collect float values and display regression stats such as min, max, and average.

## Example

```yaml
- name: iris
  kind: RealtimeAPI
  predictor:
    type: python
    path: predictor.py
  monitoring:
    model_type: classification
```
