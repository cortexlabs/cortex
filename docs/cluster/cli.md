# CLI commands

## deploy

```text
create or update a deployment

usage:
  cortex deploy [flags]

flags:
  -e, --env string   environment (default "default")
  -f, --force        override the in-progress deployment update
  -h, --help         help for deploy
  -r, --refresh      re-deploy all apis with cleared cache and rolling updates
```

## get

```text
get information about resources

usage:
  cortex get [RESOURCE_NAME] [flags]

flags:
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

usage:
  cortex logs API_NAME [flags]

flags:
  -d, --deployment string   deployment name
  -e, --env string          environment (default "default")
  -h, --help                help for logs
```

## predict

```text
make a prediction request using a json file

usage:
  cortex predict API_NAME SAMPLE_FILE [flags]

flags:
      --debug               Predict with debug mode
  -d, --deployment string   deployment name
  -e, --env string          environment (default "default")
  -h, --help                help for predict
```

## delete

```text
delete a deployment

usage:
  cortex delete [DEPLOYMENT_NAME] [flags]

flags:
  -e, --env string   environment (default "default")
  -h, --help         help for delete
  -c, --keep-cache   keep cached data for the deployment
```

## cluster up

```text
spin up a cluster

usage:
  cortex cluster up [flags]

flags:
  -c, --config string   path to a cortex cluster configuration file
  -h, --help            help for up
```

## cluster info

```text
get information about a cluster

usage:
  cortex cluster info [flags]

flags:
  -c, --config string   path to a cortex cluster configuration file
  -h, --help            help for info
```

## cluster update

```text
update a cluster

usage:
  cortex cluster update [flags]

flags:
  -c, --config string   path to a cortex cluster configuration file
  -h, --help            help for update
```

## cluster down

```text
spin down a cluster

usage:
  cortex cluster down [flags]

flags:
  -c, --config string   path to a cortex cluster configuration file
  -h, --help            help for down
```

## version

```text
print the cli and cluster versions

usage:
  cortex version [flags]

flags:
  -e, --env string   environment (default "default")
  -h, --help         help for version
```

## configure

```text
configure the cli

usage:
  cortex configure [flags]

flags:
  -e, --env string   environment (default "default")
  -h, --help         help for configure
  -p, --print        print the configuration
```

## support

```text
send a support request to the maintainers

usage:
  cortex support [flags]

flags:
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

usage:
  cortex completion [flags]

flags:
  -h, --help   help for completion
```
