# Realtime API Overview

_WARNING: you are on the master branch, please refer to the docs on the branch that matches your `cortex version`_

You can deploy a Realtime API on Cortex to serve model predictions via an HTTP endpoint. Traffic to the endpoints will be loadbalanced across autoscaling servers that scale based on incoming request traffic. You can implement a custom Predictor class in Python to perform preprocessing on incoming requests and postprocessing on requests to customize your API interface. Updates to the model or your predictor will be executed using a rolling update strategy to avoid downtime for your API. Request metrics and logs will automatically be aggregated and be accessible via the Cortex CLI (`cortex logs <api_name>`) or on your AWS console.

You may want to deploy your model as a Realtime API if any of the following scenarios apply to your use case:

* serve predictions on demand using a webserver
* each request requires a prediction to be made on a single (or a small number of samples)
* serve predictions directly to consumers
* each request takes from a few milliseconds up to minutes to complete

You may want to consider deploying your model as a [Batch API](#batchapi.md) if these scenarios don't seem to apply to you.

## Next steps

<!-- CORTEX_VERSION_MINOR -->
* Try the [tutorial](../../examples/sklearn/iris-classifier/README.md) to deploy a Realtime API locally in just a few minutes and on AWS.
* See our [exporting docs](../deployments/exporting.md) for how to export your model to use in a Realtime API.
* See the [Predictor docs](syncapi/predictors.md) to begin implementing your own Predictor class.
* See the [API configuration docs](syncapi/api-configuration.md) for a full list of features that can be used to deploy your custom Realtime API.
Next steps
