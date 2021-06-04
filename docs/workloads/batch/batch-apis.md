# Batch APIs

BatchAPIs run distributed and fault-tolerant batch processing jobs on demand.

BatchAPI is a good fit for users who want to break up their workload and distribute them across a dedicated pool of workers. Running inference on a folder of images is a common use case for the BatchAPI.

## How it works

When you deploy a BatchAPI, an endpoint to receive job submissions is created.

Upon job submission, you will receive a Job ID and asynchronously trigger a Batch Job.

Cortex will deploy an enqueuer which will break up the data in the job into batches and pushes them onto an SQS FIFO queue.

After enqueuing is complete, Cortex initializes the requested number of worker pods and attaches a dequeuer sidecar to each pod. The dequeuer is responsible for retrieving batches from the queue and making an http request to your pod once your pod is ready.

After the worker pods have emptied the queue, job is is marked as complete and Cortex will terminate the worker pods and delete the SQS queue.

You can make GET requests to the BatchAPI endpoint to get the status of the Job and metrics such as the number of batches completed and failed.

![]()