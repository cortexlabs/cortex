# Overview

Cortex applications consist of declarative resource configuration written in YAML as well as Python code to implement aggregators, transformers, and models. Each resource has a `kind`:

* [app](app.md)
* [environment](environments.md)
* [raw_column](raw-columns.md)
* [aggregator](aggregators.md)
* [aggregate](aggregates.md)
* [transformer](transformers.md)
* [transformed_column](transformed-columns.md)
* [model](models.md)
* [api](apis.md)
* [constant](constants.md)

With the exception of the `app` kind (which must be defined in a top-level `app.yaml` file), resources may be defined in any YAML file within your Cortex application folder or any subdirectories.

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
