# Async

Async APIs are designed for asynchronous workloads in which the user submits an asynchronous request and retrieves the result later (either by polling or through a webhook).

Async APIs are a good fit for users who want to submit longer workloads (such as video, audio or document processing), and do not need the result immediately or synchronously.

## How it works

When you deploy an AsyncAPI, Cortex creates an SQS queue, a pool of Async Gateway workers, and a pool of workers running your containers.

The Async Gateway is responsible for submitting the workloads to the queue and for retrieving workload statuses and results. Cortex fully implements and manages the Async Gateway and the queue.

The pool of workers running your containers autoscales based on the average number of messages in the queue and can scale down to 0 (if configured to do so).

![](https://user-images.githubusercontent.com/7456627/111491999-9b67f100-873c-11eb-87f0-effcf4aab01b.png)
