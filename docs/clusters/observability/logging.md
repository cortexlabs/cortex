# Logging

By default, logs are collected with Fluent Bit, for every workloads, and are exported to CloudWatch. It is also possible to view the logs of a single replica using the `cortex logs` command.

## `cortex logs`

The CLI includes a command to get the logs for a single API replica for debugging purposes:

```bash
# RealtimeAPI
cortex logs <api_name>

# BatchAPI or TaskAPI
cortex logs <api_name> <job_id>  # the job needs to be in a running state
```

**Important:** this method won't show the logs for all the API replicas and therefore is not a complete logging
solution.

## Logs on AWS

Logs will automatically be pushed to CloudWatch and a log group with the same name as your cluster will be created to store your logs. API logs are tagged with labels to help with log aggregation and filtering.

Below are some sample CloudWatch Log Insight queries:

**RealtimeAPI:**

```text
fields @timestamp, message
| filter labels.apiName="<INSERT API NAME>"
| filter labels.apiKind="RealtimeAPI"
| sort @timestamp asc
| limit 1000
```

**AsyncAPI:**

```text
fields @timestamp, message
| filter labels.apiName="<INSERT API NAME>"
| filter labels.apiKind="AsyncAPI"
| sort @timestamp asc
| limit 1000
```

**BatchAPI:**

```text
fields @timestamp, message
| filter labels.apiName="<INSERT API NAME>"
| filter labels.jobID="<INSERT JOB ID>"
| filter labels.apiKind="BatchAPI"
| sort @timestamp asc
| limit 1000
```

**TaskAPI:**

```text
fields @timestamp, message
| filter labels.apiName="<INSERT API NAME>"
| filter labels.jobID="<INSERT JOB ID>"
| filter labels.apiKind="TaskAPI"
| sort @timestamp asc
| limit 1000
```

## Structured logging

You can use Cortex's logger in your Python code to log in JSON, which will enrich your logs with Cortex's metadata, and
enable you to add custom metadata to the logs.

See the structured logging docs for each API kind:

- [RealtimeAPI](../../workloads/realtime/predictors.md#structured-logging)
- [AsyncAPI](../../workloads/async/predictors.md#structured-logging)
- [BatchAPI](../../workloads/batch/predictors.md#structured-logging)
- [TaskAPI](../../workloads/task/definitions.md#structured-logging)
