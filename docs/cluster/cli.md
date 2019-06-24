# CLI Commands

## deploy

```
Create or update a deployment.

Usage:
  cortex deploy [flags]

Flags:
  -f, --force
  -h, --help
```

The `deploy` command sends all deployment configuration and code to Cortex. If all validations pass, Cortex will attempt to create the desired state on the cluster.

## refresh

```
Delete cached resources and deploy.

Usage:
  cortex refresh [flags]

Flags:
  -f, --force
  -h, --help
```

The `refresh` command behaves similarly to the `deploy` command. The key difference is that `refresh` doesn't use any cached resources.

## predict

```
Make predictions.

Usage:
  cortex predict API_NAME SAMPLES_FILE [flags]

Flags:
  -d, --deployment string   deployment name
  -j, --json                print the raw json response
  -h, --help
```

The `predict` command converts samples from a JSON file into prediction requests and outputs the response. This command is useful for quickly testing model output.

## delete

```
Delete a deployment.

Usage:
  cortex delete [DEPLOYMENT_NAME] [flags]

Flags:        
  -c, --keep-cache   keep cached data for the deployment
  -h, --help
```

The `delete` command deletes an deployment's resources from the cluster.

## get

```
Get information about resources.

Usage:
  cortex get [RESOURCE_TYPE] [RESOURCE_NAME] [flags]

Flags:
  -d, --deployment string   deployment name
  -s, --summary             summarized view of resources
  -w, --watch               re-run the command every 2 seconds
  -h, --help
```

The `get` command outputs the current state of all resources on the cluster. Specifying a resource name provides a more detailed view of the configuration and state of that particular resource.

## logs

```
Get logs for a resource.

Usage:
  cortex logs [RESOURCE_TYPE] RESOURCE_NAME [flags]

Flags:
  -d, --deployment string   deployment name
  -v, --verbose
  -h, --help
```

The `logs` command streams logs from the workload corresponding to the specified resource. For example, `cortex logs models dnn` will get the Cortex logs from the most recent training workload for `dnn`. Using the `-v` or `--verbose` flag will show all of the logs for the workload (not just Cortex's logs).

## configure

```
Configure the CLI.

Usage:
  cortex configure [flags]

Flags:
  -h, --help
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
  -h, --help
```
