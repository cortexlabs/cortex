# Batch API Overview

_WARNING: you are on the master branch, please refer to the docs on the branch that matches your `cortex version`_

You can deploy your model as a Batch API to setup a web service that can receive job requests and orchestrate offline batch inference on large datasets across multiple workers. A Batch API deployed in Cortex will create/support the following:

- REST web service to receive job requests, manage running jobs and get job statuses
- autoscaling worker pool that can scale to 0
- log aggregation and streaming
- `on_job_complete` hook to chain jobs together

## How does it work

You specify the following:

- Cortex Predictor class in Python that defines how to initialize your model run batch inference
- API configuration yaml that defines how your API will behave in production e.g. compute, networking

Once you've implemented your predictor and defined API yaml, you can use the Cortex CLI to deploy a Batch API. The Cortex CLI will package your predictor implementation and the rest of the code and dependencies and upload it to the Cortex Cluster. The Cortex Cluster will setup an endpoint to a web service that can receive job requests and manage jobs.

You can begin submitting jobs to your Batch API endpoint. A job submission typically consists of an input dataset or where the location of your input dataset, the number of workers you want to the job and the batch size. When a job is submitted to your Batch API endpoint, you will immediately receive a Job ID that you can use to get the job's status, logs and stop the job if necessary. In the background, your Batch API will break down the dataset into batches and push them onto a queue. Once all of the batches have been enqueued, the Cortex Cluster will spin up the requested number of workers initialized with your predictor implementation. Each worker will take a batch from the queue and run your Predictor implementation. After all of the batches have been processed, the `on_job_complete` hook (if provided) in your predictor implementation will be executed by one of the workers.

At any point, you can use the Job ID, that was provided upon job submission, to make requests to the Batch API endpoint to get job status, progress metrics and worker statuses. Logs for each step of the job are aggregated and are accessible via the Cortex CLI or in your AWS console.

## When should I use Batch API

You may want to deploy your model as a Batch API if any of the following scenarios apply to your use case:

- run inference on a large dataset that can be distributed across multiple workers
- monitor the progress of a job and keep track of the overall job status
- inference is a part of internal data pipelines that can be chained together
- receive a small number of requests but each request takes minutes to hours to complete

You may want to consider deploying your model as a [Sync API](#syncapi.md) if these scenarios don't seem to apply to you.

## Next steps

<!-- CORTEX_VERSION_MINOR -->
* Try the [tutorial](../../examples/batch/image-classifier/README.md) to deploy a Batch API on your Cortex cluster.
* See our [exporting docs](exporting.md) for how to export your model to use in a Batch API.
* See the [Predictor docs](batchapi/predictors.md) to begin implementing your own Predictor class.
* See the [API configuration docs](batchapi/api-configuration.md) for a full list of features that can be used to deploy your custom Batch API.
