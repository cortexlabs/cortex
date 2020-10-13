# Realtime API Overview

_WARNING: you are on the master branch, please refer to the docs on the branch that matches your `cortex version`_

You can deploy a Realtime API on Cortex to serve your model via an HTTP endpoint for on-demand inferences.

## When should I use a Realtime API

You may want to deploy your model as a Realtime API if any of the following scenarios apply to your use case:

* predictions are served on demand
* predictions need to be made in the time of a single web request
* predictions need to be made on an individual basis
* predictions are served directly to consumers

You may want to consider deploying your model as a [Batch API](batch-api.md) if these scenarios don't apply to you.

A Realtime API deployed in Cortex has the following features:

* request-based autoscaling
* rolling updates to enable you to update the model/serving code without downtime
* realtime metrics collection
* log streaming
* multi-model serving
* server-side batching
* traffic splitting (e.g. for A/B testing)

## How does it work

You specify the following:

* a Cortex Predictor class in Python that defines how to initialize and serve your model
* an API configuration YAML file that defines how your API will behave in production (autoscaling, monitoring, networking, compute, etc.)

Once you've implemented your predictor and defined your API configuration, you can use the Cortex CLI to deploy a Realtime API. The Cortex CLI will package your predictor implementation and the rest of the code and dependencies and upload it to the Cortex Cluster. The Cortex Cluster will set up an HTTP endpoint that routes traffic to multiple replicas/copies of web servers initialized with your code.

When a request is made to the HTTP endpoint, it gets routed to one your API's replicas (at random). The replica receives the request, parses the payload and executes the inference code you've defined in your predictor implementation and sends a response.

The Cortex Cluster will automatically scale based on the incoming traffic and the autoscaling configuration you've defined. You can safely update your model or your code and use the Cortex CLI to deploy without experiencing downtime because updates to your API will be rolled out automatically. Request metrics and logs will automatically be aggregated and be accessible via the Cortex CLI or on your AWS console.

## Next steps

* Try the [tutorial](../../examples/pytorch/text-generator/README.md) to deploy a Realtime API locally or on AWS.
* See our [exporting guide](../guides/exporting.md) for how to export your model to use in a Realtime API.
* See the [Predictor docs](realtime-api/predictors.md) for how to implement a Predictor class.
* See the [API configuration docs](realtime-api/api-configuration.md) for a full list of features that can be used to deploy your Realtime API.
