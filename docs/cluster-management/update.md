# Update

## Prerequisites

1. [Docker](https://docs.docker.com/install)
2. [AWS credentials](aws.md)

## Updating your cluster configuration

See [cluster configuration](config.md) to learn how you can customize your cluster.

```bash
cortex cluster update
```

## Upgrading to a newer version of Cortex

```bash
# spin down your cluster
cortex cluster down

# update your CLI
bash -c "$(curl -sS https://raw.githubusercontent.com/cortexlabs/cortex/0.11/get-cli.sh)"

# confirm version
cortex version

# spin up your cluster
cortex cluster up
```

