# Batch

Batch APIs run distributed and fault-tolerant batch processing jobs on demand.

Batch APIs are a good fit for users who want to break up their workloads and distribute them across a dedicated pool of workers (for example, running inference on a set of images).

## How it works

When you deploy a Batch API, Cortex creates an endpoint to receive job submissions.

Upon job submission, Cortex responds with a Job ID, and asynchronously triggers a Batch Job.

First, Cortex deploys an enqueuer, which breaks up the data in the job into batches and pushes them onto an SQS FIFO queue.

After enqueuing is complete, Cortex initializes the requested number of worker pods and attaches a dequeuer sidecar to each pod. The dequeuer is responsible for retrieving batches from the queue and making an http request to your pod for each batch.

After the worker pods have emptied the queue, the job is marked as complete, and Cortex will terminate the worker pods and delete the SQS queue.

You can make GET requests to the BatchAPI endpoint to get the status of the Job and metrics such as the number of batches completed and failed.

![]()
