# Logs

By default logs from a cluster on GCP will be pushed to [StackDriver](https://console.cloud.google.com/logs/query) using fluent-bit. API logs are tagged with labels to help with log aggregation and filtering.

Example query for a RealtimeAPI:

```text
resource.type="k8s_container"
resource.labels.cluster_name="<INSERT CLUSTER NAME>"
jsonPayload.labels.apiKind="RealtimeAPI"
jsonPayload.labels.apiName="<INSERT API NAME>"
```

Please make sure to navigate to the project containing your cluster and adjust the time range accordingly before running queries.
