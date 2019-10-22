# Update

## Prerequisites

1. [AWS credentials](aws.md)
2. [Docker](https://docs.docker.com/install)

## Upgrading Cortex

See [cluster configuration](config.md) to customize your installation.

<!-- CORTEX_VERSION_MINOR -->

```bash
# Update the CLI TODO
./cortex.sh install cli

# Update the cluster
cortex cluster update

# Confirm version
cortex --version
```
