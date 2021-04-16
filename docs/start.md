# Get started

## Create a cluster on your AWS account

```bash
# install the CLI
pip install cortex

# create a cluster
cortex cluster up cluster.yaml
```

* [Client installation](clients/install.md) - customize your client installation.
* [Cluster configuration](clusters/management/create.md) - optimize your cluster for your workloads.
* [Environments](clusters/management/environments.md) - manage multiple clusters.

## Build scalable APIs

```bash
# deploy APIs
cortex deploy apis.yaml
```

* [RealtimeAPI](workloads/realtime/example.md) - create HTTP/gRPC APIs that respond to requests in real-time.
* [AsyncAPI](workloads/async/example.md) - create APIs that respond to requests asynchronously.
* [BatchAPI](workloads/batch/example.md) - create APIs that run distributed batch jobs.
* [TaskAPI](workloads/task/example.md) - create APIs that run jobs on-demand.
