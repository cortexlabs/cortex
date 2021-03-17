# Metrics

A cortex cluster includes a deployment of Prometheus for metrics collections and a deployment of Grafana for
visualization. You can monitor your APIs with the Grafana dashboards that ship with Cortex, or even add custom metrics
and dashboards.

## Accessing the dashboard

The dashboard URL is displayed once you run a `cortex get <api_name>` command.

Alternatively, you can access it on `http://<operator_url>/dashboard`. Run the following command to get the operator
URL:

```shell
cortex env list
```

If your operator load balancer is configured to be internal, there are a few options for accessing the dashboard:

1. Access the dashboard from a machine that has VPC Peering configured to your cluster's VPC, or which is inside of your
   cluster's VPC
1. Run `kubectl port-forward -n default grafana-0 3000:3000` to forward Grafana's port to your local machine, and access
   the dashboard on [http://localhost:3000/](http://localhost:3000/) (see instructions for setting up `kubectl`
   on [AWS](../../clusters/aws/kubectl.md) or [GCP](../../clusters/gcp/kubectl.md))
1. Set up VPN access to your cluster's
   VPC ([AWS docs](https://docs.aws.amazon.com/vpc/latest/userguide/vpn-connections.html))

### Default credentials

The dashboard is protected with username / password authentication, which by default are:

- Username: admin
- Password: admin

You will be prompted to change the admin user password in the first time you log in.

Grafana allows managing the access of several users and managing teams. For more information on this topic check
the [grafana documentation](https://grafana.com/docs/grafana/latest/manage-users/).

### Selecting an API

You can select one or more APIs to visualize in the top left corner of the dashboard.

![](https://user-images.githubusercontent.com/7456627/107375721-57545180-6ae9-11eb-9474-ba58ad7eb0c5.png)

### Selecting a time range

Grafana allows you to select a time range on which the metrics will be visualized. You can do so in the top right corner
of the dashboard.

![](https://user-images.githubusercontent.com/7456627/107376148-d9dd1100-6ae9-11eb-8c2b-c678b41ade01.png)

**Note: Cortex only retains a maximum of 2 weeks worth of data at any moment in time**

### Available dashboards

There are more than one dashboard available by default. You can view the available dashboards by accessing the Grafana
menu: `Dashboards -> Manage -> Cortex folder`.

The dashboards that Cortex ships with are the following:

- RealtimeAPI
- BatchAPI
- Cluster resources
- Node resources

## Exposed metrics

Cortex exposes more metrics with Prometheus, that can be potentially useful. To check the available metrics, access
the `Explore` menu in grafana and press the `Metrics` button.

![](https://user-images.githubusercontent.com/7456627/107377492-515f7000-6aeb-11eb-9b46-909120335060.png)

You can use any of these metrics to set up your own dashboards.

## Custom user metrics

It is possible to export your own custom metrics by using the `MetricsClient` class in your predictor code. This allows
you to create a custom metrics from your deployed API that can be later be used on your own custom dashboards.

Code examples on how to use custom metrics for each API kind can be found here:

- [RealtimeAPI](../realtime/metrics.md#custom-user-metrics)
- [RealtimeAPI](../async/metrics.md#custom-user-metrics)
- [BatchAPI](../batch/metrics.md#custom-user-metrics)
- [TaskAPI](../task/metrics.md#custom-user-metrics)

### Metric types

Currently, we only support 3 different metric types that will be converted to its respective Prometheus type:

- [Counter](https://prometheus.io/docs/concepts/metric_types/#counter) - a cumulative metric that represents a single
  monotonically increasing counter whose value can only increase or be reset to zero on restart.
- [Gauge](https://prometheus.io/docs/concepts/metric_types/#gauge) - a single numerical value that can arbitrarily go up
  and down.
- [Histogram](https://prometheus.io/docs/concepts/metric_types/#histogram) - samples observations (usually things like
  request durations or response sizes) and counts them in configurable buckets. It also provides a sum of all observed
  values.

### Pushing metrics

 - Counter

    ```python
    metrics.increment('my_counter', value=1, tags={"tag": "tag_name"})
    ```

 - Gauge

    ```python
    metrics.gauge('active_connections', value=1001, tags={"tag": "tag_name"})
    ```

 - Histogram

    ```python
    metrics.histogram('inference_time_milliseconds', 120, tags={"tag": "tag_name"})
    ```

### Metrics client class reference

```python
class MetricsClient:

    def gauge(self, metric: str, value: float, tags: Dict[str, str] = None):
        """
        Record the value of a gauge.

        Example:
        >>> metrics.gauge('active_connections', 1001, tags={"protocol": "http"})
        """
        pass

    def increment(self, metric: str, value: float = 1, tags: Dict[str, str] = None):
        """
        Increment the value of a counter.

        Example:
        >>> metrics.increment('model_calls', 1, tags={"model_version": "v1"})
        """
        pass

    def histogram(self, metric: str, value: float, tags: Dict[str, str] = None):
        """
        Set the value in a histogram metric

        Example:
        >>> metrics.histogram('inference_time_milliseconds', 120, tags={"model_version": "v1"})
        """
        pass
```
