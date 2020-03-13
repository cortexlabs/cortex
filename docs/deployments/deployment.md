# API deployment

Once your model is [exported](exporting.md), you've implemented a [Predictor](predictors.md), and you've [configured your API](api-configuration.md), you're ready to deploy!

## `cortex deploy`

The `cortex deploy` command collects your configuration and source code and deploys your API on your cluster:

```bash
$ cortex deploy

creating my-api
```

APIs are declarative, so to update your API, you can modify your source code and/or configuration and run `cortex deploy` again.

## `cortex get`

The `cortex get` command displays the status of your APIs, and `cortex get <api_name>` shows additional information about a specific API.

```bash
$ cortex get my-api

status   up-to-date   requested   last update   avg request   2XX
live     1            1           1m            -             -

endpoint: http://***.amazonaws.com/iris-classifier
...
```

Appending the `--watch` flag will re-run the `cortex get` command every second.

## `cortex logs`

You can stream logs from your API using the `cortex logs` command:

```bash
$ cortex logs my-api
```

## Making a prediction

You can use `curl` to test your prediction service, for example:

```bash
$ curl http://***.amazonaws.com/my-api \
    -X POST -H "Content-Type: application/json" \
    -d '{"key": "value"}'
```

## Debugging

You can log information about each request by adding the `?debug=true` parameter to your requests. This will print the payload and the value after running your `predict()` function in the API logs.

## `cortex delete`

Use the `cortex delete` command to delete your API:

```bash
$ cortex delete my-api

deleting my-api
```

## Additional resources

<!-- CORTEX_VERSION_MINOR -->
* [Tutorial](../../examples/sklearn/iris-classifier/README.md) provides a step-by-step walkthough of deploying an iris classifier API
* [CLI documentation](../cluster-management/cli.md) lists all CLI commands
* [Examples](https://github.com/cortexlabs/cortex/tree/0.14/examples) demonstrate how to deploy models from common ML libraries
