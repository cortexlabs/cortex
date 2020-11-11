# Environments

The `cortex` CLI can be used to deploy models locally and/or to any number of clusters. Environments are used to select which cluster to use for a `cortex` command. An environment contains the information required to connect to a cluster \(e.g. AWS credentials and Cortex operator URL\).

## Example: `aws` only

```bash
cortex cluster up       # configures the aws env; same as `cortex cluster up --configure-env aws`
cortex env default aws  # sets aws as the default env
cortex deploy           # uses aws env; same as `cortex deploy --env aws`
cortex logs my-api      # uses aws env; same as `cortex logs my-api --env aws`
cortex delete my-api    # uses aws env; same as `cortex delete my-api --env aws`
```

## Example: `local` only

```bash
cortex deploy         # uses local env; same as `cortex deploy --env local`
cortex logs my-api    # uses local env; same as `cortex logs my-api --env local`
cortex delete my-api  # uses local env; same as `cortex delete my-api --env local`
```

## Example: `local` and `aws`

```bash
cortex deploy         # uses local env; same as `cortex deploy --env local`
cortex logs my-api    # uses local env; same as `cortex logs my-api --env local`
cortex delete my-api  # uses local env; same as `cortex delete my-api --env local`

cortex cluster up       # configures the aws env; same as `cortex cluster up --configure-env aws`
cortex deploy --env aws
cortex deploy           # uses local env; same as `cortex deploy --env local`

# optional: change the default environment to aws
cortex env default aws    # sets aws as the default env
cortex deploy             # uses aws env; same as `cortex deploy --env aws`
cortex deploy --env local
```

## Example: multiple clusters

```bash
cortex cluster up --config cluster1.yaml --configure-env cluster1  # configures the cluster1 env
cortex cluster up --config cluster2.yaml --configure-env cluster2  # configures the cluster2 env

cortex deploy --env cluster1
cortex logs my-api --env cluster1
cortex delete my-api --env cluster1

cortex deploy --env cluster2
cortex logs my-api --env cluster2
cortex delete my-api --env cluster2
```

## Example: multiple clusters, if you omitted the `--configure-env` on `cortex cluster up`

```bash
cortex cluster info --config cluster1.yaml --configure-env cluster1  # configures the cluster1 env
cortex cluster info --config cluster2.yaml --configure-env cluster2  # configures the cluster2 env

cortex deploy --env cluster1
cortex logs my-api --env cluster1
cortex delete my-api --env cluster1

cortex deploy --env cluster2
cortex logs my-api --env cluster2
cortex delete my-api --env cluster2
```

## Example: configure `cortex` CLI to connect to an existing cluster

If you are installing the `cortex` CLI on a new computer, you can configure it to access an existing Cortex cluster.

On the computer which already has the CLI configured, run:

```bash
cortex env list
```

Take note of the environment name and operator endpoint of the desired environment.

On your new machine, run:

```bash
cortex env configure
```

This will prompt for the necessary configuration. Note that the AWS credentials that you use here do not need any IAM permissions attached. If you will be running any `cortex cluster` commands specify the preferred AWS credentials using cli flags `--aws-key AWS_ACCESS_KEY_ID --aws-secret AWS_SECRET_ACCESS_KEY`. See [IAM permissions](security.md#iam-permissions) for more details.

## Environments overview

By default, the CLI ships with a single environment named `local`. This is the default environment for all Cortex commands \(other than `cortex cluster` commands\), which means that APIs will be deployed locally by default.

When you create a cluster with `cortex cluster up`, an environment named `aws` is automatically created to point to your new cluster. You can name the environment something else via the `--configure-env` flag, e.g. `cortex cluster up --configure-env prod`. You can also use the `--configure-env` flag with `cortex cluster info` and `cortex cluster configure` to create/update the specified environment. You may interact with your cluster by appending `--env aws` to your `cortex` commands, e.g. `cortex deploy --env aws`.

You can list your environments with `cortex env list`, change the default environment with `cortex env default`, delete an environment with `cortex env delete`, and create/update an environment with `cortex env configure`.

