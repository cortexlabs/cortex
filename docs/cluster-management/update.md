# Update

_WARNING: you are on the master branch, please refer to the docs on the branch that matches your `cortex version`_

## Updating your cluster configuration

See [cluster configuration](config.md) to learn how you can customize your cluster.

```bash
cortex cluster configure
```

## Upgrading to a newer version of Cortex

<!-- CORTEX_VERSION_MINOR -->

```bash
# spin down your cluster
cortex cluster down

# update your CLI
bash -c "$(curl -sS https://raw.githubusercontent.com/cortexlabs/cortex/master/get-cli.sh)"

# confirm version
cortex version

# spin up your cluster
cortex cluster up
```

In production environments, you can upgrade your cluster without downtime if you have a backend service or DNS in front of your Cortex cluster:

1. Spin up a new cluster. For example: `cortex cluster up --config new-cluster.yaml --env new` (this will create a CLI environment named `new` for accessing the new cluster).
1. Re-deploy your APIs in your new cluster. For example, if the name of your CLI environment for your old cluster is `old`, you can use `cortex get --env old` to list all running APIs in your old cluster, and re-deploy them in the new cluster by changing directories to each API's project folder and running `cortex deploy --env new`.
1. Route requests to your new cluster.
    * If you are using a custom domain: update the A record in your Route 53 hosted zone to point to your new cluster's API Gateway (if you are using API Gateway) or API load balancer (if clients connect directly to the API load balancer).
    * If you have a backend service which makes requests to Cortex: update your backend service to make requests to the new cluster's endpoints.
    * If you have a self-managed API Gateway in front of your Cortex cluster: update the routes to use new cluster's endpoints.
1. Spin down your old cluster. If you updated DNS settings, wait 24-48 hours before spinning down your old cluster to allow the DNS cache to be flushed.
