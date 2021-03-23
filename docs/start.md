# Get started

## Create a cluster on your AWS account

```shell
# install the CLI
pip install cortex

# create a cluster
cortex cluster up cluster.yaml
```

* [Client installation](clients/install.md) - customize your client installation.
* [Cluster configuration](clusters/management/install.md) - optimize your cluster for your workloads.
* [Environments](clusters/management/environments.md) - manage multiple clusters.

## Run machine learning workloads at scale

```shell
# deploy machine learning APIs
cortex deploy apis.yaml
```

* [RealtimeAPI](workloads/realtime/example.md) - create APIs that respond to prediction requests in real-time.
* [AsyncAPI](workloads/async/example.md) - create APIs that respond to prediction requests asynchronously.
* [BatchAPI](workloads/batch/example.md) - create APIs that run distributed batch inference jobs.
* [TaskAPI](workloads/task/example.md) - create APIs that run training or fine-tuning jobs.
