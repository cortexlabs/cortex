# Splitting traffic between APIs

This example shows how to split traffic between 2 different iris-classifiers deployed as Realtime APIs.

To deploy this example:

1. Determine your CLI Version `cortex version`
1. Clone the repo and switch to the current version; for example, if your cortex version is 0.18.1, run `git clone -b v0.18.1 https://github.com/cortexlabs/cortex`
1. Navigate to this example directory

## `cortex deploy`

```bash
$ cortex deploy --env aws

creating iris-classifier-onnx (RealtimeAPI)
creating iris-classifier-tf (RealtimeAPI)
created iris-classifier (TrafficSplitter)
```

## `cortex get`

```bash
$ cortex get

env   realtime api           status     up-to-date   requested   last update   avg request   2XX
aws   iris-classifier-onnx   updating   0            1           27s           -             -
aws   iris-classifier-tf     updating   0            1           27s           -             -

env   traffic splitter   apis                                            last update
aws   iris-classifier    iris-classifier-onnx:30 iris-classifier-tf:70   27s
```

## `cortex get iris-classifier`

```bash
$ cortex get iris-classifier --env aws

apis                   weights   status   requested   last update   avg request   2XX   5XX
iris-classifier-onnx   30        live     1           1m            -             -     -
iris-classifier-tf     70        live     1           1m            -             -     -

last updated: 1m
endpoint: http://***.elb.us-west-2.amazonaws.com/iris-classifier
example curl: curl http://***.elb.us-west-2.amazonaws.com/iris-classifier -X POST -H "Content-Type: application/json" -d @sample.json
...
```

## Make multiple requests

```bash
$ curl http://***.elb.us-west-2.amazonaws.com/iris-classifier -X POST -H "Content-Type: application/json" -d @sample.json
setosa

$ curl http://***.elb.us-west-2.amazonaws.com/iris-classifier -X POST -H "Content-Type: application/json" -d @sample.json
setosa

$ curl http://***.elb.us-west-2.amazonaws.com/iris-classifier -X POST -H "Content-Type: application/json" -d @sample.json
setosa

$ curl http://***.elb.us-west-2.amazonaws.com/iris-classifier -X POST -H "Content-Type: application/json" -d @sample.json
setosa

$ curl http://***.elb.us-west-2.amazonaws.com/iris-classifier -X POST -H "Content-Type: application/json" -d @sample.json
setosa

$ curl http://***.elb.us-west-2.amazonaws.com/iris-classifier -X POST -H "Content-Type: application/json" -d @sample.json
setosa
```

## `cortex get iris-classifier`

Notice the requests being routed to the different Realtime APIs based on their weights (the output below may not match yours):

```bash
$ cortex get iris-classifier --env aws

using aws environment


apis                   weights   status   requested   last update   avg request   2XX   5XX
iris-classifier-onnx   30        live     1           4m            6.00791 ms    1     -
iris-classifier-tf     70        live     1           4m            5.81867 ms    5     -

last updated: 4m
endpoint: http://***.elb.us-west-2.amazonaws.com/iris-classifier
example curl: curl http://***.elb.us-west-2.amazonaws.com/iris-classifier -X POST -H "Content-Type: application/json" -d @sample.json
...
```

## Cleanup

Use `cortex delete <api_name>` to delete the Traffic Splitter and the two Realtime APIs (note that the Traffic Splitter and each Realtime API must be deleted by separate `cortex delete` commands):

```bash
$ cortex delete iris-classifier --env aws

deleting iris-classifier

$ cortex delete iris-classifier-onnx  --env aws

deleting iris-classifier-onnx

$ cortex delete iris-classifier-tf --env aws

deleting iris-classifier-tf
```

Running `cortex delete <api_name>` will free up cluster resources and allow Cortex to scale down to the minimum number of instances you specified during cluster installation. It will not spin down your cluster.
