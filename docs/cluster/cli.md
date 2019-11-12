# CLI commands

## deploy

```text
This command sends all project configuration and code to Cortex.
If validations pass, Cortex will attempt to create the desired state.

Usage:
  cortex deploy [flags]

Flags:
  -e, --env string   environment (default "default")
  -f, --force        stop all running jobs
  -h, --help         help for deploy
  -r, --refresh      re-deploy all APIs
```

## get

```text
Get information about resources.

Usage:
  cortex get [RESOURCE_NAME] [flags]

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
This command streams logs from a deployed API.

Usage:
  cortex logs API_NAME [flags]

Flags:
  -d, --deployment string   deployment name
  -e, --env string          environment (default "default")
  -h, --help                help for logs
```

## predict

```text
This command makes a prediction request using
a JSON file and displays the response.

Usage:
  cortex predict API_NAME SAMPLE_FILE [flags]

Flags:
      --debug               Predict with debug mode
  -d, --deployment string   deployment name
  -e, --env string          environment (default "default")
  -h, --help                help for predict
```

## delete

```text
This command deletes a deployment from the cluster.

Usage:
  cortex delete [DEPLOYMENT_NAME] [flags]

Flags:
  -e, --env string   environment (default "default")
  -h, --help         help for delete
  -c, --keep-cache   keep cached data for the deployment
```

## cluster up

```text
This command spins up a Cortex cluster on your AWS account.

Usage:
  cortex cluster up [flags]

Flags:
  -c, --config string   path to a Cortex cluster configuration file
  -h, --help            help for up
```

## cluster info

```text
This command gets information about a Cortex cluster.

Usage:
  cortex cluster info [flags]

Flags:
  -c, --config string   path to a Cortex cluster configuration file
  -h, --help            help for info
```

## cluster update

```text
This command updates a Cortex cluster.

Usage:
  cortex cluster update [flags]

Flags:
  -c, --config string   path to a Cortex cluster configuration file
  -h, --help            help for update
```

## cluster down

```text
This command spins down a Cortex cluster.

Usage:
  cortex cluster down [flags]

Flags:
  -c, --config string   path to a Cortex cluster configuration file
  -h, --help            help for down
```

## version

```text
This command prints the version of the CLI and cluster.

Usage:
  cortex version [flags]

Flags:
  -e, --env string   environment (default "default")
  -h, --help         help for version
```

## configure

```text
This command configures the Cortex URL and AWS credentials
in order to authenticate and send requests to Cortex.
The configuration is stored in ~/.cortex.

Usage:
  cortex configure [flags]

Flags:
  -e, --env string   environment (default "default")
  -h, --help         help for configure
  -p, --print        print the configuration
```

## support

```text
This command sends a support request to the Cortex maintainers

Usage:
  cortex support [flags]

Flags:
  -h, --help   help for support
```

## completion

```text
Generate bash completion scripts.

Add this to your bashrc or bash profile:
  source <(cortex completion)
Or run:
  echo 'source <(cortex completion)' >> ~/.bash_profile  # Mac
  echo 'source <(cortex completion)' >> ~/.bashrc  # Linux

This will also add the "cx" alias.
Note: Cortex CLI completion requires the bash_completion package to be installed on your system.

Usage:
  cortex completion [flags]

Flags:
  -h, --help   help for completion
```
