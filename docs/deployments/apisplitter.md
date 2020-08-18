# API Splitter

_WARNING: you are on the master branch, please refer to the docs on the branch that matches your `cortex version`_

The API Splitter feature allows you to split traffic between multiple Sync APIs on your Cortex Cluster. This can be useful for A/B testing models in production.

After [deploying Sync APIs](deployment.md) you can deploy an API Splitter to provide a single endpoint that can route a request randomly to one of the target Sync APIs. Weights can be assigned to Sync APIs to control the percentage of requests received by an individual Sync API.

**API Splitters are only supported on a Cortex cluster (in AWS).**

## API Splitter Configuration

API Splitter expects the specified Sync APIs to either be included in the deployment or already be running. The traffic is routed according to the defined weights. The weights of all Sync APIs must to sum to 100.

```yaml
- name: <string>  # API Splitter name (required)
  kind: APISplitter  # must be "APISplitter", create an API Splitter which routes traffic to multiple Sync APIs
  networking:
    endpoint: <string>  # the endpoint for the API Splitter (default: <api_name>)
    api_gateway: public | none  # whether to create a public API Gateway endpoint for this API (if not, the load balancer will be accessed directly) (default: public)
  apis:  # list of Sync APIs to use in API Splitter
    - name: <string>  # name of a Sync API that is already running or is a part of the deployment (required)
      weight: <int>   # percentage of traffic for the Sync API (all weights must sum to 100) (required)
```

## `cortex deploy`

The `cortex deploy` command is used to deploy an API Splitter.

```bash
$ cortex deploy

created my-apisplitter (APISplitter)
```

API Splitters are declarative, so to update your API Splitter, you can modify the configuration and run `cortex deploy` again.

## `cortex get`

The `cortex get` command displays the status of your Sync APIs and API Splitters, and `cortex get <api_name>` shows additional information about a specific API Splitter.

```bash
$ cortex get my-apisplitter

last updated: 4m

apis                      weights   status   requested   last update   avg request   2XX   5XX
another-my-api            80        live     1           5m            -             -     -
my-api                    20        live     1           6m            -             -     -

endpoint: https://******.execute-api.eu-central-1.amazonaws.com/my-apisplitter
curl: curl https://******.execute-api.eu-central-1.amazonaws.com/my-apisplitter -X POST -H "Content-Type: application/json" -d @sample.json
...
```

## Making a prediction

You can use `curl` to test your API Splitter service. This will distribute the requests across the defined Sync APIs in the API Splitter:

```bash
$ curl http://***.amazonaws.com/my-apisplitter \
    -X POST -H "Content-Type: application/json" \
    -d '{"key": "value"}'
```

## `cortex delete`

Use the `cortex delete` command to delete your API Splitter:

```bash
$ cortex delete my-apisplitter

deleted my-apisplitter
```

## Additional resources

<!-- CORTEX_VERSION_MINOR -->
* [API Splitter Tutorial](../../examples/apisplitter/README.md) provides a step-by-step walkthrough for deploying an API Splitter
* [Sync API Tutorial](../../examples/sklearn/iris-classifier/README.md) provides a step-by-step walkthough of deploying an iris classifier Sync API
* [CLI documentation](../miscellaneous/cli.md) lists all CLI commands
