# Metrics

Cortex includes Prometheus for metrics collection and Grafana for visualization. You can monitor your APIs with the default Grafana dashboards, or create custom metrics and dashboards.

## Accessing the dashboard

The dashboard URL is displayed once you run a `cortex get <api_name>` command.

Alternatively, you can access it on `http://<operator_url>/dashboard`. Run the following command to get the operator
URL:

```bash
cortex env list
```

If your operator load balancer is configured to be internal, there are a few options for accessing the dashboard:

1. Access the dashboard from a machine that has VPC Peering configured to your cluster's VPC, or which is inside of your
   cluster's VPC.
1. Run `kubectl port-forward -n default grafana-0 3000:3000` to forward Grafana's port to your local machine, and access
   the dashboard on [http://localhost:3000](http://localhost:3000) (see instructions for setting up `kubectl` [here](../advanced/kubectl.md)).
1. Set up VPN access to your cluster's
   VPC ([docs](https://docs.aws.amazon.com/vpc/latest/userguide/vpn-connections.html)).

### Default credentials

The dashboard is protected with username / password authentication, which by default are:

- Username: admin
- Password: admin

You will be prompted to change the admin user password in the first time you log in.

Grafana allows managing the access of several users and managing teams. For more information on this topic check
the [grafana documentation](https://grafana.com/docs/grafana/latest/manage-users).

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

### Available metrics

Cortex exposes more metrics with Prometheus, that can be potentially useful. To check the available metrics, access the `Explore` menu in grafana and press the `Metrics` button.

![](https://user-images.githubusercontent.com/7456627/107377492-515f7000-6aeb-11eb-9b46-909120335060.png)

You can use any of these metrics to set up your own dashboards.

## Exporting metrics to monitoring solutions

You can scrape metrics from the in-cluster prometheus server via the `/federate` endpoint and push them to monitoring solutions such as Datadog.

The steps for exporting metrics from Prometheus will vary based on your monitoring solution. Here are a few high-level steps to get you started. We will be using Datadog as an example.

### Configure kubectl

Follow these [instructions](../../clusters/advanced/kubectl.md) to setup kubectl and point it to your cluster.

### Install agent

Monitoring solutions provide Kubernetes agents that are capable of scraping prometheus metrics. Follow the instructions to install the agent onto your cluster.

Here are the [instructions](https://docs.datadoghq.com/agent/kubernetes/?tab=helm#installation) for Datadog.

### Scrape prometheus

Some agents require a prometheus endpoint to scrape directly. You can provide `http://prometheus.default:9090/federate?match[]={job=~".+"}` as the target url to indicate that all metrics need to be scraped.

Some agents look for targets to scrape via annotations. You can find instructions below to update Cortex's prometheus server with the correct annotations to enable the agents to scrape it:

Create a `patch.yaml` and add the relevant annotations for your monitoring solution. Below is an example for [Datadog](https://docs.datadoghq.com/agent/kubernetes/prometheus/).

The annotations below indicate to the Datadog agents to scrape the prometheus server at the endpoint `/federate?match[]={job=~".+"}` and extract `cortex_in_flight_requests`. Note that Datadog specifically required the query params in the prometheus url to be encoded.

```yaml
spec:
  podMetadata:
    annotations:
      ad.datadoghq.com/prometheus.check_names: |
         ["prometheus"]
      ad.datadoghq.com/prometheus.init_configs: |
         [{}]
      ad.datadoghq.com/prometheus.instances: |
         [
            {
               "prometheus_url": "http://%%host%%:%%port%%/federate?match[]=%7Bjob%3D~%22.%2B%22%7D",
               "namespace": "cortex",
               "metrics": [{"cortex_in_flight_requests":"in_flight_requests"}]
            }
         ]
```

Update prometheus with your annotations:

```bash
kubectl patch prometheuses.monitoring.coreos.com prometheus --patch "$(cat patch.yaml)" --type merge
```

## Long term metric storage

Prometheus can be configured to write metrics to other monitoring solutions or databases for long term storage. You can attach a remote storage adapter to prometheus that will receive samples from prometheus and write to your destination. You can find a list of prometheus remote storage adapters [here](https://prometheus.io/docs/operating/integrations/#remote-endpoints-and-storage). More prometheus remote storage adapters can be found online if yours isn't on the list.

Once you've found an adapter that works for you, follow the steps below:

### Configure kubectl

Follow these [instructions](../../clusters/advanced/kubectl.md) to setup kubectl and point it to your cluster.

### Update prometheus

Define a `patch.yaml` file with your changes to the prometheus server:

```yaml
spec:
  containers: # container for your adapter
    ...
  remote_write:
    url: "http://localhost:9201/write" # http endpoint for your adapter
```

Update prometheus with your changes:

```bash
kubectl patch prometheuses.monitoring.coreos.com prometheus --patch "$(cat patch.yaml)" --type merge
```
