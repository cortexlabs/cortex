# Containers

## Handling requests

In order to handle requests to your Realtime API, one of your containers must run a web server which is listening for HTTP requests on the port which is configured in the `pod.port` field of your [API configuration](configuration.md) (default: 8080).

Subpaths are supported; for example, if your API is named `hello-world`, a request to `<load_balancer_url>/hello-world` will be routed to the root (`/`) of your web server, and a request to `<load_balancer_url>/hello-world/subpatch` will be routed to `/subpath` on your web server.

## Readiness checks

It is often important to implement a readiness check for your API. By default, as soon as your web server has bound to the port, it will start receiving traffic. In some cases, the web server may start listening on the port before its workers are ready to handle traffic (e.g. `tiangolo/uvicorn-gunicorn-fastapi` behaves this way). Readiness checks ensure that traffic is not sent into your web server before it's ready to handle them.

There are three types of readiness checks which are supported: `http_get`, `tcp_socket`, and `exec` (see [API configuration](configuration.md) for usage instructions). A simple and often effective approach is to add a route to your web server (e.g. `/healthz`) which responds with status code 200, and configure your readiness probe accordingly:

```yaml
readiness_probe:
  http_get:
    port: 8080
    path: /healthz
```

## Multiple containers

Your API pod can contain multiple containers, only one of which can be listening for requests on the target port (it can be any of the containers).

The `/mnt` directory is mounted to each container's file system, and is shared across all containers.

## Resource requests

Each container in the pod requests its own amount of CPU, memory, GPU, and Inferentia resources. In addition, Cortex's proxy sidecar container (which is automatically added to the pod) requests 100m CPU and 100Mi memory.

## Observability

See docs for [logging](../../clusters/observability/logging.md), [metrics](../../clusters/observability/metrics.md), and [alerting](../../clusters/observability/metrics.md).

## Using the Cortex CLI or client

It is possible to use the Cortex CLI or client to interact with your cluster's APIs from within your API containers. All containers will have a CLI configuration file present at `/cortex/client/cli.yaml`, which is configured to connect to the cluster. In addition, the `CORTEX_CLI_CONFIG_DIR` environment variable is set to `/cortex/client` by default. Therefore, no additional configuration is required to use the CLI or Python client (which can be instantiated via `cortex.client()`).

Note: your Cortex CLI or client must match the version of your cluster (available in the `CORTEX_VERSION` environment variable).

## Chaining APIs

It is possible to make requests to Realtime APIs from any Cortex API within a Cortex cluster. Requests can be made to `http://ingressgateway-apis.istio-system.svc.cluster.local/<api_name>`, where `<api_name>` is the name of the Realtime API you are making a request to.

For example, if there is a Realtime API named `hello-world` running in the cluster, you can make a request to it from a different API in Python by using:

```python
import requests

response = requests.post(
    "http://ingressgateway-apis.istio-system.svc.cluster.local/hello-world",
    json={"text": "hello world"},
)
```

Note that if the API making the request is a Realtime API or Async API, its autoscaling configuration (i.e. `target_in_flight`) should be modified with the understanding that requests will be considered "in-flight" in the first API as the request is being fulfilled by the second API.

To make requests from your Realtime API to a Batch, Async, or Task API running within the cluster, see the "Chaining APIs" docs associated with the target workload type.
