# Environments

_WARNING: you are on the master branch, please refer to the docs on the branch that matches your `cortex version`_

The `cortex` CLI can be used to deploy models locally and/or to any number of clusters. Environments are used to select which cluster to use for a `cortex` command. An environment contains the information required to connect to a cluster (e.g. AWS credentials and Cortex operator URL).

By default, the CLI ships with a single environment named `local`. This is the default environment for all Cortex commands (other than `cortex cluster` commands), which means that APIs will be deployed locally by default.

Some cortex commands (i.e. `cortex cluster` commands) only apply to cluster environments. Unless otherwise specified by the `-e`/`--env` flag, `cortex cluster` commands create/use an environment named `aws`. For example `cortex cluster up` will configure the `aws` environment to connect to your new cluster. You may interact with this cluster by appending `--env=aws` to your `cortex` commands.

If you accidentally delete or overwrite one of your cluster environments, running `cortex cluster info --env=ENV_NAME` will automatically update the specified environment to interact with the cluster.

You can list your environments with `cortex env list`, change the default environment with `cortex env default`, delete an environment with `cortex env delete`, and create/update an environment with `cortex env configure`.

## Example: `local` only

```bash
cortex deploy         # uses local env; same as `cortex deploy --env=local`
cortex logs my-api    # uses local env; same as `cortex logs my-api --env=local`
cortex delete my-api  # uses local env; same as `cortex delete my-api --env=local`
```

## Example: `local` and `aws`

```bash
cortex deploy         # uses local env; same as `cortex deploy --env=local`
cortex logs my-api    # uses local env; same as `cortex logs my-api --env=local`
cortex delete my-api  # uses local env; same as `cortex delete my-api --env=local`

cortex cluster up       # configures the aws env; same as `cortex cluster up --env=aws`
cortex deploy --env=aws
cortex deploy           # uses local env; same as `cortex deploy --env=local`

# optional: change the default environment to aws
cortex env default aws    # sets aws as the default env
cortex deploy             # uses aws env; same as `cortex deploy --env=aws`
cortex deploy --env=local
```

## Example: only `aws`

```bash
cortex cluster up       # configures the aws env; same as `cortex cluster up --env=aws`
cortex env default aws  # sets aws as the default env
cortex deploy           # uses aws env; same as `cortex deploy --env=aws`
cortex logs my-api      # uses aws env; same as `cortex logs my-api --env=aws`
cortex delete my-api    # uses aws env; same as `cortex delete my-api --env=aws`
```

## Example: multiple clusters

```bash
cortex cluster up --config=cluster1.yaml --env=cluster1  # configures the cluster1 env
cortex cluster up --config=cluster2.yaml --env=cluster2  # configures the cluster2 env

cortex deploy --env=cluster1
cortex logs my-api --env=cluster1
cortex delete my-api --env=cluster1

cortex deploy --env=cluster2
cortex logs my-api --env=cluster2
cortex delete my-api --env=cluster2
```

## Example: multiple clusters, if you omitted the `--env` on `cortex cluster up`

```bash
cortex cluster info --config=cluster1.yaml --env=cluster1  # configures the cluster1 env
cortex cluster info --config=cluster2.yaml --env=cluster2  # configures the cluster2 env

cortex deploy --env=cluster1
cortex logs my-api --env=cluster1
cortex delete my-api --env=cluster1

cortex deploy --env=cluster2
cortex logs my-api --env=cluster2
cortex delete my-api --env=cluster2
```
