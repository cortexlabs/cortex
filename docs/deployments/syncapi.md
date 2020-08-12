# Sync API Overview

_WARNING: you are on the master branch, please refer to the docs on the branch that matches your `cortex version`_

You can deploy a Sync API on Cortex to serve your model via an HTTP endpoint. A Sync API deployed in Cortex has the following features:

- request-based autoscaling
- rolling updates to enable you to update the model/serving code without downtime
- realtime metrics collection
- log streaming
- multi-model serving

## How does it work

You specify the following:

- Cortex Predictor class in Python that defines how to initialize and serve your model
- API configuration yaml that defines how your API will behave in production (compute, autoscaling, networking, monitoring, etc.)

Once you've implemented your predictor and defined API yaml, you can use the Cortex CLI to deploy a Sync API. The Cortex CLI will package your predictor implementation and the rest of the code and dependencies and upload it to the Cortex Cluster. The Cortex Cluster will set up an HTTP endpoint that routes traffic to multiple replicas/copies of web servers initialized with your code.

When a request is made to the HTTP endpoint, it gets randomly routed to one your API's replicas. The replica receives the request, parses the payload and executes the serving code you've defined in the predictor implementation and sends a response.

The Cortex Cluster will automatically scale based on the incoming traffic and the autoscaling configuration you've defined. You can safely update your model or your code and use the Cortex CLI to deploy without experiencing any downtime because updates to your cluster will be rolled automatically. Request metrics and logs will automatically be aggregated and be accessible via the Cortex CLI or on your AWS console.

## When should I use Sync API

You may want to deploy your model as a Sync API if any of the following scenarios apply to your use case:

* serve predictions on demand
* predictions need to be made in the time of a single web request
* predictions need to be served on an individual basis
* serve predictions directly to consumers

You may want to consider deploying your model as a [Batch API](#batchapi.md) if these scenarios don't seem to apply to you.

## Next steps

<!-- CORTEX_VERSION_MINOR -->
* Try the [tutorial](../../examples/sklearn/iris-classifier/README.md) to deploy a Sync API locally in just a few minutes and on AWS.
* See our [exporting docs](exporting.md) for how to export your model to use in a Sync API.
* See the [Predictor docs](syncapi/predictors.md) to begin implementing your own Predictor class.
* See the [API configuration docs](syncapi/api-configuration.md) for a full list of features that can be used to deploy your custom Sync API.
