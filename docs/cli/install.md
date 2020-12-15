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

## Environments

By default, the CLI has a single environment named `local`. When you create a cluster with `cortex cluster up`, an environment named `aws` or `gcp` is automatically created to point to your cluster. You can name the environment something else via the `--configure-env` flag, e.g. `cortex cluster up --configure-env prod`. You can also use the `--configure-env` flag with `cortex cluster info` and `cortex cluster configure` to create / update the specified environment.

### Example

```bash
cortex deploy         # uses local env; same as `cortex deploy --env local`
cortex logs my-api    # uses local env; same as `cortex logs my-api --env local`
cortex delete my-api  # uses local env; same as `cortex delete my-api --env local`

cortex cluster up       # configures the aws env; same as `cortex cluster up --configure-env aws`
cortex deploy --env aws
cortex deploy           # uses local env; same as `cortex deploy --env local`

# optional: change the default environment to aws
cortex env default aws    # sets aws as the default env
cortex deploy             # uses aws env; same as `cortex deploy --env aws`
cortex deploy --env local
```
