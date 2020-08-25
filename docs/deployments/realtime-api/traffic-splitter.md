# Traffic Splitter

_WARNING: you are on the master branch, please refer to the docs on the branch that matches your `cortex version`_

The Traffic Splitter feature allows you to split traffic between multiple Realtime APIs on your Cortex Cluster. This can be useful for A/B testing models in production.

After [deploying Realtime APIs](deployment.md), you can deploy an Traffic Splitter to provide a single endpoint that can route a request randomly to one of the target Realtime APIs. Weights can be assigned to Realtime APIs to control the percentage of requests routed to each API.

**Traffic Splitters are only supported on a Cortex cluster (in AWS).**

## Traffic Splitter Configuration

Traffic Splitter expects the target Realtime APIs to already be running or be included in the same configuration file as the Traffic Splitter. The traffic is routed according to the specified weights. The weights assigned to all Realtime APIs must to sum to 100.

```yaml
- name: <string>  # Traffic Splitter name (required)
  kind: TrafficSplitter  # must be "TrafficSplitter", create an Traffic Splitter which routes traffic to multiple Realtime APIs
  networking:
    endpoint: <string>  # the endpoint for the Traffic Splitter (default: <api_name>)
    api_gateway: public | none  # whether to create a public API Gateway endpoint for this API (if not, the load balancer will be accessed directly) (default: public)
  apis:  # list of Realtime APIs to target
    - name: <string>  # name of a Realtime API that is already running or is included in the same configuration file (required)
      weight: <int>   # percentage of traffic to route to the Realtime API (all weights must sum to 100) (required)
```

## `cortex deploy`

The `cortex deploy` command is used to deploy an Traffic Splitter.

```bash
$ cortex deploy

created traffic-splitter (TrafficSplitter)
```

Traffic Splitters are declarative, so to update your Traffic Splitter, you can modify the configuration and re-run `cortex deploy`.

## `cortex get`

The `cortex get` command displays the status of your Realtime APIs and Traffic Splitters, and `cortex get <api_name>` shows additional information about a specific Traffic Splitter.

```bash
$ cortex get traffic-splitter

apis             weights   status   requested   last update   avg request   2XX   5XX
another-my-api   80        live     1           5m            -             -     -
my-api           20        live     1           6m            -             -     -

last updated: 4m
endpoint: https://******.execute-api.eu-central-1.amazonaws.com/traffic-splitter
curl: curl https://******.execute-api.eu-central-1.amazonaws.com/traffic-splitter -X POST -H "Content-Type: application/json" -d @sample.json
...
```

## Making a prediction

You can use `curl` to test your Traffic Splitter. This will distribute the requests across the Realtime APIs targeted by the Traffic Splitter:

```bash
$ curl http://***.amazonaws.com/traffic-splitter \
    -X POST -H "Content-Type: application/json" \
    -d '{"key": "value"}'
```

## `cortex delete`

Use `cortex delete <api_name>` to delete your Traffic Splitter:

```bash
$ cortex delete traffic-splitter

deleted traffic-splitter
```

Note that this will not delete the Realtime APIs targeted by the Traffic Splitter.

## Additional resources

* [Traffic Splitter Tutorial](../../../examples/traffic-splitter/README.md) provides a step-by-step walkthrough for deploying an Traffic Splitter
* [Realtime API Tutorial](../../../examples/pytorch/text-generator/README.md) provides a step-by-step walkthrough of deploying a realtime API for text generation
* [CLI documentation](../../miscellaneous/cli.md) lists all CLI commands
