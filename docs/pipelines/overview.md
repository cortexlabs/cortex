# Overview

Cortex deployments consist of declarative resource configuration written in YAML as well as Python code to implement aggregators, transformers, and models. Each resource has a `kind`.

Resources can reference other resources from within their configuration (e.g. when defining input values) by prefixing the other resource's name with an `@` symbol. For example, a model may specify `input: @column1`, which denotes that a resource named "column1" is an input to this model.

With the exception of the `deployment` kind (which must be defined in a top-level `cortex.yaml` file), resources may be defined in any YAML file within your Cortex folder or any subdirectories.

The `cortex deploy` command will validate all resource configuration and attempt to create the requested state on the cluster.

# Execution pipeline

Cortex processes resources in the following order:

1. Raw Columns
1. Aggregates
1. Transformed Columns
1. Models
1. APIs

If resource configuration changes, Cortex attempts to reuse cached resources whenever possible.

Below is an example execution DAG encompassing all possible resource dependencies:

<img src='https://s3-us-west-2.amazonaws.com/cortex-public/dag.png'>
