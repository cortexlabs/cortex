# Update

## Prerequisites

1. [Docker](https://docs.docker.com/install)
2. [AWS credentials](aws-credentials.md)

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

In production environments, you can upgrade your cluster without downtime if you have a service in front of your Cortex cluster (for example, a backend server or an external API Gateway): first spin up your new cluster, then update your client-facing service to route traffic to your new cluster, and then spin down your old cluster.

If you've set up HTTPS by specifying an SSL Certificate for a subdomain in your cluster configuration, you can upgrade your cluster with minimal downtime: first spin up a new cluster, then update the A record in your subdomain hosted zone to point to the API loadbalancer of your new cluster. Wait at least a 24 to 48 hours before spinning down your old cluster to allow old DNS cache to be flushed.
