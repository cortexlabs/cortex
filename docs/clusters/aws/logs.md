# Logs

A standard cluster on AWS streams logs to [CloudWatch](https://console.cloud.google.com/logs/query) using fluent-bit. A log group with the same name as your cluster will be created to store your logs. API logs are tagged with labels to help with log aggregation and filtering. Below are some sample CloudWatch Log Insight queries:

RealtimeAPI:

```text
fields @timestamp, log
| filter labels.apiName="INSERT API NAME"
| filter labels.apiKind="RealtimeAPI"
| sort @timestamp asc
| limit 1000
```

BatchAPI:

```text
fields @timestamp, log
| filter labels.apiName="INSERT API NAME"
| filter labels.jobID="INSERT JOB ID"
| filter labels.apiKind="BatchAPI"
| sort @timestamp asc
| limit 1000
```

Please make sure to select the log group for your cluster and adjust the time range accordingly before running the queries.
