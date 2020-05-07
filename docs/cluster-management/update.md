# Update

_WARNING: you are on the master branch, please refer to the docs on the branch that matches your `cortex version`_

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

In production environments, you can upgrade your cluster without downtime if you have a service in front of your Cortex cluster (for example, you can [configure API Gateway as a proxy service](../guides/api-gateway.md)): first spin up your new cluster, then update your client-facing service to route traffic to your new cluster, and then spin down your old cluster.
