# Metrics

## Custom user metrics

It is possible to export custom user metrics by adding the `metrics_client`
argument to the Handler class constructor. Below there is an example of how to use the metrics client. The implementation is similar to all handler types.

```python
class Handler:
    def __init__(self, config, metrics_client):
        self.metrics = metrics_client

    def handle_batch(self, payload):
        # --- my handler code here ---
        result = ...

        # increment a counter with name "my_metric" and tags model:v1
        self.metrics.increment(metric="my_counter", value=1, tags={"model": "v1"})

        # set the value for a gauge with name "my_gauge" and tags model:v1
        self.metrics.gauge(metric="my_gauge", value=42, tags={"model": "v1"})

        # set the value for an histogram with name "my_histogram" and tags model:v1
        self.metrics.histogram(metric="my_histogram", value=100, tags={"model": "v1"})
```

**Note**: The metrics client uses the UDP protocol to push metrics, so if it fails during a metrics push, no exception is thrown.
