# Update

## Prerequisites

1. [AWS credentials](aws.md)
2. [Docker](https://docs.docker.com/install)

## Upgrading Cortex

See [cluster configuration](config.md) for how to customize your installation.

<!-- CORTEX_VERSION_MINOR -->

```bash
# Update the CLI
bash -c "$(curl -sS https://raw.githubusercontent.com/cortexlabs/cortex/master/get-cli.sh)"

# Update the cluster
cortex cluster update

# Confirm version
cortex --version
```
