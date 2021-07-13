# Update

## Modify existing cluster

You can add/remove node groups, resize existing node groups and update a subset of the cluster configuration on a running cluster.

Fetch the current cluster configuration

```bash
cortex cluster info --print-config --name <cluster_name> --region <region> > cluster.yaml # fetch the current cluster configuration
```

Make the desired changes and apply your modifications

```
cortex cluster configure cluster.yaml
```

Cortex will calculate the difference and you will be prompted with the update plan. Only certain fields can be modified on a running cluster.

If you would like to update fields that can not be modified on a running cluster, you have to create a new cluster with the desired configuration.

## Update to a new version

```bash
# spin down your cluster
cortex cluster down --name <name> --region <region>

# update your CLI to the latest version
pip install --upgrade cortex

# confirm version
cortex version

# spin up your cluster
cortex cluster up cluster.yaml
```

## Update or upgrade without downtime

See [migration guide](../../guides/migrating.md) for details.
