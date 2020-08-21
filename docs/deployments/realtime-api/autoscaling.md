# Autoscaling

_WARNING: you are on the master branch, please refer to the docs on the branch that matches your `cortex version`_

Cortex autoscales your web services on a per-API basis based on your configuration.

## Autoscaling Replicas

**`min_replicas`**: The lower bound on how many replicas can be running for an API.

<br>

**`max_replicas`**: The upper bound on how many replicas can be running for an API.

<br>

**`target_replica_concurrency`** (default: `processes_per_replica` * `threads_per_process`): This is the desired number of in-flight requests per replica, and is the metric which the autoscaler uses to make scaling decisions.

Replica concurrency is simply how many requests have been sent to a replica and have not yet been responded to (also referred to as in-flight requests). Therefore, it includes requests which are currently being processed and requests which are waiting in the replica's queue.

The autoscaler uses this formula to determine the number of desired replicas:

`desired replicas = sum(in-flight requests accross all replicas) / target_replica_concurrency`

For example, setting `target_replica_concurrency` to `processes_per_replica` * `threads_per_process` (the default) causes the cluster to adjust the number of replicas so that on average, requests are immediately processed without waiting in a queue, and processes/threads are never idle.

<br>

**`max_replica_concurrency`** (default: 1024): This is the maximum number of in-flight requests per replica before requests are rejected with HTTP error code 503. `max_replica_concurrency` includes requests that are currently being processed as well as requests that are waiting in the replica's queue (a replica can actively process `processes_per_replica` * `threads_per_process` requests concurrently, and will hold any additional requests in a local queue). Decreasing `max_replica_concurrency` and configuring the client to retry when it receives 503 responses will improve queue fairness by preventing requests from sitting in long queues.

*Note (if `processes_per_replica` > 1): In reality, there is a queue per process; for most purposes thinking of it as a per-replica queue will be sufficient, although in some cases the distinction is relevant. Because requests are randomly assigned to processes within a replica (which leads to unbalanced process queues), clients may receive 503 responses before reaching `max_replica_concurrency`. For example, if you set `processes_per_replica: 2` and `max_replica_concurrency: 100`, each process will be allowed to handle 50 requests concurrently. If your replica receives 90 requests that take the same amount of time to process, there is a 24.6% possibility that more than 50 requests are routed to 1 process, and each request that is routed to that process above 50 is responded to with a 503. To address this, it is recommended to implement client retries for 503 errors, or to increase `max_replica_concurrency` to minimize the probability of getting 503 responses.*

<br>

**`window`** (default: 60s): The time over which to average the API wide in-flight requests (which is the sum of in-flight requests in each replica). The longer the window, the slower the autoscaler will react to changes in API wide in-flight requests, since it is averaged over the `window`. API wide in-flight requests is calculated every 10 seconds, so `window` must be a multiple of 10 seconds.

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

## Autoscaling Instances

Cortex spins up and down instances based on the aggregate resource requests of all APIs. The number of instances will be at least `min_instances` and no more than `max_instances` ([configured during installation](../../cluster-management/config.md) and modifiable via `cortex cluster configure`).
