# CLI commands

_WARNING: you are on the master branch, please refer to the docs on the branch that matches your `cortex version`_

## deploy

```text
create or update a deployment

Usage:
  cortex deploy [flags]

Flags:
  -e, --env string   environment (default "default")
  -f, --force        override the in-progress deployment update
  -h, --help         help for deploy
  -r, --refresh      re-deploy all apis with cleared cache and rolling updates
```

## get

```text
get information about deployments

Usage:
  cortex get [API_NAME] [flags]

Flags:
  -a, --all-deployments     list all deployments
  -d, --deployment string   deployment name
  -e, --env string          environment (default "default")
  -h, --help                help for get
  -s, --summary             show summarized output
  -v, --verbose             show verbose output
  -w, --watch               re-run the command every second
```

## logs

```text
stream logs from an api

Usage:
  cortex logs API_NAME [flags]

Flags:
  -d, --deployment string   deployment name
  -e, --env string          environment (default "default")
  -h, --help                help for logs
```

## predict

```text
make a prediction request using a json file

Usage:
  cortex predict API_NAME JSON_FILE [flags]

Flags:
      --debug               Predict with debug mode
  -d, --deployment string   deployment name
  -e, --env string          environment (default "default")
  -h, --help                help for predict
```

## delete

```text
delete a deployment

Usage:
  cortex delete [DEPLOYMENT_NAME] [flags]

Flags:
  -e, --env string   environment (default "default")
  -h, --help         help for delete
  -c, --keep-cache   keep cached data for the deployment
```

## cluster up

```text
spin up a cluster

Usage:
  cortex cluster up [flags]

Flags:
  -c, --config string   path to a cortex cluster configuration file
  -h, --help            help for up
```

## cluster info

```text
get information about a cluster

Usage:
  cortex cluster info [flags]

Flags:
  -c, --config string   path to a cortex cluster configuration file
  -h, --help            help for info
```

## cluster update

```text
update a cluster

Usage:
  cortex cluster update [flags]

Flags:
  -c, --config string   path to a cortex cluster configuration file
  -h, --help            help for update
```

## cluster down

```text
spin down a cluster

Usage:
  cortex cluster down [flags]

Flags:
  -c, --config string   path to a cortex cluster configuration file
  -h, --help            help for down
```

## version

```text
print the cli and cluster versions

Usage:
  cortex version [flags]

Flags:
  -e, --env string   environment (default "default")
  -h, --help         help for version
```

## configure

```text
configure the cli

Usage:
  cortex configure [flags]

Flags:
  -e, --env string   environment (default "default")
  -h, --help         help for configure
  -p, --print        print the configuration
```

## support

```text
send a support request to the maintainers

Usage:
  cortex support [flags]

Flags:
  -h, --help   help for support
```

## completion

```text
generate bash completion scripts

add this to your bashrc or bash profile:
  source <(cortex completion)
or run:
  echo 'source <(cortex completion)' >> ~/.bash_profile  # mac
  echo 'source <(cortex completion)' >> ~/.bashrc  # linux

this will also add the "cx" alias (note: cli completion requires the bash_completion package to be installed on your system)

Usage:
  cortex completion [flags]

Flags:
  -h, --help   help for completion
```
