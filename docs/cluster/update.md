# Update

## Prerequisites

1. [AWS credentials](aws.md)
2. [Docker](https://docs.docker.com/install)
3. See [cluster configuration](config.md) to learn how you can customize your cluster.

## Updating your cluster configuration

```bash
cortex cluster update
```

## Upgrading to a newer version of Cortex

<!-- CORTEX_VERSION_MINOR -->

```bash
# Spin down your cluster
cortex cluster down

# Update your CLI
bash -c "$(curl -sS https://raw.githubusercontent.com/cortexlabs/cortex/master/get-cli.sh)"

# Confirm version
cortex --version

# Spin up your cluster
cortex cluster up
```
