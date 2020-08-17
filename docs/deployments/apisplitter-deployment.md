# APISplitter configuration

_WARNING: you are on the master branch, please refer to the docs on the branch that matches your `cortex version`_



Once your APIs are deployed you can deploy an APISplitter to split traffic between different SyncAPIs.

## `cortex deploy`

The `cortex deploy` command is used to deploy an APISplitter.

```bash
$ cortex deploy apisplitter.yaml

creating my-apisplitter
```

APISplitters are declarative, so to update your APISplitter, you can modify the configuration and run `cortex deploy` again.

## `cortex get`

The `cortex get` command displays the status of your SyncAPIs and APISplitters, and `cortex get <api_name>` shows additional information about a specific APISplitter.

```bash
$ cortex get my-apisplitter

kind: APISplitter

last updated: 4m

apis                      weights   status   requested   last update   avg request   2XX   5XX
another-my-api            80        live     1           5m            -             -     -
my-api                    20        live     1           6m            -             -     -

endpoint: https://******.execute-api.eu-central-1.amazonaws.com/my-apisplitter
curl: curl https://******.execute-api.eu-central-1.amazonaws.com/my-apisplitter -X POST -H "Content-Type: application/json" -d @sample.json
...
```

## Making a prediction

You can use `curl` to test your APISplitter service. This will distribute the requests across the defined SyncAPIs in the APISplitter:

```bash
$ curl http://***.amazonaws.com/my-apisplitter \
    -X POST -H "Content-Type: application/json" \
    -d '{"key": "value"}'
```

## `cortex delete`

Use the `cortex delete` command to delete your APISplitter:

```bash
$ cortex delete my-apisplitter

deleting my-apisplitter
```
