# AsyncAPI

The AsyncAPI kind is designed for asynchronous workloads, in which the user submits a request to start the processing
and retrieves the result later, either by polling or through a webhook.

The design is summarized in the image below.

![AsyncAPI Design](https://user-images.githubusercontent.com/7456627/111491999-9b67f100-873c-11eb-87f0-effcf4aab01b.png)

The Async Gateway is responsible for submitting the workloads to the queue and for the retrieval of the respective
workload status and results. Cortex fully manages the Async Gateway and the queue, while the user is in charge of implementing
the API predictor. Autoscaling is provided based on the average number of messages in the queue.

## Use-cases

AsyncAPI matches the use-cases of the users that want to submit longer machine learning workloads, such as video, audio
or document processing, and do not need the result immediately.

{% hint style="info" %}
AsyncAPI is still in a beta state.
{% endhint %}
