# Batch API endpoint

_WARNING: you are on the master branch, please refer to the docs on the branch that matches your `cortex version`_

Once your model is [exported](../../guides/exporting.md), you've implemented a [Predictor](predictors.md), you've [configured your API](api-configuration.md), and you've [deployed an api](deployment.md), you can submit and manage jobs by making HTTP requests to your Batch API endpoint.

A deployed Batch API endpoint supports the following:

1. Submitting a batch job
1. Getting the status of a job
1. Stopping a job

You can find the url for your Batch API using Cortex CLI command `cortex get <batch_api_name>`.

## Submit a Job

There are three options for providing the dataset for your job:

1. [Data in the request](#data-in-the-request)
1. [List S3 file paths](#s3-file-paths)
1. [Newline delimited JSON file(s) in S3](#newline-delimited-json-files-in-s3)

### Data in the request

The input data for your job can be included directly in your job submission request by specifying an `item_list` in your json request payload. Each item can be any type (object, list, string, etc.) and is treated as a single sample. `item_list.batch_size` specifies how many items to include in a single batch.

__Each batch must be smaller than 256 KiB, and the total request size must be less than 10 MiB.__ If you want to submit more data, explore the other job submission methods.

Submitting data in the request can be useful in the following scenarios:

* the request only has a few items
* each item in the request is small (e.g. urls to images/videos)
* you want to avoid using S3 as an intermediate storage layer

```yaml
POST <batch_api_endpoint>/:
{
    "workers": <int>,         # the number of workers to allocate for this job (required)
    "item_list": {
        "items": [            # a list items that can be of any type (required)
            <any>,
            <any>
        ],
        "batch_size": <int>,  # the number of items per batch (the predict() function is called once per batch) (required)
    }
    "config": {               # custom fields for this specific job (will override values in `config` specified in your api configuration) (optional)
        "string": <any>
    }
}

RESPONSE:
{
    "job_id": <string>,
    "api_name": <string>,
    "workers": <int>,
    "config": {<string>: <any>},
    "api_id": <string>,
    "sqs_url": <string>,
    "created_time": <string>  # e.g. 2020-07-16T14:56:10.276007415Z
}
```

### S3 file paths

If your input data is a list of files such as images/videos in an S3 directory, you can define `file_path_lister` in your submission request payload. You can use `file_path_lister.s3_paths` to specify a list of files or prefixes, and `file_path_lister.includes` and/or `file_path_lister.excludes` to remove unwanted files. The S3 file paths will be aggregated into batches of size `file_path_lister.batch_size`. To learn more about fine-grained S3 file filtering see [filtering files](#filtering-files).

__The total size of a batch must be less than 256 KiB.__

This submission pattern can be useful in the following scenarios:

* you have a list of images/videos in an S3 directory
* each S3 file represents a single sample or a small number of samples

If a single S3 file contains a lot of samples/rows, try the next submission strategy.

```yaml
POST <batch_api_endpoint>/:
{
    "workers": <int>,            # the number of workers to allocate for this job (required)
    "file_path_lister": {
        "s3_paths": [<string>],  # can be S3 prefixes or complete S3 paths (required)
        "includes": [<string>],  # glob patterns (optional)
        "excludes": [<string>],  # glob patterns (optional)
        "batch_size": <int>,     # the number of S3 file paths per batch (the predict() function is called once per batch) (required)
    }
    "config": {                  # custom fields for this specific job (will override values in `config` specified in your api configuration) (optional)
        "string": <any>
    }
}

RESPONSE:
{
    "job_id": <string>,
    "api_name": <string>,
    "workers": <int>,
    "config": {<string>: <any>},
    "api_id": <string>,
    "sqs_url": <string>,
    "created_time": <string>  # e.g. 2020-07-16T14:56:10.276007415Z
}
```

### Newline delimited JSON files in S3

If your input dataset is a newline delimited json file in an S3 directory (or a list of them), you can define `delimited_files` in your request payload to break up the contents of the file into batches of size `delimited_files.batch_size`.

Upon receiving `delimited_files`, your Batch API will iterate through the `delimited_files.s3_paths` to generate the set of S3 files to process. You can use `delimited_files.includes` and `delimited_files.excludes` to filter out unwanted files. Each S3 file will be parsed as a newline delimited JSON file. Each line in the file should be a JSON object, which will be treated as a single sample. The S3 file will be broken down into batches of size `delimited_files.batch_size` and submitted to your workers. To learn more about fine-grained S3 file filtering see [filtering files](#filtering-files).

__The total size of a batch must be less than 256 KiB.__

This submission pattern is useful in the following scenarios:

* one or more S3 files contains a large number of samples and must be broken down into batches

```yaml
POST <batch_api_endpoint>/:
{
    "workers": <int>,            # the number of workers to allocate for this job (required)
    "delimited_files": {
        "s3_paths": [<string>],  # can be S3 prefixes or complete S3 paths (required)
        "includes": [<string>],  # glob patterns (optional)
        "excludes": [<string>],  # glob patterns (optional)
        "batch_size": <int>,     # the number of json objects per batch (the predict() function is called once per batch) (required)
    }
    "config": {                  # custom fields for this specific job (will override values in `config` specified in your api configuration) (optional)
        "string": <any>
    }
}

RESPONSE:
{
    "job_id": <string>,
    "api_name": <string>,
    "workers": <int>,
    "config": {<string>: <any>},
    "api_id": <string>,
    "sqs_url": <string>,
    "created_time": <string>  # e.g. 2020-07-16T14:56:10.276007415Z
}
```

## Job status

You can get the status of a job by making a GET request to `<batch_api_endpoint>/<job_id>` (note that you can also get a job's status with the Cortex CLI command `cortex get <api_name> <job_id>`).

See [Job Status Codes](statuses.md) for a list of the possible job statuses and what they mean.

```yaml
GET <batch_api_endpoint>/<job_id>:

RESPONSE:
{
    "job_status": {
        "job_id": <string>,
        "api_name": <string>,
        "workers": <int>,
        "batches_per_worker": <int>,
        "config": {<string>: <any>},
        "api_id": <string>,
        "sqs_url": <string>,
        "status": <string>,   # will be one of the following values: status_unknown|status_enqueuing|status_running|status_enqueue_failed|status_completed_with_failures|status_succeeded|status_unexpected_error|status_worker_error|status_worker_oom|status_stopped
        "batches_in_queue": <int>        # number of batches remaining in the queue
        "batch_metrics": {
            "succeeded": <int>           # number of succeeded batches
            "failed": int                # number of failed batches
            "avg_time_per_batch": <float> (optional)  # only available if batches have been completed
        },
        "worker_counts": {               # worker counts are only available while a job is running
            "pending": <int>,            # number of workers that are waiting for compute resources to be provisioned
            "initializing": <int>,       # number of workers that are initializing (downloading images or running your predictor's init function)
            "running": <int>,            # number of workers that are actively working on batches from the queue
            "succeeded": <int>,          # number of workers that have completed after verifying that the queue is empty
            "failed": <int>,             # number of workers that have failed
            "stalled": <int>,            # number of workers that have been stuck in pending for more than 10 minutes
        },
        "created_time": <string>         # e.g. 2020-07-16T14:56:10.276007415Z
        "start_time": <string>           # e.g. 2020-07-16T14:56:10.276007415Z
        "end_time": <string> (optional)  # e.g. 2020-07-16T14:56:10.276007415Z (only present if the job has completed)
    },
    "api_spec": <string>,  # a base64 encoded string of your api configuration YAML that has been encoded in msgpack
    "endpoint": <string>   # endpoint for this job
}
```

## Stop a Job

Stop a job in progress. You can also use the Cortex CLI command

You stop a running job by making a DELETE request to `<batch_api_endpoint>/<job_id>` (note that you can also delete a job with the Cortex CLI command `cortex delete <api_name> <job_id>`).

```yaml
DELETE <batch_api_endpoint>/<job_id>:

RESPONSE:
{"message":"stopped job <job_id>"}
```

## Additional Information

### Filtering files

When submitting a job using `delimited_files` or `file_path_lister`, you can use `s3_paths` in conjunction with `includes` and `excludes` to precisely filter files.

The Batch API will iterate through each S3 path in `s3_paths`. If the S3 path is a prefix, it iterates through each file in that prefix. For each file, if `includes` is non-empty, it will discard the S3 path if the S3 file doesn't match any of the glob patterns provided in `includes`. After passing the `includes` filter (if specified), if the `excludes` is non-empty, it will discard the S3 path if the S3 files matches any of the glob patterns provided in `excludes`.

If you aren't sure which files will be processed in your request, specify the `dryRun=true` query parameter in the job submission request to see the target list.

Here are a few examples of filtering for a folder structure like this:

```text
├── s3://bucket
    └── images
        ├── img_1.png
        ├── img_2.jpg
        ├── img_3.jpg
        └── img_4.gif
```

Select all files

```yaml
{
    "s3_paths": ["s3://bucket/images/"]
}

# or

{
    "s3_paths": ["s3://bucket/images/img"]
}

# Would select the following files:
# s3://bucket/images/img_1.png
# s3://bucket/images/img_2.jpg
# s3://bucket/images/img_3.jpg
# s3://bucket/images/img_4.gif
```

Select specific files

```yaml
{
    "s3_paths": [
        "s3://bucket/images/img_1.png",
        "s3://bucket/images/img_2.jpg"
    ]
}

# Would select the following files:
# s3://bucket/images/img_1.png
# s3://bucket/images/img_2.jpg
```

Only select JPG files

```yaml
{
    "s3_paths": ["s3://bucket/images/"],
    "includes": ["**.jpg"]
}

# Would select the following files:
# s3://bucket/images/img_2.jpg
# s3://bucket/images/img_3.jpg
```

Select all JPG files except one specific JPG file

```yaml
{
    "s3_paths": ["s3://bucket/images/"],
    "includes": ["**.jpg"],
    "excludes": ["**_3.jpg"]
}

# Would select the file:
# s3://bucket/images/img_2.jpg
```

Select all files except GIFs

```yaml
{
    "s3_paths": ["s3://bucket/images/"],
    "excludes": ["**.gif"]
}

# Would select the files:
# s3://bucket/images/img_1.png
# s3://bucket/images/img_2.jpg
# s3://bucket/images/img_3.jpg
```
