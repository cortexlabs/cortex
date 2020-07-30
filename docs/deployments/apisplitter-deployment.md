# APISplitter configuration

_WARNING: you are on the master branch, please refer to the docs on the branch that matches your `cortex version`_



Once your APIs are deployed you can deploy an APISplitter to split traffic between different predictor APIs.

## `cortex deploy`

The `cortex deploy` command is used to deploy an APISplitter.

```bash
$ cortex deploy apisplitter.yaml

creating my-apisplitter
```

APISplitters are declarative, so to update your APISplitter, you can modify your source code and/or configuration and run `cortex deploy` again.

## `cortex get`

The `cortex get` command displays the status of your APIs, and `cortex get <api_name>` shows additional information about a specific API.

```bash
$ cortex get my-api

status   up-to-date   requested   last update   avg request   2XX
live     1            1           1m            -             -

endpoint: http://***.amazonaws.com/iris-classifier
...
```

## Making a prediction

You can use `curl` to test your prediction service. This will distribute the requests across the defined APIs in the APISplitter:

```bash
$ curl http://***.amazonaws.com/my-apisplitter \
    -X POST -H "Content-Type: application/json" \
    -d '{"key": "value"}'
```

## `cortex delete`

Use the `cortex delete` command to delete your API:

```bash
$ cortex delete my-apisplitter

deleting my-apisplitter
```

