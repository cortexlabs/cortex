# Async APIs

The AsyncAPI kind is designed for asynchronous workloads, in which the user submits a request to start the processing
and retrieves the result later, either by polling or through a webhook.

The design is summarized in the image below.

![](https://user-images.githubusercontent.com/7456627/111491999-9b67f100-873c-11eb-87f0-effcf4aab01b.png)

The Async Gateway is responsible for submitting the workloads to the queue and for the retrieval of the respective
workload status and results. Cortex fully manages the Async Gateway and the queue. Autoscaling is provided based on the average number of messages in the queue.

## Use-cases

AsyncAPI is a good fit for users who want to submit longer workloads (such as video, audio
or document processing), and do not need the result immediately or synchronously. It is also useful for lowering the costs as the API can be scaled to zero when there are no pending requests. 

