# Containers

## Handling requests

In order to handle requests to your Async API, one of your containers must run a web server which is listening for HTTP requests on the port which is configured in the `pod.port` field of your [API configuration](configuration.md) (default: 8080).

Requests will be sent to your web server via HTTP POST requests to the root path (`/`) as they are pulled off of the queue. The payload and the content type header of the HTTP request to your web server will match those of the original request to your Async API. In addition, the request's ID will be passed in via the "X-Cortex-Request-ID" header.

Your web server must respond with valid JSON (with the `Content-Type` header set to "application/json"). The response will remain queryable for 7 days.

## Readiness checks

It is often important to implement a readiness check for your API. By default, as soon as your web server has bound to the port, it will start receiving traffic. In some cases, the web server may start listening on the port before its workers are ready to handle traffic (e.g. `tiangolo/uvicorn-gunicorn-fastapi` behaves this way). Readiness checks ensure that traffic is not sent into your web server before it's ready to handle them.

There are two types of readiness checks which are supported: `http_get` and `tcp_socket` (see [API configuration](configuration.md) for usage instructions). A simple and often effective approach is to add a route to your web server (e.g. `/healthz`) which responds with status code 200, and configure your readiness probe accordingly:

```yaml
readiness_probe:
  http_get:
    port: 8080
    path: /healthz
```

## Multiple containers

Your API pod can contain multiple containers, only one of which can be listening for requests on the target port (it can be any of the containers).

The `/mnt` directory is mounted to each container's filesystem, and is shared across all containers.

## Resource requests

Each container in the pod requests its own amount of CPU, memory, GPU, and Inferentia resources. In addition, Cortex's dequeuer sidecar container (which is automatically added to the pod) requests 100m CPU and 100Mi memory.

## Observability

See docs for [logging](../../clusters/observability/logging.md), [metrics](../../clusters/observability/metrics.md), and [alerting](../../clusters/observability/metrics.md).

## Using the Cortex CLI or client

It is possible to use the Cortex CLI or client to interact with your cluster's APIs from within your API containers. All containers will have a CLI configuration file present at `/cortex/client/cli.yaml`, which is configured to connect to the cluster. In addition, the `CORTEX_CLI_CONFIG_DIR` environment variable is set to `/cortex/client` by default. Therefore, no additional configuration is required to use the CLI or Python client (which can be instantiated via `cortex.client()`).

Note: your Cortex CLI or client must match the version of your cluster (available in the `CORTEX_VERSION` environment variable).

## Chaining APIs

It is possible to submit requests to Async APIs from any Cortex API within a Cortex cluster. Requests can be made to `http://ingressgateway-apis.istio-system.svc.cluster.local/<api_name>`, where `<api_name>` is the name of the Async API you are making a request to.

For example, if there is an Async API named `hello-world` running in the cluster, you can make a request to it from a different API in Python by using:

```python
import requests

# make a request to an Async API
response = requests.post(
    "http://ingressgateway-apis.istio-system.svc.cluster.local/hello-world",
    json={"text": "hello world"},
)

# retreive a result from an Async API
response = requests.get("http://ingressgateway-apis.istio-system.svc.cluster.local/hello-world/<id>")
```

To make requests from your Async API to a Realtime, Batch, or Task API running within the cluster, see the "Chaining APIs" docs associated with the target workload type.
