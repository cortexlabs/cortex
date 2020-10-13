# Batch API Overview

_WARNING: you are on the master branch, please refer to the docs on the branch that matches your `cortex version`_

You can deploy your model as a Batch API to create a web service that can receive job requests and orchestrate offline batch inference on large datasets across multiple workers.

## When should I use a Batch API

You may want to deploy your model as a Batch API if any of the following scenarios apply to your use case:

* inference will run on a large dataset and can be distributed across multiple workers
* job progress and status needs to be monitored
* inference is a part of internal data pipelines that may be chained together
* a small number of requests are received, but each request takes minutes or hours to complete

You may want to consider deploying your model as a [Realtime API](realtime-api.md) if these scenarios don't apply to you.

A Batch API deployed in Cortex will create/support the following:

* a REST web service to receive job requests, manage running jobs, and retrieve job statuses
* an autoscaling worker pool that can scale to 0
* log aggregation and streaming
* `on_job_complete` hook to for aggregation or triggering webhooks

## How does it work

You specify the following:

* a Cortex Predictor class in Python that defines how to initialize your model run batch inference
* an API configuration YAML file that defines how your API will behave in production (parallelism, networking, compute, etc.)

Once you've implemented your predictor and defined your API configuration, you can use the Cortex CLI to deploy a Batch API. The Cortex CLI will package your predictor implementation and the rest of the code and dependencies and upload it to the Cortex Cluster. The Cortex Cluster will setup an endpoint to a web service that can receive job submission requests and manage jobs.

A job submission typically consists of an input dataset or the location of your input dataset, the number of workers for your job, and the batch size. When a job is submitted to your Batch API endpoint, you will immediately receive a Job ID that you can use to get the job's status and logs, and stop the job if necessary. Behind the scenes, your Batch API will break down the dataset into batches and push them onto a queue. Once all of the batches have been enqueued, the Cortex Cluster will spin up the requested number of workers and initialize them with your predictor implementation. Each worker will take one batch at a time from the queue and run your Predictor implementation. After all batches have been processed, the `on_job_complete` hook in your predictor implementation (if provided) will be executed by one of the workers.

At any point, you can use the Job ID that was provided upon job submission to make requests to the Batch API endpoint to get job status, progress metrics, and worker statuses. Logs for each job are aggregated and are accessible via the Cortex CLI or in your AWS console.

## Next steps

* Try the [tutorial](../../examples/batch/image-classifier/README.md) to deploy a Batch API on your Cortex cluster.
* See our [exporting guide](../guides/exporting.md) for how to export your model to use in a Batch API.
* See the [Predictor docs](batch-api/predictors.md) for how to implement a Predictor class.
* See the [API configuration docs](batch-api/api-configuration.md) for a full list of features that can be used to deploy your Batch API.
