# TaskAPI jobs

## Get the Task API's endpoint

```bash
cortex get <task_api_name>
```

## Submit a Job

```yaml
POST <task_api_endpoint>:
{
    "timeout": <int>,   # duration in seconds since the submission of a job before it is terminated (optional)
    "config": {         # arbitrary input for this specific job (optional)
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
    "created_time": <string>
}
```

The entire job specification is written to `/cortex/spec/job.json` in the API containers.

## Get a job's status

```bash
cortex get <task_api_name> <job_id>
```

Or make a GET request to `<task_api_endpoint>?jobID=<jobID>`:

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
        "status": <string>,
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
cortex delete <task_api_name> <job_id>
```

Or make a DELETE request to `<task_api_endpoint>?jobID=<jobID>`:

```yaml
DELETE <task_api_endpoint>?jobID=<jobID>:

RESPONSE:
{"message":"stopped job <job_id>"}
```
