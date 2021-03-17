# Logging

Cortex provides a logging solution, out-of-the-box, without the need to configure anything. By default, logs are
collected with FluentBit, on every API kind, and are exported to each cloud provider logging solution. It is also
possible to view the logs of a single API replica, while developing, through the `cortex logs` command.

## Cortex logs command

The cortex CLI tool provides a command to quickly check the logs for a single API replica while debugging.

To check the logs of an API run one of the following commands:

```shell
# RealtimeAPI
cortex logs <api_name>

# BatchAPI or TaskAPI
cortex logs <api_name> <job_id>  # the job needs to be in a running state
```

**Important:** this method won't show the logs for all the API replicas and therefore is not a complete logging
solution.

## Logs on AWS

For AWS clusters, logs will be pushed to [CloudWatch](https://console.aws.amazon.com/cloudwatch/home) using fluent-bit.
A log group with the same name as your cluster will be created to store your logs. API logs are tagged with labels to
help with log aggregation and filtering.

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

## Logs on GCP

Logs will be pushed to [StackDriver](https://console.cloud.google.com/logs/query) using fluent-bit. API logs are tagged
with labels to help with log aggregation and filtering.

Below are some sample Stackdriver queries:

**RealtimeAPI:**

```text
resource.type="k8s_container"
resource.labels.cluster_name="<INSERT CLUSTER NAME>"
labels.apiKind="RealtimeAPI"
labels.apiName="<INSERT API NAME>"
```

**TaskAPI:**

```text
resource.type="k8s_container"
resource.labels.cluster_name="<INSERT CLUSTER NAME>"
labels.apiKind="TaskAPI"
labels.apiName="<INSERT API NAME>"
labels.jobID="<INSERT JOB ID>"
```

Please make sure to navigate to the project containing your cluster and adjust the time range accordingly before running
queries.

## Structured logging

You can use Cortex's logger in your Python code to log in JSON, which will enrich your logs with Cortex's metadata, and
enable you to add custom metadata to the logs.

See the structured logging docs for each API kind:

- [RealtimeAPI](../../workloads/realtime/predictors.md#structured-logging)
- [AsyncAPI](../../workloads/async/predictors.md#structured-logging)
- [BatchAPI](../../workloads/batch/predictors.md#structured-logging)
- [TaskAPI](../../workloads/task/definitions.md#structured-logging)
