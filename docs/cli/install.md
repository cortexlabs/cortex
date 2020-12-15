# Install

_WARNING: you are on the master branch, please refer to the docs on the branch that matches your `cortex version`_

## Install with pip

```bash
pip install cortex
```

## Install without the Python client

```bash
# Replace `VERSION` with the complete CLI version
$ bash -c "$(curl -sS https://raw.githubusercontent.com/cortexlabs/cortex/vVERSION/get-cli.sh)"

# Example command to download CLI version 0.18.1 (note the "v")
$ bash -c "$(curl -sS https://raw.githubusercontent.com/cortexlabs/cortex/v0.18.1/get-cli.sh)"
```

By default, the Cortex CLI is installed at `/usr/local/bin/cortex`. To install the executable elsewhere, export the `CORTEX_INSTALL_PATH` environment variable to your desired location before running the command above.

By default, the Cortex CLI creates a directory at `~/.cortex/` and uses it to store environment configuration. To use a different directory, export the `CORTEX_CLI_CONFIG_DIR` environment variable before running a `cortex` command.
