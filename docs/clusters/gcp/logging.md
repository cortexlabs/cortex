# Logs

By default, logs will be pushed to [StackDriver](https://console.cloud.google.com/logs/query) using fluent-bit. API logs are tagged with labels to help with log aggregation and filtering. Below are some sample Stackdriver queries:

RealtimeAPI:

```text
resource.type="k8s_container"
resource.labels.cluster_name="<INSERT CLUSTER NAME>"
labels.apiKind="RealtimeAPI"
labels.apiName="<INSERT API NAME>"
```

TaskAPI:

```text
resource.type="k8s_container"
resource.labels.cluster_name="<INSERT CLUSTER NAME>"
labels.apiKind="TaskAPI"
labels.apiName="<INSERT API NAME>"
labels.jobID="<INSERT JOB ID>"
```

Please make sure to navigate to the project containing your cluster and adjust the time range accordingly before running queries.

## Structured logging

You can use Cortex's logger in your Python code to log in JSON, which will enrich your logs with Cortex's metadata, and enable you to add custom metadata to the logs. See the structured logging docs for [Realtime](../../workloads/realtime/predictors.md#structured-logging) and [Task](../../workloads/task/definitions.md#structured-logging) APIs.
