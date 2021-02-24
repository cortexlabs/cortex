# BatchAPI jobs

## Get the TaskAPI endpoint

```bash
cortex get <batch_api_name>
```

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
POST <batch_api_endpoint>:
{
    "workers": <int>,         # the number of workers to allocate for this job (required)
    "timeout": <int>,         # duration in seconds since the submission of a job before it is terminated (optional)
    "sqs_dead_letter_queue": {      # specify a queue to redirect failed batches (optional)
        "arn": <string>,            # arn of dead letter queue e.g. arn:aws:sqs:us-west-2:123456789:failed.fifo
        "max_receive_count": <int>  # number of a times a batch is allowed to be handled by a worker before it is considered to be failed and transferred to the dead letter queue (must be >= 1)
    },
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
    "kind": "BatchAPI",
    "workers": <int>,
    "config": {<string>: <any>},
    "api_id": <string>,
    "sqs_url": <string>,
    "timeout": <int>,
    "sqs_dead_letter_queue": {
        "arn": <string>,
        "max_receive_count": <int>
    },
    "created_time": <string>
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
POST <batch_api_endpoint>:
{
    "workers": <int>,            # the number of workers to allocate for this job (required)
    "timeout": <int>,            # duration in seconds since the submission of a job before it is terminated (optional)
    "sqs_dead_letter_queue": {      # specify a queue to redirect failed batches (optional)
        "arn": <string>,            # arn of dead letter queue e.g. arn:aws:sqs:us-west-2:123456789:failed.fifo
        "max_receive_count": <int>  # number of a times a batch is allowed to be handled by a worker before it is considered to be failed and transferred to the dead letter queue (must be >= 1)
    },
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
    "kind": "BatchAPI",
    "workers": <int>,
    "config": {<string>: <any>},
    "api_id": <string>,
    "sqs_url": <string>,
    "timeout": <int>,
    "sqs_dead_letter_queue": {
        "arn": <string>,
        "max_receive_count": <int>
    },
    "created_time": <string>
}
```

### Newline delimited JSON files in S3

If your input dataset is a newline delimited json file in an S3 directory (or a list of them), you can define `delimited_files` in your request payload to break up the contents of the file into batches of size `delimited_files.batch_size`.

Upon receiving `delimited_files`, your Batch API will iterate through the `delimited_files.s3_paths` to generate the set of S3 files to process. You can use `delimited_files.includes` and `delimited_files.excludes` to filter out unwanted files. Each S3 file will be parsed as a newline delimited JSON file. Each line in the file should be a JSON object, which will be treated as a single sample. The S3 file will be broken down into batches of size `delimited_files.batch_size` and submitted to your workers. To learn more about fine-grained S3 file filtering see [filtering files](#filtering-files).

__The total size of a batch must be less than 256 KiB.__

This submission pattern is useful in the following scenarios:

* one or more S3 files contains a large number of samples and must be broken down into batches

```yaml
POST <batch_api_endpoint>:
{
    "workers": <int>,            # the number of workers to allocate for this job (required)
    "timeout": <int>,            # duration in seconds since the submission of a job before it is terminated (optional)
    "sqs_dead_letter_queue": {      # specify a queue to redirect failed batches (optional)
        "arn": <string>,            # arn of dead letter queue e.g. arn:aws:sqs:us-west-2:123456789:failed.fifo
        "max_receive_count": <int>  # number of a times a batch is allowed to be handled by a worker before it is considered to be failed and transferred to the dead letter queue (must be >= 1)
    },
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
    "kind": "BatchAPI",
    "workers": <int>,
    "config": {<string>: <any>},
    "api_id": <string>,
    "sqs_url": <string>,
    "timeout": <int>,
    "sqs_dead_letter_queue": {
        "arn": <string>,
        "max_receive_count": <int>
    },
    "created_time": <string>
}
```

## Get a job's status

```bash
cortex get <batch_api_name> <job_id>
```

Or make a GET request to `<batch_api_endpoint>?jobID=<jobID>`:

```yaml
GET <batch_api_endpoint>?jobID=<jobID>:

RESPONSE:
{
    "job_status": {
        "job_id": <string>,
        "api_name": <string>,
        "kind": "BatchAPI",
        "workers": <int>,
        "config": {<string>: <any>},
        "api_id": <string>,
        "sqs_url": <string>,
        "status": <string>,
        "batches_in_queue": <int>        # number of batches remaining in the queue
        "batch_metrics": {
            "succeeded": <int>           # number of succeeded batches
            "failed": int                # number of failed attempts
            "avg_time_per_batch": <float> (optional)  # average time spent working on a batch (only considers successful attempts)
        },
        "worker_counts": {               # worker counts are only available while a job is running
            "pending": <int>,            # number of workers that are waiting for compute resources to be provisioned
            "initializing": <int>,       # number of workers that are initializing (downloading images or running your predictor's init function)
            "running": <int>,            # number of workers that are actively working on batches from the queue
            "succeeded": <int>,          # number of workers that have completed after verifying that the queue is empty
            "failed": <int>,             # number of workers that have failed
            "stalled": <int>,            # number of workers that have been stuck in pending for more than 10 minutes
        },
        "created_time": <string>
        "start_time": <string>
        "end_time": <string> (optional)
    },
    "endpoint": <string>
    "api_spec": {
        ...
    }
}
```

## Stop a job

```bash
cortex delete <batch_api_name> <job_id>
```

Or make a DELETE request to `<batch_api_endpoint>?jobID=<jobID>`:

```yaml
DELETE <batch_api_endpoint>?jobID=<jobID>:

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
