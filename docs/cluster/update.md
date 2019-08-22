# Update

## Prerequisites

1. [AWS credentials](aws.md)
2. [Docker](https://docs.docker.com/install)

## Upgrading Cortex

See [cluster configuration](config.md) to customize your installation.

<!-- CORTEX_VERSION_MINOR -->

```bash
# Download
curl -O https://raw.githubusercontent.com/cortexlabs/cortex/master/cortex.sh

# Change permissions
chmod +x cortex.sh

# Set AWS credentials
export AWS_ACCESS_KEY_ID=***
export AWS_SECRET_ACCESS_KEY=***

# Update Cortex
./cortex.sh update

# Update the CLI
./cortex.sh install cli

# Confirm version
cortex --version
```
