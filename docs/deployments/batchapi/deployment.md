# Batch API deployment

_WARNING: you are on the master branch, please refer to the docs on the branch that matches your `cortex version`_

Once your model is [exported](../exporting.md), you've implemented a [Predictor](predictors.md), and you've [configured your API](api-configuration.md), you're ready to deploy!

## `cortex deploy`

The `cortex deploy` command collects your configuration and source code and deploys your API on your cluster:

```bash
$ cortex deploy

created image-classifier
```

APIs are declarative, so to update your API, you can modify your source code and/or configuration and run `cortex deploy` again.

## `cortex get`

The `cortex get` command displays the status of all of your APIs, and `cortex get <api_name>` shows additional information about a specific API.

```bash
$ cortex get image-classifier

no submitted jobs

endpoint: http://***.amazonaws.com/image-classifier
...
```

Appending the `--watch` flag will re-run the `cortex get` command every 2 seconds.

## Available endpoints

A Batch API provides the following endpoints:

### Submit a job

Request: POST <batch_api_endpoint>
```yaml
{
    "workers": int,    # the number of workers you want to allocate for this job
    "item_list": {
        "items": [         # a list items that can be of any type
            <any>,
            <any>
        ],
        "batch_size": int, # the number of items in the items_list that should be in a batch
    }
        "config": {        # fields custom for this specific job (will override values specified in api configuration)
        "string": <any>
    }
}
```

NOTE: The maximum size of a request must be less than 10 MB. If your request needs to be more than 10 MB please write the payload in a newline delimited JSON format to a storage such S3 and use Delimited files to stream the file contents.

NOTE: A single item must not exceed 256 KB

Response:
```yaml
{
    "job_id": string,
    "api_name": string,
    "workers": int,
    "batches_per_worker": int,
    "config": {string: any},
    "api_id": string,
    "sqs_url": string,
    "created_time": string # e.g. 2020-07-16T14:56:10.276007415Z
}
```

### Get a Job Status

Request: GET <batch_api_endpoint>/<job_id>

Response:
```yaml
{
    "job_status": {
        "job_id": string,
        "api_name": string,
        "workers": int,
        "batches_per_worker": int,
        "config": {string: any},
        "api_id": string,
        "sqs_url": string,
        "status": string,   # string can take one of the following values: status_unknown|status_enqueuing|status_running|status_enqueue_failed|status_completed_with_failures|status_succeeded|status_unexpected_error|status_worker_error|status_worker_oom|status_stopped
        "batches_in_queue": int          # number of batches in queue
        "batch_metrics": {
            "succeeded": int
            "failed": int
            "avg_time_per_batch": float (optional)  # only available if batches have been completed
            "total": int
        },
        "worker_stats": {                # worker stats are only available when job status is running
            "pending": int,              # number of workers that are waiting for compute resources to be provisioned
            "initializing": int,         # number of workers that are initializing (downloading images, running your predictor's init function)
            "running": int,              # number of workers that are running and working on batches from the queue
            "succeeded": int,            # number of workers that have completed after verifying that the queue is empty
            "failed": int,               # number of workers that have failed
            "stalled": int,              # number of workers that have been stuck in pending for more than 10 minutes
        },
        "created_time": string          # e.g. 2020-07-16T14:56:10.276007415Z
        "start_time": string            # e.g. 2020-07-16T14:56:10.276007415Z
        "end_time": string (optional)   # e.g. 2020-07-16T14:56:10.276007415Z (only present if the job has completed)
    }
}
```

This information is available in the CLI via `cortex get <api_name> <job_id>`

### Stop a Job

Request: DELETE <batch_api_endpoint>/<job_id>

Response:
```
{"message":"stopped job <job_id>"}
```

You can stop a job using the CLI via `cortex delete <api_name> <job_id>

## `cortex get <api_name> <job_id>`

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
