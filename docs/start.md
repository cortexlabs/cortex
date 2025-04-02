# Get started

## Create a cluster on your AWS account

<!-- CORTEX_VERSION_README -->
```bash
# install the CLI
bash -c "$(curl -sS https://raw.githubusercontent.com/cortexlabs/cortex/v0.42.2/get-cli.sh)"

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

* [Realtime](workloads/realtime/example.md) - create APIs that respond to requests in real-time.
* [Async](workloads/async/example.md) - create APIs that respond to requests asynchronously.
* [Batch](workloads/batch/example.md) - create APIs that run distributed batch jobs.
* [Task](workloads/task/example.md) - create APIs that run jobs on-demand.
