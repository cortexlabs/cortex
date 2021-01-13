# Task API endpoint

A deployed Task API endpoint supports the following:

1. Submitting a task job
1. Getting the status of a job
1. Stopping a job

You can find the url for your Task API using Cortex CLI command `cortex get <task_api_name>`.

## Submit a Job

```yaml
POST <task_api_endpoint>:
{
    "timeout": <int>,  # duration in seconds since the submission of a job before it is terminated (optional)
    "config": {  # custom fields for this specific job (will override values in `config` specified in your api configuration) (optional)
        "string": <any>
    }
}

RESPONSE:
{
    "job_id": <string>,
    "api_name": <string>,
    "kind": "TaskAPI",
    "workers": 1,
    "config": {<string>: <any>},
    "api_id": <string>,
    "timeout": <int>,
    "created_time": <string>  # e.g. 2020-07-16T14:56:10.276007415Z
}
```

## Job status

You can get the status of a job by making a GET request to `<task_api_endpoint>/<job_id>` (note that you can also get a job's status with the Cortex CLI command `cortex get <api_name> <job_id>`).

See [Job Status Codes](statuses.md) for a list of the possible job statuses and what they mean.

```yaml
GET <task_api_endpoint>?jobID=<jobID>:

RESPONSE:
{
    "job_status": {
        "job_id": <string>,
        "api_name": <string>,
        "kind": "TaskAPI",
        "workers": 1,
        "config": {<string>: <any>},
        "api_id": <string>,
        "status": <string>,   # will be one of the following values: status_unknown|status_running|status_succeeded|status_unexpected_error|status_worker_error|status_worker_oom|status_timed_out|status_stopped
        "created_time": <string>         # e.g. 2020-07-16T14:56:10.276007415Z
        "start_time": <string>           # e.g. 2020-07-16T14:56:10.276007415Z
        "end_time": <string> (optional)  # e.g. 2020-07-16T14:56:10.276007415Z (only present if the job has completed)
    },
    "endpoint": <string>   # endpoint for this job
    "api_spec": {
        ...
    }
}
```

## Stop a Job

Stop a job in progress. You can also use the Cortex CLI command

You stop a running job by making a DELETE request to `<task_api_endpoint>/<job_id>` (note that you can also delete a job with the Cortex CLI command `cortex delete <api_name> <job_id>`).

```yaml
DELETE <task_api_endpoint>?jobID=<jobID>:

RESPONSE:
{"message":"stopped job <job_id>"}
```
