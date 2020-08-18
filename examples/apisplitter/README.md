# Splitting traffic between APIs

_WARNING: you are on the master branch, please refer to the examples on the branch that matches your `cortex version`_

This example shows how to split traffic between 2 different iris-classifiers deployed as SyncAPIs.

To deploy this example:

1. Determine your CLI Version `cortex version`
1. Clone the repo and switch to the current version by replacing `<CORTEX_VERSION>` with your CLI version: `git clone -b v<CORTEX_VERSION> https://github.com/cortexlabs/cortex` (e.g. if the output of `cortex version` is 0.18.1, the clone command would be `git clone -b v0.18.1 https://github.com/cortexlabs/cortex`)
1. Navigate to this example directory

## `cortex deploy`

```bash
$ cortex deploy --env aws

creating iris-classifier-onnx (SyncAPI)
creating iris-classifier-tf (SyncAPI)
created iris-classifier-apisplitter (APISplitter)
```

## `cortex get`

```bash
$ cortex get

env   sync api               status     up-to-date   requested   last update   avg request   2XX
aws   iris-classifier-onnx   updating   0            1           27s           -             -
aws   iris-classifier-tf     updating   0            1           27s           -             -

env   api splitter                  apis                                            last update
aws   iris-classifier-apisplitter   iris-classifier-onnx:30 iris-classifier-tf:70   27s
```

## `cortex get iris-classifier-apisplitter`

```bash
$ cortex get iris-classifier-apisplitter --env aws

last updated: 1m

apis                   weights   status   requested   last update   avg request   2XX   5XX
iris-classifier-onnx   30        live     1           1m            -             -     -
iris-classifier-tf     70        live     1           1m            -             -     -

endpoint: https://abcedefg.execute-api.us-west-2.amazonaws.com/iris-classifier-apisplitter
curl: curl https://abcedefg.execute-api.us-west-2.amazonaws.com/iris-classifier-apisplitter -X POST -H "Content-Type: application/json" -d @sample.json
...
```

## Make multiple requests

```bash
$ curl https://abcedefg.execute-api.us-west-2.amazonaws.com/iris-classifier-apisplitter -X POST -H "Content-Type: application/json" -d @sample.json
setosa

$ curl https://abcedefg.execute-api.us-west-2.amazonaws.com/iris-classifier-apisplitter -X POST -H "Content-Type: application/json" -d @sample.json
setosa

$ curl https://abcedefg.execute-api.us-west-2.amazonaws.com/iris-classifier-apisplitter -X POST -H "Content-Type: application/json" -d @sample.json
setosa

$ curl https://abcedefg.execute-api.us-west-2.amazonaws.com/iris-classifier-apisplitter -X POST -H "Content-Type: application/json" -d @sample.json
setosa

$ curl https://abcedefg.execute-api.us-west-2.amazonaws.com/iris-classifier-apisplitter -X POST -H "Content-Type: application/json" -d @sample.json
setosa

$ curl https://abcedefg.execute-api.us-west-2.amazonaws.com/iris-classifier-apisplitter -X POST -H "Content-Type: application/json" -d @sample.json
setosa
```

## `cortex get iris-classifier-apisplitter`

Notice the requests being routed to the different SyncAPIs based on their weights (the output below may not match yours):

```bash
$ cortex get iris-classifier-apisplitter --env aws

using aws environment

last updated: 4m

apis                   weights   status   requested   last update   avg request   2XX   5XX
iris-classifier-onnx   30        live     1           4m            6.00791 ms    1     -
iris-classifier-tf     70        live     1           4m            5.81867 ms    5     -

endpoint: https://comtf6hs64.execute-api.us-west-2.amazonaws.com/iris-classifier-apisplitter
curl: curl https://comtf6hs64.execute-api.us-west-2.amazonaws.com/iris-classifier-apisplitter -X POST -H "Content-Type: application/json" -d @sample.json
...
```

## Cleanup

Use `cortex delete <api_name>` to delete the API Splitter and the two SyncAPIs (note that the API Splitter and each Sync API must be deleted by separate `cortex delete` commands):

```bash
$ cortex delete iris-classifier-apisplitter --env aws

deleting iris-classifier-apisplitter

$ cortex delete iris-classifier-onnx  --env aws

deleting iris-classifier-onnx

$ cortex delete iris-classifier-tf --env aws

deleting iris-classifier-tf
```

Running `cortex delete <api_name>` will free up cluster resources and allow Cortex to scale down to the minimum number of instances you specified during cluster installation. It will not spin down your cluster.
