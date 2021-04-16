# Update

## Update node group size

```bash
cortex cluster scale --node-group <node-group-name> --min-instances <min-instances> --max-instances <max-instances>
```

## Upgrade to a newer version

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

## Upgrade without downtime

In production environments, you can upgrade your cluster without downtime if you have a backend service or DNS in front of your Cortex cluster:

1. Spin up a new cluster. For example: `cortex cluster up new-cluster.yaml --configure-env new` (this will create a CLI environment named `new` for accessing the new cluster).
1. Re-deploy your APIs in your new cluster. For example, if the name of your CLI environment for your existing cluster is `previous`, you can use `cortex get --env previous` to list all running APIs in your cluster, and re-deploy them in the new cluster by changing directories to each API's project folder and running `cortex deploy --env new`. Alternatively, you can run `cortex cluster export --name <previous_cluster_name> --region <region>` to export all of your APIs (including configuration and application code), change directories into each API/ID subfolder that was exported, and run `cortex deploy --env new`.
1. Route requests to your new cluster.
    * If you are using a custom domain: update the A record in your Route 53 hosted zone to point to your new cluster's API load balancer.
    * If you have a backend service which makes requests to Cortex: update your backend service to make requests to the new cluster's endpoints.
    * If you have a self-managed API Gateway in front of your Cortex cluster: update the routes to use new cluster's endpoints.
1. Spin down your previous cluster. If you updated DNS settings, wait 24-48 hours before spinning down your previous cluster to allow the DNS cache to be flushed.
