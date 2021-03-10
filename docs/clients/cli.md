# CLI commands

## deploy

```text
create or update apis

Usage:
  cortex deploy [CONFIG_FILE] [flags]

Flags:
  -e, --env string      environment to use
  -f, --force           override the in-progress api update
  -y, --yes             skip prompts
  -o, --output string   output format: one of pretty|json (default "pretty")
  -h, --help            help for deploy
```

## get

```text
get information about apis or jobs

Usage:
  cortex get [API_NAME] [JOB_ID] [flags]

Flags:
  -e, --env string      environment to use
  -w, --watch           re-run the command every 2 seconds
  -o, --output string   output format: one of pretty|json (default "pretty")
  -v, --verbose         show additional information (only applies to pretty output format)
  -h, --help            help for get
```

## logs

```text
stream logs from a single replica of an api or a single worker for a job

Usage:
  cortex logs API_NAME [JOB_ID] [flags]

Flags:
  -e, --env string   environment to use
  -y, --yes          skip prompts
  -h, --help         help for logs
```

## patch

```text
update API configuration for a deployed API

Usage:
  cortex patch [CONFIG_FILE] [flags]

Flags:
  -e, --env string      environment to use
  -f, --force           override the in-progress api update
  -o, --output string   output format: one of pretty|json (default "pretty")
  -h, --help            help for patch
```

## refresh

```text
restart all replicas for an api (without downtime)

Usage:
  cortex refresh API_NAME [flags]

Flags:
  -e, --env string      environment to use
  -f, --force           override the in-progress api update
  -o, --output string   output format: one of pretty|json (default "pretty")
  -h, --help            help for refresh
```

## delete

```text
delete any kind of api or stop a batch job

Usage:
  cortex delete API_NAME [JOB_ID] [flags]

Flags:
  -e, --env string      environment to use
  -f, --force           delete the api without confirmation
  -c, --keep-cache      keep cached data for the api
  -o, --output string   output format: one of pretty|json (default "pretty")
  -h, --help            help for delete
```

## cluster up

```text
spin up a cluster on aws

Usage:
  cortex cluster up [CLUSTER_CONFIG_FILE] [flags]

Flags:
  -e, --configure-env string   name of environment to configure (default "aws")
  -y, --yes                    skip prompts
  -h, --help                   help for up
```

## cluster info

```text
get information about a cluster

Usage:
  cortex cluster info [flags]

Flags:
  -c, --config string          path to a cluster configuration file
  -n, --name string            name of the cluster
  -r, --region string          aws region of the cluster
  -e, --configure-env string   name of environment to configure
  -d, --debug                  save the current cluster state to a file
  -y, --yes                    skip prompts
  -h, --help                   help for info
```

## cluster configure

```text
update a cluster's configuration

Usage:
  cortex cluster configure [CLUSTER_CONFIG_FILE] [flags]

Flags:
  -e, --configure-env string   name of environment to configure
  -y, --yes                    skip prompts
  -h, --help                   help for configure
```

## cluster down

```text
spin down a cluster

Usage:
  cortex cluster down [flags]

Flags:
  -c, --config string   path to a cluster configuration file
  -n, --name string     name of the cluster
  -r, --region string   aws region of the cluster
  -y, --yes             skip prompts
      --keep-volumes    keep cortex provisioned persistent volumes
  -h, --help            help for down
```

## cluster export

```text
download the code and configuration for APIs

Usage:
  cortex cluster export [API_NAME] [API_ID] [flags]

Flags:
  -c, --config string   path to a cluster configuration file
  -n, --name string     name of the cluster
  -r, --region string   aws region of the cluster
  -h, --help            help for export
```

## cluster-gcp up

```text
spin up a cluster on gcp

Usage:
  cortex cluster-gcp up [CLUSTER_CONFIG_FILE] [flags]

Flags:
  -e, --configure-env string   name of environment to configure (default "gcp")
  -y, --yes                    skip prompts
  -h, --help                   help for up
```

## cluster-gcp info

```text
get information about a cluster

Usage:
  cortex cluster-gcp info [flags]

Flags:
  -c, --config string          path to a cluster configuration file
  -n, --name string            name of the cluster
  -p, --project string         gcp project id
  -z, --zone string            gcp zone of the cluster
  -e, --configure-env string   name of environment to configure
  -d, --debug                  save the current cluster state to a file
  -y, --yes                    skip prompts
  -h, --help                   help for info
```

## cluster-gcp down

```text
spin down a cluster

Usage:
  cortex cluster-gcp down [flags]

Flags:
  -c, --config string    path to a cluster configuration file
  -n, --name string      name of the cluster
  -p, --project string   gcp project id
  -z, --zone string      gcp zone of the cluster
  -y, --yes              skip prompts
      --keep-volumes     keep cortex provisioned persistent volumes
  -h, --help             help for down
```

## env configure

```text
configure an environment

Usage:
  cortex env configure [ENVIRONMENT_NAME] [flags]

Flags:
  -o, --operator-endpoint string   set the operator endpoint without prompting
  -h, --help                       help for configure
```

## env list

```text
list all configured environments

Usage:
  cortex env list [flags]

Flags:
  -o, --output string   output format: one of pretty|json (default "pretty")
  -h, --help            help for list
```

## env default

```text
set the default environment

Usage:
  cortex env default [ENVIRONMENT_NAME] [flags]

Flags:
  -h, --help   help for default
```

## env delete

```text
delete an environment configuration

Usage:
  cortex env delete [ENVIRONMENT_NAME] [flags]

Flags:
  -h, --help   help for delete
```

## version

```text
print the cli and cluster versions

Usage:
  cortex version [flags]

Flags:
  -e, --env string   environment to use
  -h, --help         help for version
```

## completion

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
