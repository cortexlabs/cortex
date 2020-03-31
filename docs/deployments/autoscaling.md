# Autoscaling

_WARNING: you are on the master branch, please refer to the docs on the branch that matches your `cortex version`_

Cortex autoscales your web services based on your configuration.

## Replica Parallelism

* `workers_per_replica` (default: 1): Each replica runs a web server with `workers_per_replica` workers, each of which runs in it's own process. For APIs running with multiple CPUs per replica, using 1-3 workers per unit of CPU generally leads to optimal throughput. For example, if `cpu` is 2, a value between 2 and 6 `workers_per_replica` is reasonable. The optimal number will vary based on the workload and the CPU request for the API.

* `threads_per_worker` (default: 1): Each worker uses a thread pool of size `threads_per_worker` to process requests. For applications that are not CPU intensive such as high I/O (e.g. downloading files) or GPU-based inference, increasing the number of threads per worker can increase throughput. For CPU-bound applications such as running your model inference on a CPU, using 1 thread per worker is recommended to avoid unnecessary context switching. Some applications are not thread-safe, and therefore must be run with 1 thread per worker.

`workers_per_replica` * `threads_per_worker` represents the number of requests that your replica can work in parallel. For example, if `workers_per_replica` is 2 and `threads_per_worker` is 2, and the replica was hit with 5 concurrent requests, 4 would immediately begin to be processed, 1 would be waiting for a thread to become available, and the concurrency for the replica would be 5. If the replica was hit with 3 concurrent requests, all three would begin processing immediately, and the replica concurrency would be 3.

## Autoscaling Replicas

* `min_replicas`: The lower bound on how many replicas can be running for an API.

* `max_replicas`: The upper bound on how many replicas can be running for an API.

* `target_replica_concurrency` (default: `workers_per_replica` * `threads_per_worker`): This is the desired number of in-flight requests per replica, and is the metric which the autoscaler uses to make scaling decisions.

  Replica concurrency is simply how many requests have been sent to a replica and have not yet been responded to (also referred to as in-flight requests). Therefore, it includes requests which are currently being processed and requests which are waiting in the replica's queue.

  The autoscaler uses this formula to determine the number of desired replicas:

  `desired replicas = sum(in-flight requests in each replica) / target_replica_concurrency`

  For example, setting `target_replica_concurrency` to `workers_per_replica` * `threads_per_worker` (the default) causes the cluster to adjust the number of replicas so that on average, requests are immediately processed without waiting in a queue, and workers/threads are never idle.

* `max_replica_concurrency` (default: 1024): This is the maximum number of in-flight requests per replica before requests are rejected with HTTP error code 503. `max_replica_concurrency` includes requests that are currently being processed as well as requests that are waiting in the replica's queue (a replica can actively process `workers_per_replica` * `threads_per_worker` requests concurrently, and will hold any additional requests in a local queue). Decreasing `max_replica_concurrency` and configuring the client to retry when it receives 503 responses will improve queue fairness by preventing requests from sitting in long queues.

  *Note (if `workers_per_replica` > 1): Because requests are randomly assigned to workers within a replica (which leads to unbalanced worker queues), clients may receive 503 responses before reaching `max_replica_concurrency`. For example, if you set `workers_per_replica: 2` and `max_replica_concurrency: 100`, each worker will be allowed to handle 50 requests concurrently. If your replica receives 90 requests, there is a possibility that more than 50 requests are routed to 1 worker, therefore each additional request beyond the 50 requests are responded with a 503.*

* `window` (default: 60s): The time over which to average the API wide in-flight requests (which is the sum of in-flight requests in each replica). The longer the window, the slower the autoscaler will react to changes in API wide in-flight requests, since it is averaged over the `window`. API wide in-flight requests is calculated every 10 seconds, so `window` must be a multiple of 10 seconds.

* `downscale_stabilization_period` (default: 5m): The API will not scale below the highest recommendation made during this period. Every 10 seconds, the autoscaler makes a recommendation based on all of the other configuration parameters described here. It will then take the max of the current recommendation and all recommendations made during the `downscale_stabilization_period`, and use that to determine the final number of replicas to scale to. Increasing this value will cause the cluster to react more slowly to decreased traffic, and will reduce thrashing.

* `upscale_stabilization_period` (default: 1m): The API will not scale above the lowest recommendation made during this period. Every 10 seconds, the autoscaler makes a recommendation based on all of the other configuration parameters described here. It will then take the min of the current recommendation and all recommendations made during the `upscale_stabilization_period`, and use that to determine the final number of replicas to scale to. Increasing this value will cause the cluster to react more slowly to increased traffic, and will reduce thrashing. The default is 0 minutes, which means that the cluster will react quickly to increased traffic.

* `max_downscale_factor` (default: 0.75): The maximum factor by which to scale down the API on a single scaling event. For example, if `max_downscale_factor` is 0.5 and there are 10 running replicas, the autoscaler will not recommend fewer than 5 replicas. Increasing this number will allow the cluster to shrink more quickly in response to dramatic dips in traffic.

* `max_upscale_factor` (default: 1.5): The maximum factor by which to scale up the API on a single scaling event. For example, if `max_upscale_factor` is 10 and there are 5 running replicas, the autoscaler will not recommend more than 50 replicas. Increasing this number will allow the cluster to grow more quickly in response to dramatic spikes in traffic.

* `downscale_tolerance` (default: 0.05): Any recommendation falling within this factor below the current number of replicas will not trigger a scale down event. For example, if `downscale_tolerance` is 0.1 and there are 20 running replicas, a recommendation of 18 or 19 replicas will not be acted on, and the API will remain at 20 replicas. Increasing this value will prevent thrashing, but setting it too high will prevent the cluster from maintaining it's optimal size.

* `upscale_tolerance` (default: 0.05): Any recommendation falling within this factor above the current number of replicas will not trigger a scale up event. For example, if `upscale_tolerance` is 0.1 and there are 20 running replicas, a recommendation of 21 or 22 replicas will not be acted on, and the API will remain at 20 replicas. Increasing this value will prevent thrashing, but setting it too high will prevent the cluster from maintaining it's optimal size.

## Autoscaling Nodes

Cortex spins up and down nodes based on the aggregate resource requests of all APIs. The number of nodes will be at least `min_instances` and no more than `max_instances` ([configured during installation](../cluster-management/config.md) and modifiable via `cortex cluster update`).
