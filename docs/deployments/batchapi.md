# Batch API Overview

_WARNING: you are on the master branch, please refer to the docs on the branch that matches your `cortex version`_

You can deploy your model as a Batch API to setup a web service that can receive job requests and orchestrate offline batch inference on large datasets across multiple workers. A Batch API deployed in Cortex will create/support the following:

- REST web service to receive job requests, manage running jobs and get job statuses
- autoscaling worker pool that can scale to 0
- log aggregation and streaming
- on_job_complete hook to chain jobs together

To deploy a Batch API on Cortex, you need to specify a custom Predictor class that defines how to initialize your model and apply your model to a batch of data. You can use the Cortex CLI to deploy your Batch API. Your predictor implementation, along with the rest of your code and dependencies will automatically be containerized. A Batch API web service will be created for you that can receive job requests and manage jobs. When a job request is submitted to your Batch API endpoint you will receive a Job ID. In the background, the Batch API will break down the dataset provided in the job request into batches and push them onto a queue. A pool workers will take batches from the queue and execute your Predictor implementation. After all of the batches have been processed, the on job complete hook provided in your predictor implementation will be executed by one of the workers.

At any point, you can use the Job ID, that was provided upon job submission, to make requests to the Batch API endpoint to get job status, progress metrics and worker statuses. Logs for each step of the job are aggregated and are accessible via the Cortex CLI (`cortex logs <batch_api_name> <job_id>`) or in your CloudWatch console.

## When should I use Batch API

You may want to deploy your model as a Batch API if any of the following scenarios apply to your use case:

- run inference on a large dataset distributed across multiple workers
- monitor the progress of a job and keep track of the overall job status
- inference is a part of internal data pipelines that can be chained together
- receive a small number of requests but each request takes minutes to hours to complete

You may want to consider deploying your model as a [Sync API](#syncapi.md) if these scenarios don't seem to apply to you.

## Next steps

<!-- CORTEX_VERSION_MINOR -->
* Try the [tutorial](../../examples/batch/image-classifier/README.md) to deploy a Batch API on your Cortex cluster.
* See our [exporting docs](deployments/exporting.md) for how to export your model to use in a Batch API.
* See the [Predictor docs](batchapi/predictors.md) to begin implementing your own Predictor class.
* See the [API configuration docs](batchapi/api-configuration.md) for a full list of features that can be used to deploy your custom Batch API.
