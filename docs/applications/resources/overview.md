# Overview

Cortex applications consist of declarative resource configuration written in YAML as well as Python code to implement models, transformations, and aggregations. Each resource has one of the following `kind`s:

* [environment](environments.md)
* [raw_feature](raw-features.md)
* [aggregator](aggregators.md)
* [aggregate](aggregates.md)
* [transformer](transformers.md)
* [transformed_feature](transformed-features.md)
* [model](models.md)
* [api](apis.md)
* [constant](constants.md)

Once defined, the `cortex deploy` command will validate all resource configuration and attempt to create the requested state on the cluster.
