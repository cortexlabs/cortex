# Overview

Cortex applications consist of declarative resource configuration written in YAML as well as Python code to implement aggregators, transformers, and models. Each resource has one of the following `kind`s:

* [app](app.md)
* [environment](environments.md)
* [raw_feature](raw-features.md)
* [aggregator](aggregators.md)
* [aggregate](aggregates.md)
* [transformer](transformers.md)
* [transformed_feature](transformed-features.md)
* [model](models.md)
* [api](apis.md)
* [constant](constants.md)

With the exception of the `app` kind (which must be defined in a top-level `app.yaml` file), resources may be defined in any YAML file within your Cortex application folder or any subdirectories.

The `cortex deploy` command will validate all resource configuration and attempt to create the requested state on the cluster.
