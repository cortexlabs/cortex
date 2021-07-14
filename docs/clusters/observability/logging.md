# Logging

Logs are collected with Fluent Bit and are exported to CloudWatch.

## Logs on AWS

Logs will automatically be pushed to CloudWatch and a log group with the same name as your cluster will be created to store your logs. API logs are tagged with labels to help with log aggregation and filtering. Log lines greater than 5 MB in size will be ignored.

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

## Streaming logs from the CLI

You can stream logs directly from a random pod of a running workload to iterate and debug quickly. These logs will not be as comprehensive as the logs that are available in CloudWatch.

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

## Exporting logs

You can export both the Cortex system logs and your application logs to your desired destination by configuring FluentBit.

### Configure kubectl

Follow these [instructions](../advanced/kubectl.md) to set up kubectl.

### Find supported destinations in FluentBit

Visit FluentBit's [output docs](https://docs.fluentbit.io/manual/concepts/data-pipeline/output) to see a list supported destinations.

Make sure to navigate to the version of FluentBit being used in your cluster. You can find the version of FluentBit by looking at the first view lines of one of the FluentBit pod logs.

Get the FluentBit pods:

```bash
kubectl get pods --selector app=fluent-bit
```

FluentBit's version should be in the first few log lines of a FluentBit pod:

```bash
kubectl logs fluent-bit-kxmzn | head -n 20
```

### Update FluentBit configuration

Define `patch.yaml` with your new output configuration:

```yaml
data:
  output.conf: |
    [OUTPUT]
        Name              es
        Match             k8s_container.*
        Host              https://abc123.us-west-2.es.amazonaws.com
        Port              443
        AWS_Region        us-west-2
        AWS_Auth          On
        tls               On
        Logstash_Format   On
        Logstash_Prefix   my-logs
```

Update FluentBit's configuration:

```bash
kubectl patch configmap fluent-bit-config --patch-file patch.yaml --type merge
```

### Restart FluentBit

Restart FluentBit to apply the new configuration:

```bash
kubectl rollout restart daemonset/fluent-bit
```
