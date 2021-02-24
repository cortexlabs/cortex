# Install

## Install with pip

To install the latest version:

```bash
pip install cortex
```

<!-- CORTEX_VERSION_README x2 -->
To install or upgrade to a specific version (e.g. v0.29.0):

```bash
pip install cortex==0.29.0
```

To upgrade to the latest version:

```bash
pip install --upgrade cortex
```

## Install without the Python client

<!-- CORTEX_VERSION_README x2 -->
```bash
# For example to download CLI version 0.29.0 (Note the "v"):
bash -c "$(curl -sS https://raw.githubusercontent.com/cortexlabs/cortex/v0.29.0/get-cli.sh)"
```

By default, the Cortex CLI is installed at `/usr/local/bin/cortex`. To install the executable elsewhere, export the `CORTEX_INSTALL_PATH` environment variable to your desired location before running the command above.

## Changing the CLI/client configuration directory

By default, the Cortex CLI/client creates a directory at `~/.cortex/` and uses it to store environment configuration. To use a different directory, export the `CORTEX_CLI_CONFIG_DIR` environment variable before running any `cortex` commands.
