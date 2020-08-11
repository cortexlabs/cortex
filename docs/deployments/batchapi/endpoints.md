# Batch API endpoints

_WARNING: you are on the master branch, please refer to the docs on the branch that matches your `cortex version`_

A deployed Batch API endpoint supports the following:

1. Submitting a batch job
1. Getting the status of a job
1. Stopping a job

You can find the url for your Batch API using Cortex CLI command `cortex get <batch_api_name>`.

## Submit a Job

There are three options for providing the dataset for your job:

1. [Include the dataset directly in the JSON request](#dataset-in-the-request)
1. [Provide a list of S3 file paths](#s3-files-paths)
1. [Provide a list newline delimited JSON files on S3 containing the input dataset](#newline-delimited-json-files-in-s3)

### Dataset in the request

The dataset for your job can be included directly in your job submission request providing an `item_list` in your request payload. Each item can be any type (object, list, string, etc.) and is treated as a single sample. `item_list.batch_size` determines how many items are included in a single batch. Make sure that the total size of a batch is less than 256 KiB.

__The total request size must be less than 10 MiB.__ If you want to submit more data, explore some of the other methods.

Submitting data in the request can be useful in the following use cases:

- avoiding the use of an intermediate storage layer such as S3
- small request
- each item in the request is small (e.g. urls to images/videos)

```yaml
POST <batch_api_url>/:
{
    "workers": int,        # the number of workers you want to allocate for this job
    "item_list": {
        "items": [         # a list items that can be of any type
            <any>,
            <any>
        ],
        "batch_size": int, # the number of items in the items_list that should be in a batch
    }
    "config": {            # fields custom for this specific job (will override values specified in your api configuration)
        "string": <any>
    }
}

RESPONSE:
{
    "job_id": string,
    "api_name": string,
    "workers": int,
    "config": {string: any},
    "api_id": string,
    "sqs_url": string,
    "created_time": string # e.g. 2020-07-16T14:56:10.276007415Z
}
```

### S3 file paths

If your input dataset is a list of files such as images/videos in an s3 directory, you can define `file_path_lister` in your request payload to generate a list of S3 file paths. You can use `file_path_lister.s3_paths` to specify a list of files or prefixes and`file_path_lister.includes` and `file_path_lister.excludes` to remove unwanted files. The list of s3 file paths will be broken up into batches of size `file_path_lister.batch_size`. To learn more about fine grained S3 file filtering see the [filter files](#filtering-files) section.

Make sure that the total size of a batch is less than 256 KiB.

This submission pattern can be useful in the following scenarios:

- you have a list of images/videos in an s3 directory
- each s3 file represents a single sample or a small number of samples

If a single S3 file contains a lot of samples/rows, try this submission strategy. // TODO

```yaml

POST <batch_api_url>/:
{
    "workers": int,        # the number of workers you want to allocate for this job
    "file_path_lister": {
        "s3_paths": [string],  # can be s3 prefixes or complete s3 paths
        "includes": [string],  # (optional) glob patterns (delimited by "/"), any s3 file that satisfies at least one glob pattern in this list will be included in the dataset
        "excludes": [string],  # (optional) glob patterns (delimited by "/"), any s3 file that satisfies at least one glob pattern will be excluded from the dataset
        "batch_size": int, # the number of items in the items_list that should be in a batch
    }
    "config": {            # fields custom for this specific job (will override values specified in api configuration)
        "string": <any>
    }
}

RESPONSE:
{
    "job_id": string,
    "api_name": string,
    "workers": int,
    "config": {string: any},
    "api_id": string,
    "sqs_url": string,
    "created_time": string # e.g. 2020-07-16T14:56:10.276007415Z
}
```

### Newline delimited JSON files in S3

If your input dataset is a list new line delimited json files in an s3 directory, you can define `delimited_files` in your request payload to break up the files into batches of size `delimited_files.batch_size` and submitted to your workers. Make sure that the total size of a batch is less than 256 KiB.

Upon receiving `delimited_files` your Batch API will iterate through the `delimited_files.s3_paths` and generate a collection of s3 files. You can use `delimited_files.includes` and `delimited_files.excludes` to filter out unwanted files. Each S3 file will be parsed as a newline delimited JSON file. Each JSON object will be treated as a single sample. The S3 file will be broken down into batches of specified size `delimited_files.batch_size` and submitted to your workers. To learn more about fine grained S3 file filtering see the [filter files](#filtering-files) section.

Make sure that the total size of a batch is less than 256 KiB.

This submission pattern is useful in the following scenarios:

- you have a list of JSON files in S3 that need
- an s3 file contains a large number of samples and must be broken down into batches.

```yaml
POST <batch_api_url>/:
{
    "workers": int,            # the number of workers you want to allocate for this job
    "delimited_files": {
        "s3_paths": [string],  # can be s3 prefixes or complete s3 paths
        "includes": [string],  # glob patterns (delimited by "/"), any s3 file that satisfies at least one glob pattern in this list will be included in the dataset (optional)
        "excludes": [string],  # glob patterns (delimited by "/"), any s3 file that satisfies at least one glob pattern will be excluded from the dataset (optional)
        "batch_size": int,     # the number of items in the items_list that should be in a batch (default: 1)
    }
    "config": {                # fields custom for this specific job (will override values specified in api configuration)
        "string": <any>
    }
}

RESPONSE:
{
    "job_id": string,
    "api_name": string,
    "workers": int,
    "config": {string: any},
    "api_id": string,
    "sqs_url": string,
    "created_time": string # e.g. 2020-07-16T14:56:10.276007415Z
}
```

## Job status


Response:

```yaml
GET <batch_api_url>/<job_id>:

RESPONSE:
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
        },
        "worker_counts": {                # worker stats are only available when job status is running
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

## Stop a Job

```yaml
DELETE <batch_api_url>/<job_id>:

RESPONSE:
{"message":"stopped job <job_id>"}
```


## Additional Information

### Filtering files


When submitting a job using `delimited_files` or `file_path_lister`, you can use `s3_paths` in conjunction with in `includes` and `excludes` to precisely filter files.

The Batch API will iterate through each s3 path in `s3_paths`. If the s3 path is a prefix, it iterates through each file in that prefix. For each file, if the `includes` is non empty, it will discard the s3 path if the s3 file doesn't match any of the glob patterns provided in `includes`. Then, if the `excludes` is non empty, it will discard the s3 path if the s3 files matches any of the glob patterns provided in `excludes`. The order matters.

If you aren't sure which files will be processed in your request, specify `dryRun=true` query parameter to get the target list.

Here are a few scenarios:

For a folder structure like this:

```
├── s3://bucket
    └── images
        ├── img_1.png
        ├── img_2.jpg
        ├── img_3.jpg
        └── img_4.gif
```

1. Select all files

```yaml
{
    "s3_paths": ["s3://bucket/images/img"]
}

# Would select the following files:
# s3://bucket/images/img_1.png
# s3://bucket/images/img_2.jpg
# s3://bucket/images/img_3.jpg
# s3://bucket/images/img_4.gif
```

1. Only select JPGs

```yaml
{
    "s3_paths": ["s3://bucket/images/img"],
    "includes": ["**.jpg"]
}

# Would select the following files:
# s3://bucket/images/img_2.jpg
# s3://bucket/images/img_3.jpg
```

1. Select all JPGs but one

```yaml
{
    "s3_paths": ["s3://bucket/images/img"],
    "includes": ["**.jpg"],
    "excludes": ["**_3.jpg"]
}

# Would select the file:
# s3://bucket/images/img_2.jpg
```

1. Select everything but GIFs
```yaml
{
    "s3_paths": ["s3://bucket/images/img"],
    "excludes": ["**.gif"]
}

# Would select the files:
# s3://bucket/images/img_1.png
# s3://bucket/images/img_2.jpg
# s3://bucket/images/img_3.jpg
```
