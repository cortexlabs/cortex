# Batch API deployment

_WARNING: you are on the master branch, please refer to the docs on the branch that matches your `cortex version`_

Once your model is [exported](../exporting.md), you've implemented a [Predictor](predictors.md), and you've [configured your API](api-configuration.md), you're ready to deploy a Batch API.

## `cortex deploy`

The `cortex deploy` command collects your configuration and source code and deploys your API on your cluster:

```bash
$ cortex deploy

created image-classifier
```

APIs are declarative, so to update your API, you can modify your source code and/or configuration and run `cortex deploy` again.

After deploying a Batch API you can make the following requests to the API:
1. Submit a batch job
1. Get the status of a job
1. Stop a job

You can find the documentation for your deployed Batch API Endpoint Documentation [here](endpoints.md).

## `cortex get`

The `cortex get` command displays the status of all of your API:

```bash
$ cortex get

env   batch api          running jobs   latest job id                          last update
aws   image-classifier   1              69d9c0013c2d0d97 (submitted 30s ago)   46s

env     sync api          status   last update   avg request   2XX
aws     iris-classifier   live     10s           -             -
local   iris-classifier   live     11s           -             -
```

## `cortex get <api_name>`

 and `cortex get <api_name>` shows additional information about your Batch API and lists a summary of all of the currently running jobs and the most recently submitted jobs.

```bash
$ cortex get image-classifier

job id             status                    progress   failed   start time                 duration
69d9c0013c2d0d97   running                   1/24       0        29 Jul 2020 14:38:01 UTC   30s
69da5b1f8cd3b2d3   completed with failures   15/16      1        29 Jul 2020 13:38:01 UTC   5m20s
69da5bc32feb6aa0   succeeded                 40/40      0        29 Jul 2020 12:38:01 UTC   10m21s
69da5bd5b2f87258   succeeded                 34/34      0        29 Jul 2020 11:38:01 UTC   8m54s

endpoint: http://***.amazonaws.com/image-classifier
...
```

Appending the `--watch` flag will re-run the `cortex get` command every 2 seconds.

## Job commands

Once a job has been submitted to your Batch API (see [here](endpoints.md#submit-a-job)), you can use the Job ID from job submission to get the status, stream logs and stop a running job using the CLI.

### `cortex get <api_name> <job_id>`

After a submitting a job (see [Batch API endpoint documentation](endpoints.md)), you can use the

```bash
$ cortex get image-classifier 69d9c0013c2d0d97

job id: 69d9c0013c2d0d97
status: running

start time: 29 Jul 2020 14:38:01 UTC
end time:   -
duration:   32s

batch stats
total   succeeded   failed   avg time per batch
24      1           0        20s

worker stats
requested   running   failed   succeeded
2           2         0        0

job endpoint: https://://***..amazonaws.com/image-classifier/69d9c0013c2d0d97
```

### `cortex logs <api_name> <job_id>`

```bash
$ cortex logs image-classifier 69d9c0013c2d0d97

started enqueuing batches
partitioning 240 items found in job submission into 24 batches of size 10
completed enqueuing a total of 24 batches
spinning up workers...
2020-07-30 16:50:30.147522:cortex:pid-1:INFO:downloading the project code
2020-07-30 16:50:30.268987:cortex:pid-1:INFO:downloading the python serving image
....
```

### `cortex delete <api_name> <job_id>`

```bash
$ cortex delete image-classifier 69d9c0013c2d0d97

stopped job 69d96a01ea55da8c
```

## `cortex delete`

Use the `cortex delete` command to delete your API:

```bash
$ cortex delete my-api

deleting my-api
```

## Additional resources

<!-- CORTEX_VERSION_MINOR -->
* [Tutorial](../../examples/sklearn/iris-classifier/README.md) provides a step-by-step walkthough of deploying an iris classifier API
* [CLI documentation](../miscellaneous/cli.md) lists all CLI commands
* [Examples](https://github.com/cortexlabs/cortex/tree/master/examples) demonstrate how to deploy models from common ML libraries
