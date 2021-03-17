# Metrics

## Custom user metrics

It is possible to export custom user metrics by adding the `metrics_client`
argument to the predictor constructor.

```python
class PythonPredictor:
    def __init__(self, config, metrics_client):
        self.metrics = metrics_client

    def predict(self, payload):
        # --- my predict code here ---
        result = ...

        # increment a counter with name "my_metric" and tags model:v1
        self.metrics.increment(metric="my_counter", value=1, tags={"model": "v1"})

        # set the value for a gauge with name "my_gauge" and tags model:v1
        self.metrics.gauge(metric="my_gauge", value=42, tags={"model": "v1"})

        # set the value for an histogram with name "my_histogram" and tags model:v1
        self.metrics.histogram(metric="my_histogram", value=100, tags={"model": "v1"})
```

Refer to the [observability documentation](../observability/metrics.md#custom-user-metrics) for more information on
custom metrics.

**Note**: The metrics client uses the UDP protocol to push metrics, to be fault tolerant, so if it fails during a
metrics push there is no exception thrown.
