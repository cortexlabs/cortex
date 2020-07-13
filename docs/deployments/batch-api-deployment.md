# API deployment

_WARNING: you are on the master branch, please refer to the docs on the branch that matches your `cortex version`_

Once your model is [exported](exporting.md), you've implemented a [Predictor](predictors-batch.md), and you've [configured your API](api-configuration-batch.md), you're to create a BatchAPI.

## `cortex deploy`

The `cortex deploy` command collects your configuration and source code and creates an endpoint that can recieve job requests.

```bash
$ cortex deploy

created my-api
```

APIs are declarative, so to update your API, you can modify your source code and/or configuration and run `cortex deploy` again.

## `cortex get`

The `cortex get` command displays the status of your APIs, and `cortex get <api_name>` shows additional information about a specific API. # TODO

```bash
$ cortex get my-api

status   up-to-date   requested   last update   avg request   2XX
live     1            1           1m            -             -

endpoint: http://***.amazonaws.com/iris-classifier
...
```

Appending the `--watch` flag will re-run the `cortex get` command every second.

## Submit a Job

You can use `curl` to test your prediction service, for example:

```bash
$ curl http://***.amazonaws.com/my-api \
    -X POST -H "Content-Type: application/json" \
    -d '{"batches": "value"}'
```

## Get the status of the Job

## Stream logs for the job

## Stop the Job

Use the `cortex delete` command to delete your API:

```bash
$ cortex delete my-api

deleting my-api
```

## See job history

## Delete the API

## Additional resources

<!-- CORTEX_VERSION_MINOR -->
* [Tutorial](../../examples/sklearn/iris-classifier/README.md) provides a step-by-step walkthough of deploying an iris classifier API
* [CLI documentation](../miscellaneous/cli.md) lists all CLI commands
* [Examples](https://github.com/cortexlabs/cortex/tree/master/examples) demonstrate how to deploy models from common ML libraries
