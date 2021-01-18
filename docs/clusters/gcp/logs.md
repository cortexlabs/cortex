# Logs

By default, logs will be pushed to [StackDriver](https://console.cloud.google.com/logs/query) using fluent-bit. API logs are tagged with labels to help with log aggregation and filtering. Below are some sample Stackdriver queries:

RealtimeAPI:

```text
resource.type="k8s_container"
resource.labels.cluster_name="<INSERT CLUSTER NAME>"
jsonPayload.labels.apiKind="RealtimeAPI"
jsonPayload.labels.apiName="<INSERT API NAME>"
```

TaskAPI:

```text
resource.type="k8s_container"
resource.labels.cluster_name="<INSERT CLUSTER NAME>"
jsonPayload.labels.apiKind="TaskAPI"
jsonPayload.labels.apiName="<INSERT API NAME>"
jsonPayload.labels.jobID="<INSERT JOB ID>"
```

Please make sure to navigate to the project containing your cluster and adjust the time range accordingly before running queries.
