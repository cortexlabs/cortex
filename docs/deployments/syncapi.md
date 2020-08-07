# Sync API Overview

_WARNING: you are on the master branch, please refer to the docs on the branch that matches your `cortex version`_

You can deploy a Sync API on Cortex to serve model predictions via an HTTP endpoint. A Sync API deployed in Cortex has the following features:

- request based autoscaling
- rolling updates to enable you to update the model/Predictor code without downtime
- realtime metrics collection
- log streaming
- multi-model serving
- traffic splitting/ A/B testing
- post_predict hooks that can run after a request has been responded to

To deploy a Sync API on Cortex, you need to specify a custom Predictor class that defines how to initialize your model and apply your model to incoming requests. You can use the Cortex CLI to deploy your Sync API. Your predictor implementation along with the rest of your code and dependencies will automatically be containerized as a web server. An endpoint will be created to allow access to your Sync API web server. Cortex will automatically spin up more copies of the webserver based on incoming traffic to your endpoint. You can safely update your model or your Predictor class without experiencing downtime because your updates will be rolled automatically. Request metrics and logs will automatically be aggregated and be accessible via the Cortex CLI (`cortex logs <api_name>`) or on your AWS console.

## When should I use Sync API

You may want to deploy your model as a Sync API if any of the following scenarios apply to your use case:

* serve predictions on demand using a webserver
* each request requires a prediction to be made on a single (or a small number of samples)
* serve predictions directly to consumers
* each request takes from a few milliseconds up to minutes to complete

You may want to consider deploying your model as a [Batch API](#batchapi.md) if these scenarios don't seem to apply to you.

## Next steps

<!-- CORTEX_VERSION_MINOR -->
* Try the [tutorial](../../examples/sklearn/iris-classifier/README.md) to deploy a Sync API locally in just a few minutes and on AWS.
* See our [exporting docs](../deployments/exporting.md) for how to export your model to use in a Sync API.
* See the [Predictor docs](syncapi/predictors.md) to begin implementing your own Predictor class.
* See the [API configuration docs](syncapi/api-configuration.md) for a full list of features that can be used to deploy your custom Sync API.
