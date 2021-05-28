# Logging

By default, logs are collected with Fluent Bit and are exported to CloudWatch. It is also possible to view the logs of a single replica using the `cortex logs` command.

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
| filter cortex_labels.apiName="<INSERT API NAME>"
| filter cortex_labels.apiKind="RealtimeAPI"
| sort @timestamp asc
| limit 1000
```

**AsyncAPI:**

```text
fields @timestamp, message
| filter cortex_labels.apiName="<INSERT API NAME>"
| filter cortex_labels.apiKind="AsyncAPI"
| sort @timestamp asc
| limit 1000
```

**BatchAPI:**

```text
fields @timestamp, message
| filter cortex_labels.apiName="<INSERT API NAME>"
| filter cortex_labels.jobID="<INSERT JOB ID>"
| filter cortex_labels.apiKind="BatchAPI"
| sort @timestamp asc
| limit 1000
```

**TaskAPI:**

```text
fields @timestamp, message
| filter cortex_labels.apiName="<INSERT API NAME>"
| filter cortex_labels.jobID="<INSERT JOB ID>"
| filter cortex_labels.apiKind="TaskAPI"
| sort @timestamp asc
| limit 1000
```
