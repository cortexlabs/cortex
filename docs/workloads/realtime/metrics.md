# Metrics

The `cortex get` and `cortex get API_NAME` commands display the request time (averaged over the past 2 weeks) and
response code counts (summed over the past 2 weeks) for your APIs:

```bash
cortex get

env   api                         status   up-to-date   requested   last update   avg request   2XX
aws   iris-classifier             live     1            1           17m           24ms          1223
aws   text-generator              live     1            1           8m            180ms         433
aws   image-classifier-resnet50   live     2            2           1h            32ms          1121126
```

The `cortex get API_NAME` command also provides a link to a Grafana dashboard:

![dashboard](https://user-images.githubusercontent.com/7456627/107253455-9c6b7b80-6a36-11eb-8600-f36a7bab6d3b.png)

---

## Metrics in the dashboard

| Panel             | Description                                                                        | Note                                                                                               |
|-------------------|------------------------------------------------------------------------------------|----------------------------------------------------------------------------------------------------|
| Request Rate      | Request rate, computed over every minute, of an API                                |                                                                                                    |
| In Flight Request | Active in-flight requests for an API.                                              | In-flight requests are recorded every 10 seconds, which will correspond to the minimum resolution. |
| Active Replicas   | Active replicas for an API                                                         |                                                                                                    |
| 2XX Responses     | Request rate, computed over a minute, for responses with status code 2XX of an API |                                                                                                    |
| 4XX Responses     | Request rate, computed over a minute, for responses with status code 4XX of an API |                                                                                                    |
| 5XX Responses     | Request rate, computed over a minute, for responses with status code 5XX of an API |                                                                                                    |
| p99 Latency       | 99th percentile latency, computed over a minute, for an API                        | Value might not be accurate because the histogram buckets are not dynamically set.                 |
| p90 Latency       | 90th percentile latency, computed over a minute, for an API                        | Value might not be accurate because the histogram buckets are not dynamically set.                 |
| p50 Latency       | 50th percentile latency, computed over a minute, for an API                        | Value might not be accurate because the histogram buckets are not dynamically set.                 |
| Average Latency   | Average latency, computed over a minute, for an API                                |                                                                                                    |

## Custom user metrics

It is possible to export custom user metrics by adding the `metrics_client`
argument to the predictor constructor. Below there is an example of how to use the metrics client with
the `PythonPredictor` type. The implementation would be similar to other predictor types.

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
