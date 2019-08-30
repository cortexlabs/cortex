# CLI Commands

## deploy

```text
Create or update a deployment.

Usage:
  cortex deploy [flags]

Flags:
  -e, --env string   environment (default "dev")
  -f, --force        stop all running jobs
  -h, --help         help for deploy
```

The `deploy` command sends all deployment configuration and code to Cortex. If validations pass, Cortex will attempt to create the desired state on the cluster.

## get

```text
Get information about resources.

Usage:
  cortex get [RESOURCE_TYPE] [RESOURCE_NAME] [flags]

Flags:
  -a, --all-deployments     list all deployments
  -d, --deployment string   deployment name
  -e, --env string          environment (default "dev")
  -h, --help                help for get
  -s, --summary             show summarized output
  -v, --verbose             show verbose output
  -w, --watch               re-run the command every second
```

The `get` command displays the current state of all resources on the cluster. Specifying a resource name provides the state of the particular resource. A detailed view of the configuration and additional metdata of a specific resource can be retrieved by adding the `-v` or `--verbose` flag. Using the `-s` or `--summary` flag will show a summarized view of all resource statuses. A list of deployments can be displayed by specifying the `-a` or `--all-deployments` flag.

## logs

```text
Get logs for a resource.

Usage:
  cortex logs [RESOURCE_TYPE] RESOURCE_NAME [flags]

Flags:
  -d, --deployment string   deployment name
  -e, --env string          environment (default "dev")
  -h, --help                help for logs
```

The `logs` command streams logs from the workload corresponding to the specified resource. For example, `cortex logs api my-api` will stream the logs from the most recent api named `my-api`. `RESOURCE_TYPE` is optional (unless there are name colisions), so `cortex logs my-api` will also work.

## refresh

```text
Delete cached resources and deploy.

Usage:
  cortex refresh [flags]

Flags:
  -e, --env string   environment (default "dev")
  -f, --force        stop all running jobs
  -h, --help         help for refresh
```

The `refresh` command behaves similarly to the `deploy` command. The key difference is that `refresh` doesn't use any cached resources.

## predict

```text
Usage:
  cortex predict API_NAME SAMPLE_FILE [flags]

Flags:
      --debug               Predict with debug mode
  -d, --deployment string   deployment name
  -e, --env string          environment (default "dev")
  -h, --help                help for predict
```

The `predict` command converts a sample from a JSON file into a prediction request and displays the response. This command is useful for quickly testing predictions.

## delete

```text
Delete a deployment.

Usage:
  cortex delete [DEPLOYMENT_NAME] [flags]

Flags:
  -e, --env string   environment (default "dev")
  -h, --help         help for delete
  -c, --keep-cache   keep cached data for the deployment
```

The `delete` command deletes an deployment's resources from the cluster.

## configure

```text
Configure the CLI.

Usage:
  cortex configure [flags]

Flags:
  -e, --env string   environment (default "dev")
  -h, --help         help for configure
```

The `configure` command is used to connect to the Cortex cluster. The CLI needs a Cortex operator URL as well as valid AWS credentials in order to authenticate requests. The CLI stores this information in the `~/.cortex` directory.

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
  -h, --help
```
