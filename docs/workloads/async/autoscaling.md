# Autoscaling

Cortex auto-scales AsyncAPIs on a per-API basis based on your configuration.

## Autoscaling replicas

### Relevant pod configuration

In addition to the autoscaling configuration options (described below), there is one field in the pod configuration which are relevant to replica autoscaling:

**`max_concurrency`** (default: 1): The maximum number of requests that will be concurrently sent into the container by Cortex. If your web server is designed to handle multiple concurrent requests, increasing `max_concurrency` will increase the throughput of a replica (and result in fewer total replicas for a given load).

<br>

### Autoscaling configuration

**`min_replicas`**: The lower bound on how many replicas can be running for an API.

<br>

**`max_replicas`**: The upper bound on how many replicas can be running for an API.

<br>

**`target_in_flight`** (default: `max_concurrency` in the pod configuration): This is the desired number of in-flight requests per replica, and is the metric which the autoscaler uses to make scaling decisions. The number of in-flight requests is simply how many requests have been submitted and are not yet finished being processed. Therefore, this number includes requests which are actively being processed as well as requests which are waiting in the queue.

The autoscaler uses this formula to determine the number of desired replicas:

`desired replicas = total in-flight requests / target_in_flight`

For example, setting `target_in_flight` to `max_concurrency` (the default) causes the cluster to adjust the number of replicas so that on average, there are no requests waiting in the queue.

<br>

**`window`** (default: 60s): The time over which to average the API's in-flight requests. The longer the window, the slower the autoscaler will react to changes in in-flight requests, since it is averaged over the `window`. An API's in-flight requests is calculated every 10 seconds, so `window` must be a multiple of 10 seconds.

<br>

**`downscale_stabilization_period`** (default: 5m): The API will not scale below the highest recommendation made during this period. Every 10 seconds, the autoscaler makes a recommendation based on all of the other configuration parameters described here. It will then take the max of the current recommendation and all recommendations made during the `downscale_stabilization_period`, and use that to determine the final number of replicas to scale to. Increasing this value will cause the cluster to react more slowly to decreased traffic, and will reduce thrashing.

<br>

**`upscale_stabilization_period`** (default: 1m): The API will not scale above the lowest recommendation made during this period. Every 10 seconds, the autoscaler makes a recommendation based on all of the other configuration parameters described here. It will then take the min of the current recommendation and all recommendations made during the `upscale_stabilization_period`, and use that to determine the final number of replicas to scale to. Increasing this value will cause the cluster to react more slowly to increased traffic, and will reduce thrashing.

<br>

**`max_downscale_factor`** (default: 0.75): The maximum factor by which to scale down the API on a single scaling event. For example, if `max_downscale_factor` is 0.5 and there are 10 running replicas, the autoscaler will not recommend fewer than 5 replicas. Increasing this number will allow the cluster to shrink more quickly in response to dramatic dips in traffic.

<br>

**`max_upscale_factor`** (default: 1.5): The maximum factor by which to scale up the API on a single scaling event. For example, if `max_upscale_factor` is 10 and there are 5 running replicas, the autoscaler will not recommend more than 50 replicas. Increasing this number will allow the cluster to grow more quickly in response to dramatic spikes in traffic.

<br>

**`downscale_tolerance`** (default: 0.05): Any recommendation falling within this factor below the current number of replicas will not trigger a scale down event. For example, if `downscale_tolerance` is 0.1 and there are 20 running replicas, a recommendation of 18 or 19 replicas will not be acted on, and the API will remain at 20 replicas. Increasing this value will prevent thrashing, but setting it too high will prevent the cluster from maintaining it's optimal size.

<br>

**`upscale_tolerance`** (default: 0.05): Any recommendation falling within this factor above the current number of replicas will not trigger a scale up event. For example, if `upscale_tolerance` is 0.1 and there are 20 running replicas, a recommendation of 21 or 22 replicas will not be acted on, and the API will remain at 20 replicas. Increasing this value will prevent thrashing, but setting it too high will prevent the cluster from maintaining it's optimal size.

<br>

## Autoscaling instances

Cortex spins up and down instances based on the aggregate resource requests of all APIs. The number of instances will be at least `min_instances` and no more than `max_instances` for each node group (configured during installation and modifiable via `cortex cluster scale`).

## Overprovisioning

The default value for `target_in_flight` is `max_concurrency`, which behaves well in many situations (see above for an explanation of how `target_in_flight` affects autoscaling). However, if your application is sensitive to spikes in traffic or if creating new replicas takes too long (see below), you may find it helpful to maintain extra capacity to handle the increased traffic while new replicas are being created. This can be accomplished by setting `target_in_flight` to a lower value relative to the expected replica's concurrency. The smaller `target_in_flight` is, the more unused capacity your API will have, and the more room it will have to handle sudden increased load. The increased request rate will still trigger the autoscaler, and your API will stabilize again (maintaining the overprovisioned capacity).

For example, if you've determined that each replica in your API can handle 2 concurrent requests, you would typically set `target_in_flight` to 2. In a scenario where your API is receiving 8 concurrent requests on average, the autoscaler would maintain 4 live replicas (8/2 = 4). If you wanted to overprovision by 25%, you could set `target_in_flight` to 1.6, causing the autoscaler maintain 5 live replicas (8/1.6 = 5).

## Autoscaling responsiveness

Assuming that `window` and `upscale_stabilization_period` are set to their default values (1 minute), it could take up to 2 minutes of increased traffic before an extra replica is requested. As soon as the additional replica is requested, the replica request will be visible in the output of `cortex get`, but the replica won't yet be running. If an extra instance is required to schedule the newly requested replica, it could take a few minutes for AWS to provision the instance (depending on the instance type), plus a few minutes for the newly provisioned instance to download your api image and for the api to initialize.

Keep these delays in mind when considering overprovisioning (see above) and when determining appropriate values for `window` and `upscale_stabilization_period`. If you want the autoscaler to react as quickly as possible, set `upscale_stabilization_period` and `window` to their minimum values (0s and 10s respectively).
