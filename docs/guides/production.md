# Running in production

**Tips for batch and realtime APIs:**

* Consider using [spot instances](../aws/spot.md) to reduce cost.

* If you're using multiple clusters and/or multiple developers are interacting with your cluster(s), see our documention on [environments](../miscellaneous/environments.md)

**Additional tips for realtime APIs**

* Consider tuning `processes_per_replica` and `threads_per_process` in your [Realtime API configuration](../workloads/realtime/configuration.md). Each model behaves differently, so the best way to find a good value is to run a load test on a single replica (you can set `min_replicas` to 1 to avoid autocaling). Here is [additional information](../workloads/realtime/parallelism.md#concurrency) about these fields.

* You may wish to customize the autoscaler for your APIs. The [autoscaling documentation](../workloads/realtime/autoscaling.md) describes each of the parameters that can be configured.

* When creating an API that you will send large amounts of traffic to all at once, set `min_replicas` at (or slightly above) the number of replicas you expect will be necessary to handle the load at steady state. After traffic has been fully shifted to your API, `min_replicas` can be reduced to allow automatic downscaling.

* [Traffic splitters](./workloads/realtime/traffic-splitter.md) can be used to route a subset of traffic to an updated API. For example, you can create a traffic splitter named `my-api`, and route requests to `my-api` to any number of Realtime APIs (e.g. `my-api_v1`, `my-api_v2`, etc). The percentage of traffic that the traffic splitter routes to each API can be updated on the fly.

* If initialization of your API replicas takes a while (e.g. due to downloading large models from slow hosts or installing dependencies), and responsive autoscaling is important to you, consider pre-building your API's Docker image. See [here](../workloads/system-packages.md#custom-docker-image) for instructions.

* If your API is receiving many queries per second and you are using the TensorFlow Predictor, consider enabling [server-side batching](../workloads/realtime/parallelism.md#server-side-batching).

* [Overprovisioning](../workloads/realtime/autoscaling.md#overprovisioning) can be used to reduce the chance of large queues building up. This can be especially important when inferences take a long time.

**Additional tips for inferences that take a long time:**

* Consider using [GPUs](../aws/gpu.md) or [Inferentia](../aws/inferentia.md) to speed up inference.

* Consider setting a low value for `max_replica_concurrency`, since if there are many requests in the queue, it will take a long time until newly received requests are processed. See [autoscaling docs](../workloads/realtime/autoscaling.md) for more details.

* Keep in mind that API Gateway has a 29 second timeout; if your requests take longer (due to a long inference time and/or long request queues), you will need to disable API Gateway for your API by setting `api_gateway: none` in the `networking` config in your [Realtime API configuration](../workloads/realtime/configuration.md) and/or [Batch API configuration](../workloads/batch/configuration.md). Alternatively, you can disable API gateway for all APIs in your cluster by setting `api_gateway: none` in your [cluster configuration file](../aws/install.md) before creating your cluster.
