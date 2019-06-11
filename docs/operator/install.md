# Install

## Prerequisites

1. [Virutalbox](virtualbox.md)

## Download the install script

<!-- CORTEX_VERSION_MINOR -->

```bash
# Download
curl -O https://raw.githubusercontent.com/cortexlabs/cortex/master/cortex-installer-local.sh

# Change permissions
chmod +x cortex-installer-local.sh

./cortex-install-local.sh install all
```

## Deploy an application

<!-- CORTEX_VERSION_MINOR -->

```bash
# Clone the Cortex repository
git clone -b master https://github.com/cortexlabs/cortex.git

# Navigate to the iris classification example
cd cortex/examples/iris

# Deploy the application to the cluster
cortex deploy

# View the status of the deployment
cortex status --watch

# Classify a sample
cortex predict iris-type irises.json
```

## Cleanup

```
# Delete the deployment
$ cortex delete iris
```

See [uninstall](uninstall.md) if you'd like to uninstall Cortex.
