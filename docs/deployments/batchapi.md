# Batch API Overview

_WARNING: you are on the master branch, please refer to the docs on the branch that matches your `cortex version`_

You can deploy your model as Batch API to setup an HTTP endpoint that can receive jobs and run distributed inference across multiple worker. When a job request is submitted to your Batch API endpoint, the dataset is broken down into batches and pushed onto a queue. After enqueuing all of the batches, a group of workers is initialized. The workers takes one item from the queue at a time and runs your custom Predictor implementation. Your predictor implementation can be used perform preprocessing and write the outcome of the inference to the desired storage. Once the queue is empty, the group of workers are shutdown. At any point you can use the Job ID which is provided upon job submission and make requests to the Batch API endpoint to get job status, progress metrics and worker statuses. Logs for each step of the job and across multiple workers are aggregated and are accessible via the Cortex CLI (`cortex logs <api_name>`) or in your CloudWatch console.

![batch api architecture diagram](https://user-images.githubusercontent.com/4365343/87894668-6373a700-ca11-11ea-901e-350809f72821.png)

You may want to deploy your model as a Batch API if any of the following scenarios apply to your use case:

* inferences need to be run on a schedule
* each individual request requires offline inference on large datasets
* inference is a part of internal data pipelines
* each request takes minutes to hours to complete

You may want to consider deploying your model as a [Realtime API](#syncapi.md) if these scenarios don't seem to apply to you.

## Next steps

<!-- CORTEX_VERSION_MINOR -->
* Try the [tutorial](../../examples/sklearn/iris-classifier/README.md) to deploy a Batch API on your Cortex cluster.
* See our [exporting docs](../deployments/exporting.md) for how to export your model to use in aa Batch API.
* See the [Predictor docs](batchapi/predictors.md) to begin implementing your own Predictor class.
* See the [API configuration docs](batchapi/api-configuration.md) for a full list of features that can be used to deploy your custom Batch API.
Next steps
