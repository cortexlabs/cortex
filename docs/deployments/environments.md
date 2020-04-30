# Environments

_WARNING: you are on the master branch, please refer to the docs on the branch that matches your `cortex version`_

The `cortex` CLI can be used to deploy models locally and/or to any number of clusters. CLI environments are used to select which cluster to use for a `cortex` command.

By default, the CLI ships with a single environment: `local`. This is the default environment for all Cortex commands (other than `cortex cluster` commands).

`cortex cluster` commands use an environment named `aws` by default. For example `cortex cluster up` will configure the `aws` environment to connect to your new cluster, and you may then reference your cluster using the `aws` environment.

Environments can always be specified via the `-e`/`--env` flag.

You can list your environments with `cortex env list`, change the default environment with `cortex env default`, delete an environment with `cortex env delete`, and create/configure an environment with `cortex env configure`. _Note: `cortex cluster up [--config=cluster.yaml]` and `cortex cluster info [--config=cluster.yaml]` configure the specified environment (or the `aws` environment if omitted) to connect to the cluster._

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

cortex cluster up     # configures the aws env; same as `cortex cluster up --env=aws`
cortex deploy --env=aws
cortex logs my-api --env=aws
cortex delete my-api --env=aws
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
