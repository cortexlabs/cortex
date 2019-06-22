# CLI Commands

## init

```
Initialize an application.

Usage:
  cortex init APP_NAME [flags]

Flags:
  -h, --help   help for init
```

The `init` command creates a scaffold for a new Cortex application.

## deploy

```
Deploy an application.

Usage:
  cortex deploy [flags]

Flags:
  -e, --env string   environment (default "dev")
  -f, --force        stop all running jobs
  -h, --help         help for deploy
```

The `deploy` command sends all application configuration and code to the operator. If all validations pass, the operator will attempt to create the desired state on the cluster.

## refresh

```
Delete cached resources and deploy.

Usage:
  cortex refresh [flags]

Flags:
  -e, --env string   environment (default "dev")
  -f, --force        stop all running jobs
  -h, --help         help for refresh
```

The `refresh` behaves similarly to the `deploy` command. The key difference is that `refresh` doesn't use any cached resource and will recreate all state using raw data from the data warehouse.

## predict

```
Make predictions.

Usage:
  cortex predict API_NAME SAMPLES_FILE [flags]

Flags:
  -a, --app string   app name
  -e, --env string   environment (default "dev")
  -h, --help         help for predict
  -j, --json         print the raw json response
```

The `predict` command converts samples from a JSON file into prediction requests and outputs the response. This command is useful for quickly testing model output.

## delete

```
Delete an application.

Usage:
  cortex delete [APP_NAME] [flags]

Flags:
  -e, --env string   environment (default "dev")
  -h, --help         help for delete
  -c, --keep-cache   keep cached data for the app
```

The `delete` command deletes an application's resources from the cluster.

## get

```
Get information about resources.

Usage:
  cortex get [RESOURCE_TYPE] [RESOURCE_NAME] [flags]

Resource Types:
  raw_column
  aggregate
  transformed_column
  training_dataset
  model
  api

Flags:
  -a, --app string   app name
  -e, --env string   environment (default "dev")
  -h, --help         help for get
  -s, --summary      show summarized output
  -w, --watch        re-run the command every 2 seconds
```

The `get` command outputs the current state of all resources on the cluster. Specifying a resource name provides a more detailed view of the configuration and state of that particular resource. Using the `-s` or `--summary` flag will show a summarized view of all resource statuses.

## logs

```
Get logs for a resource.

Usage:
  cortex logs [RESOURCE_TYPE] RESOURCE_NAME [flags]

Resource Types:
  raw_column
  aggregate
  transformed_column
  training_dataset
  model
  api

Flags:
  -a, --app string   app name
  -e, --env string   environment (default "dev")
  -h, --help         help for logs
  -v, --verbose      show verbose output
```

The `logs` command streams logs from the workload corresponding to the specified resource. For example, `cortex logs models dnn` will get the Cortex logs from the most recent training workload for `dnn`. Using the `-v` or `--verbose` flag will show all of the logs for the workload (not just Cortex's logs).

## configure

```
Configure the CLI.

Usage:
  cortex configure [flags]

Flags:
  -e, --env string   environment (default "dev")
  -h, --help         help for configure
```

The `configure` command is used to connect to the Cortex cluster. The CLI needs a Cortex operator URL as well as valid AWS credentials in order to authenticate requests.

The CLI stores this information in the `~/.cortex` directory.

## completion

```
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
