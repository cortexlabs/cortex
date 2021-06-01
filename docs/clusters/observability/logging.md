# Logging

Logs are collected with Fluent Bit and are exported to CloudWatch.

## Logs on AWS

Logs will automatically be pushed to CloudWatch and a log group with the same name as your cluster will be created to store your logs. API logs are tagged with labels to help with log aggregation and filtering.

You can use the `cortex logs` command to get a CloudWatch Insights URL of query to fetch logs for your API. Please note that there may be a few minutes of delay from when a message is logged to when it is available in CloudWatch Insights.

**RealtimeAPI:**

```text
fields @timestamp, message
| filter cortex.labels.apiName="<INSERT API NAME>"
| filter cortex.labels.apiKind="RealtimeAPI"
| sort @timestamp asc
| limit 1000
```

**AsyncAPI:**

```text
fields @timestamp, message
| filter cortex.labels.apiName="<INSERT API NAME>"
| filter cortex.labels.apiKind="AsyncAPI"
| sort @timestamp asc
| limit 1000
```

**BatchAPI:**

```text
fields @timestamp, message
| filter cortex.labels.apiName="<INSERT API NAME>"
| filter cortex.labels.jobID="<INSERT JOB ID>"
| filter cortex.labels.apiKind="BatchAPI"
| sort @timestamp asc
| limit 1000
```

**TaskAPI:**

```text
fields @timestamp, message
| filter cortex.labels.apiName="<INSERT API NAME>"
| filter cortex.labels.jobID="<INSERT JOB ID>"
| filter cortex.labels.apiKind="TaskAPI"
| sort @timestamp asc
| limit 1000
```

## Streaming logs for an API or a running job

You can stream logs directly from a random pod of an API or a running job to iterate and debug quickly. These logs will not be as comprehensive as the logs that are available in CloudWatch.

```bash
# RealtimeAPI
cortex logs --random-pod <api_name>

# BatchAPI or TaskAPI
cortex logs --random-pod <api_name> <job_id>  # the job must be in a running state
```

## Structured logging

If you log JSON strings from your APIs, they will be automatically parsed before pushing to CloudWatch.

It is recommended to configure your JSON logger to use `message` or `msg` as the key for the log line if you would like the sample queries above to display the messages in your logs.

Avoid using top-level keys that start with "cortex" to prevent collisions with Cortex's internal logging.
