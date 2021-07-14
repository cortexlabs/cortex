# Containers

## Job specification

If you need access to any parameters in the job submission (e.g. `config`), the entire job specification is available at `/cortex/spec/job.json` in your API containers' filesystems.

## Multiple containers

Your Task's pod can contain multiple containers. The `/mnt` directory is mounted to each container's filesystem, and is shared across all containers.

## Observability

See docs for [logging](../../clusters/observability/logging.md), [metrics](../../clusters/observability/metrics.md), and [alerting](../../clusters/observability/metrics.md).

## Using the Cortex CLI or client

It is possible to use the Cortex CLI or client to interact with your cluster's APIs from within your API containers. All containers will have a CLI configuration file present at `/cortex/client/cli.yaml`, which is configured to connect to the cluster. In addition, the `CORTEX_CLI_CONFIG_DIR` environment variable is set to `/cortex/client` by default. Therefore, no additional configuration is required to use the CLI or Python client (which can be instantiated via `cortex.client()`).

Note: your Cortex CLI or client must match the version of your cluster (available in the `CORTEX_VERSION` environment variable).

## Chaining APIs

It is possible to submit Task jobs from any Cortex API within a Cortex cluster. Jobs can be submitted to `http://ingressgateway-operator.istio-system.svc.cluster.local/tasks/<api_name>`, where `<api_name>` is the name of the Task API you are making a request to.

For example, if there is a Task API named `hello-world` running in the cluster, you can make a request to it from a different API in Python by using:

```python
import requests

response = requests.post(
    "http://ingressgateway-operator.istio-system.svc.cluster.local/tasks/hello-world",
    json={"config": {"my_key": "my_value"}},
)
```

To make requests from your Task API to a Realtime, Batch, or Async API running within the cluster, see the "Chaining APIs" docs associated with the target workload type.
