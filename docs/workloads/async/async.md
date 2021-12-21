# Async

Async APIs are designed for asynchronous workloads in which the user submits an asynchronous request and retrieves the result later (either by polling or through a webhook).

Async APIs are a good fit for users who want to submit longer workloads (such as video, audio or document processing), and do not need the result immediately or synchronously.

**Key features**

* asynchronously process requests
* retrieve status and response via HTTP endpoint
* autoscale based on queue length
* avoid cold starts
* scale to zero
* perform rolling updates
* automatically recover from failures and spot instance termination

## How it works

When you deploy an AsyncAPI, Cortex creates an SQS queue, a pool of Async Gateway workers, and a pool of worker pods. Each worker pod is running a dequeuer sidecar and your containers.

Upon receiving a request, the Async Gateway will save the request payload to S3, enqueue the request ID onto an SQS FIFO queue, and respond with the request ID.

The dequeuer sidecar in the worker pod will pull the request from the SQS queue, download the request's payload from S3, and make a POST request to your containers. After the dequeuer receives a response, the corresponding request payload will be deleted from S3 and the response will be saved in S3 for 7 days.

You can fetch the result by making a GET request to the AsyncAPI endpoint with the request ID. The Async Gateway will respond with the status and the result (if the request has been completed).

The pool of workers running your containers autoscales based on the average number of messages in the queue and can scale down to 0 (if configured to do so).

![](https://user-images.githubusercontent.com/808475/146854251-fed4235f-3627-4cd0-bc86-066272d7f138.png)
