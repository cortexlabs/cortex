# CLI reference

_WARNING: you are on the master branch, please refer to the docs on the branch that matches your `cortex version`_

## Install the CLI

```bash
pip install cortex
```

## Install the CLI without Python Client

### Mac/Linux OS

```bash
# Replace `INSERT_CORTEX_VERSION` with the complete CLI version (e.g. 0.18.1):
$ bash -c "$(curl -sS https://raw.githubusercontent.com/cortexlabs/cortex/vINSERT_CORTEX_VERSION/get-cli.sh)"

# For example to download CLI version 0.18.1 (Note the "v"):
$ bash -c "$(curl -sS https://raw.githubusercontent.com/cortexlabs/cortex/v0.18.1/get-cli.sh)"
```

By default, the Cortex CLI is installed at `/usr/local/bin/cortex`. To install the executable elsewhere, export the `CORTEX_INSTALL_PATH` environment variable to your desired location before running the command above.

By default, the Cortex CLI creates a directory at `~/.cortex/` and uses it to store environment configuration. To use a different directory, export the `CORTEX_CLI_CONFIG_DIR` environment variable before running a `cortex` command.

### Windows

To install the Cortex CLI on a Windows machine, follow [this guide](../guides/windows-cli.md).

## Command overview
