# Batch API Overview

_WARNING: you are on the master branch, please refer to the docs on the branch that matches your `cortex version`_

You can deploy your model as a Batch API to setup a web service that can receive job requests and orchestrate offline batch inference on large datasets across multiple workers. You provide a Predictor implementation in Python that defines how to apply your model to a batch of input data and Cortex will provide:

- web service to receive job requests, manage running jobs and get job statuses
- partitioner that breaks down large datasets into batches
- pool of workers initialized with your Predictor implementation that scale to 0
- execute an on_job_complete hook to enable you to chain jobs together and perform post job completion tasks
- log streaming

## Batch API workflow

When a job request is submitted to your Batch API endpoint you will immediately receive a Job ID. In the background, the Batch API will break down the dataset provided in the job request into batches. Each batch will be pushed onto a queue. After the entire dataset has been enqueued, the number of workers specified in your job request will be instantiated. This pool of workers will take one batch from the queue at a time and execute your Predictor implementation. Your predictor implementation will typically include preprocessing on the batch, running batch inference and writing the outcome of the inference to a desired storage. After all of the batches have been processed, the on job complete hook provided in your predictor implementation will be executed by one of the workers.

![batch api architecture diagram](https://user-images.githubusercontent.com/4365343/87894668-6373a700-ca11-11ea-901e-350809f72821.png)

At any point, you can use the Job ID, provided upon job submission, to make requests to the Batch API endpoint to get job status, progress metrics and worker statuses. Logs for each step of the job are aggregated and are accessible via the Cortex CLI (`cortex logs <api_name> <job_id>`) or in your CloudWatch console.

## When should I use Batch API

You may want to deploy your model as a Batch API if any of the following scenarios apply to your use case:

- run inference on a large dataset distributed across multiple workers
- monitor the progress of a job and keep track of the overall job status
- inference is a part of internal data pipelines that can be chained together
- receive a small number of requests but each request takes minutes to hours to complete

You may want to consider deploying your model as a [Sync API](#syncapi.md) if these scenarios don't seem to apply to you.

## Next steps

<!-- CORTEX_VERSION_MINOR -->
* Try the [tutorial](../../examples/sklearn/iris-classifier/README.md) to deploy a Batch API on your Cortex cluster.
* See our [exporting docs](../deployments/exporting.md) for how to export your model to use in a Batch API.
* See the [Predictor docs](batchapi/predictors.md) to begin implementing your own Predictor class.
* See the [API configuration docs](batchapi/api-configuration.md) for a full list of features that can be used to deploy your custom Batch API.


