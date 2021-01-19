# Logs

By default, logs will be pushed to [CloudWatch](https://us-west-2.console.aws.amazon.com/cloudwatch/home) using fluent-bit. A log group with the same name as your cluster will be created to store your logs. API logs are tagged with labels to help with log aggregation and filtering. Below are some sample CloudWatch Log Insight queries:

RealtimeAPI:

```text
fields @timestamp, log
| filter labels.apiName="<INSERT API NAME>"
| filter labels.apiKind="RealtimeAPI"
| sort @timestamp asc
| limit 1000
```

BatchAPI:

```text
fields @timestamp, log
| filter labels.apiName="<INSERT API NAME>"
| filter labels.jobID="<INSERT JOB ID>"
| filter labels.apiKind="BatchAPI"
| sort @timestamp asc
| limit 1000
```

TaskAPI:

```text
fields @timestamp, log
| filter labels.apiName="<INSERT API NAME>"
| filter labels.jobID="<INSERT JOB ID>"
| filter labels.apiKind="TaskAPI"
| sort @timestamp asc
| limit 1000
```

Please make sure to select the log group for your cluster and adjust the time range accordingly before running the queries.

## Structured logging

You can use Cortex's logger in your Python code to log in JSON, which will enrich your logs with Cortex's metadata, and enable you to add custom metadata to the logs. See the structured logging docs for [Realtime](../../workloads/realtime/predictors.md#structured-logging), [Batch](../../workloads/batch/predictors.md#structured-logging), and [Task](../../workloads/task/definitions.md#structured-logging) APIs.
