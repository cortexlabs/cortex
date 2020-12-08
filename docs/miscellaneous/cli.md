# CLI commands

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

### deploy

```text
create or update apis

Usage:
  cortex deploy [CONFIG_FILE] [flags]

Flags:
  -e, --env string      environment to use (default "local")
  -f, --force           override the in-progress api update
  -y, --yes             skip prompts
  -o, --output string   output format: one of pretty|json (default "pretty")
  -h, --help            help for deploy
```

### get

```text
get information about apis or jobs

Usage:
  cortex get [API_NAME] [JOB_ID] [flags]

Flags:
  -e, --env string      environment to use (default "local")
  -w, --watch           re-run the command every 2 seconds
  -o, --output string   output format: one of pretty|json (default "pretty")
  -v, --verbose         show additional information (only applies to pretty output format)
  -h, --help            help for get
```

### logs

```text
stream logs from an api

Usage:
  cortex logs API_NAME [JOB_ID] [flags]

Flags:
  -e, --env string   environment to use (default "local")
  -h, --help         help for logs
```

### patch

```text
update API configuration for a deployed API

Usage:
  cortex patch [CONFIG_FILE] [flags]

Flags:
  -e, --env string      environment to use (default "local")
  -f, --force           override the in-progress api update
  -o, --output string   output format: one of pretty|json (default "pretty")
  -h, --help            help for patch
```

### refresh

```text
restart all replicas for an api (without downtime)

Usage:
  cortex refresh API_NAME [flags]

Flags:
  -e, --env string      environment to use (default "local")
  -f, --force           override the in-progress api update
  -o, --output string   output format: one of pretty|json (default "pretty")
  -h, --help            help for refresh
```

### predict

```text
make a prediction request using a json file

Usage:
  cortex predict API_NAME JSON_FILE [flags]

Flags:
  -e, --env string   environment to use (default "local")
  -h, --help         help for predict
```

### delete

```text
delete any kind of api or stop a batch job

Usage:
  cortex delete API_NAME [JOB_ID] [flags]

Flags:
  -e, --env string      environment to use (default "local")
  -f, --force           delete the api without confirmation
  -c, --keep-cache      keep cached data for the api
  -o, --output string   output format: one of pretty|json (default "pretty")
  -h, --help            help for delete
```

### cluster up

```text
spin up a cluster on aws

Usage:
  cortex cluster up [flags]

Flags:
  -c, --config string               path to a cluster configuration file
      --aws-key string              aws access key id
      --aws-secret string           aws secret access key
      --cluster-aws-key string      aws access key id to be used by the cluster
      --cluster-aws-secret string   aws secret access key to be used by the cluster
  -e, --configure-env string        name of environment to configure (default "aws")
  -y, --yes                         skip prompts
  -h, --help                        help for up
```

### cluster info

```text
get information about a cluster

Usage:
  cortex cluster info [flags]

Flags:
  -c, --config string          path to a cluster configuration file
  -n, --name string            name of the cluster
  -r, --region string          aws region of the cluster
      --aws-key string         aws access key id
      --aws-secret string      aws secret access key
  -e, --configure-env string   name of environment to configure
  -d, --debug                  save the current cluster state to a file
  -y, --yes                    skip prompts
  -h, --help                   help for info
```

### cluster configure

```text
update a cluster's configuration

Usage:
  cortex cluster configure [flags]

Flags:
  -c, --config string               path to a cluster configuration file
      --aws-key string              aws access key id
      --aws-secret string           aws secret access key
      --cluster-aws-key string      aws access key id to be used by the cluster
      --cluster-aws-secret string   aws secret access key to be used by the cluster
  -e, --configure-env string        name of environment to configure
  -y, --yes                         skip prompts
  -h, --help                        help for configure
```

### cluster down

```text
spin down a cluster

Usage:
  cortex cluster down [flags]

Flags:
  -c, --config string       path to a cluster configuration file
  -n, --name string         name of the cluster
  -r, --region string       aws region of the cluster
      --aws-key string      aws access key id
      --aws-secret string   aws secret access key
  -y, --yes                 skip prompts
  -h, --help                help for down
```

### cluster export

```text
download the code and configuration for APIs

Usage:
  cortex cluster export [API_NAME] [API_ID] [flags]

Flags:
  -c, --config string       path to a cluster configuration file
  -n, --name string         name of the cluster
  -r, --region string       aws region of the cluster
      --aws-key string      aws access key id
      --aws-secret string   aws secret access key
  -h, --help                help for export
```

### env configure

```text
configure an environment

Usage:
  cortex env configure [ENVIRONMENT_NAME] [flags]

Flags:
  -p, --provider string                set the provider without prompting
  -o, --operator-endpoint string       set the operator endpoint without prompting
  -k, --aws-access-key-id string       set the aws access key id without prompting
  -s, --aws-secret-access-key string   set the aws secret access key without prompting
  -r, --aws-region string              set the aws region without prompting
  -h, --help                           help for configure
```

### env list

```text
list all configured environments

Usage:
  cortex env list [flags]

Flags:
  -o, --output string   output format: one of pretty|json (default "pretty")
  -h, --help            help for list
```

### env default

```text
set the default environment

Usage:
  cortex env default [ENVIRONMENT_NAME] [flags]

Flags:
  -h, --help   help for default
```

### env delete

```text
delete an environment configuration

Usage:
  cortex env delete [ENVIRONMENT_NAME] [flags]

Flags:
  -h, --help   help for delete
```

### version

```text
print the cli and cluster versions

Usage:
  cortex version [flags]

Flags:
  -e, --env string   environment to use (default "local")
  -h, --help         help for version
```

### completion

```text
generate shell completion scripts

to enable cortex shell completion:
    bash:
        add this to ~/.bash_profile (mac) or ~/.bashrc (linux):
            source <(cortex completion bash)

        note: bash-completion must be installed on your system; example installation instructions:
            mac:
                1) install bash completion:
                   brew install bash-completion
                2) add this to your ~/.bash_profile:
                   source $(brew --prefix)/etc/bash_completion
                3) log out and back in, or close your terminal window and reopen it
            ubuntu:
                1) install bash completion:
                   apt update && apt install -y bash-completion  # you may need sudo
                2) open ~/.bashrc and uncomment the bash completion section, or add this:
                   if [ -f /etc/bash_completion ] && ! shopt -oq posix; then . /etc/bash_completion; fi
                3) log out and back in, or close your terminal window and reopen it

    zsh:
        option 1:
            add this to ~/.zshrc:
                source <(cortex completion zsh)
            if that failed, you can try adding this line (above the source command you just added):
                autoload -Uz compinit && compinit
        option 2:
            create a _cortex file in your fpath, for example:
                cortex completion zsh > /usr/local/share/zsh/site-functions/_cortex

Note: this will also add the "cx" alias for cortex for convenience

Usage:
  cortex completion SHELL [flags]

Flags:
  -h, --help   help for completion
```
