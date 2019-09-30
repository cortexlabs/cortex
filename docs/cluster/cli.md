# CLI Commands

## deploy

```text
This command sends all deployment configuration and code to Cortex.
If validations pass, Cortex will attempt to create the desired state.

Usage:
  cortex deploy [flags]

Flags:
  -e, --env string   environment (default "default")
  -f, --force        stop all running jobs
  -h, --help         help for deploy
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

## refresh

```text
This command sends all deployment configuration and code to Cortex.
If validations pass, Cortex will attempt to create the desired state,
and override the existing deployment.

Usage:
  cortex refresh [flags]

Flags:
  -e, --env string   environment (default "default")
  -f, --force        stop all running jobs
  -h, --help         help for refresh
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
```

## support

```text
This command sends a support request to Cortex developers.

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
