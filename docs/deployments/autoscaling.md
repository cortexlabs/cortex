# Autoscaling

_WARNING: you are on the master branch, please refer to the docs on the branch that matches your `cortex version`_

Cortex autoscales your web services based on your configuration.

## Autoscaling Replicas

Cortex adjusts the number of replicas that are serving predictions by monitoring the request queue of each API. The number of replicas will always be at least `min_replicas` and no more than `max_replicas`.

Here are the parameters which affect the autoscaling behavior:

* `workers_per_replica` (default: 1): Each replica runs a web server with `workers_per_replica` workers, each of which runs in it's own process. For APIs running with multiple CPUs per replica, using 1-3 workers per unit of CPU generally leads to optimal throughput. For example, if `cpu` is 2, a value between 2 and 6 `workers_per_replica` is reasonable. The optimal number will vary based on the workload and the CPU request for the API.

* `threads_per_worker` (default: 1): Each worker uses a thread pool of size `threads_per_worker` to process requests. For CPU-bound applications, using 1 thread per worker is recommended to avoid unnecessary context switching. For applications with I/O, or for applications which utilize both the CPU and the GPU heavily, increasing the number of threads per worker can increase throughput. Some applications are not thread-safe, and therefore must be run with 1 thread per worker.

* `target_queue_length` (default: 0): This is the desired queue length per replica, and is the metric which the autoscaler uses to make scaling decisions. The queue length is not inclusive of requests that are currently being processed. For example, if `workers_per_replica` is 2 and `threads_per_worker` is 2, and the replica was hit with 5 concurrent requests, 4 would immediately begin to be processed, and 1 would be waiting for a thread to become available, leading to a queue length of 1. With only 4 concurrent requests, all would be processed immediately, and the queue length would be 0. (these are slightly idealized numbers, since in practice requests are assigned to workers at random)

In other words, the autoscaler creates or removes replicas so that for the average request, there are `target_queue_length` requests ahead of it in the queue (note that each replica has it's own queue).

Here is the equation that the autoscaler maintains:

`(workers_per_replica * threads_per_worker) + target_queue_length = average in-flight requests per replica`

For example, setting `target_queue_length` to 0 (the default) causes the cluster to adjust the number of replicas so that on average, requests are immediately processed without waiting in a queue, and workers/threads are never idle.

* `max_queue_length` (default: 100): This is the maximum number of queued requests per replica before requests are rejected with HTTP error code 503. `queue_length` does not include requests that are currently being processed by the replica (a replica can process `workers_per_replica * threads_per_worker` requests concurrently).

* `window` (default: 60s): The time over which to average an API's queue length. The longer the window, the slower the autoscaler will be to react to changes in queue length, since queue length values will be averaged over the `window`. Queue length is calculated by each replica every second, and is reported every 10 seconds, so `window` must be a multiple of 10 seconds.

* `downscale_stabilization_period` (default: 5m): The API will not scale below the highest recommendation made during this period. Every 10 seconds, the autoscaler makes a recommendation based on all of the other configuration parameters described here. It will then take the max of the current recommendation and all recommendations made during the `downscale_stabilization_period`, and use that to determine the final number of replicas to scale to. Increasing this value will cause the cluster to react more slowly to decreased traffic, and will reduce thrashing.

* `upscale_stabilization_period` (default: 0m): The API will not scale above the lowest recommendation made during this period. Every 10 seconds, the autoscaler makes a recommendation based on all of the other configuration parameters described here. It will then take the min of the current recommendation and all recommendations made during the `upscale_stabilization_period`, and use that to determine the final number of replicas to scale to. Increasing this value will cause the cluster to react more slowly to increased traffic, and will reduce thrashing. The default is 0 mintues, which means that the cluster will react quickly to increased traffic.

* `max_downscale_factor` (default: 0.5): The maximum factor by which to scale down the API on a single scaling event. For example, if `max_downscale_factor` is 0.5 and there are 10 running replicas, the autoscaler will not recommend fewer than 5 replicas. Increasing this number will allow the cluster to shrink more quickly in response to dramatic dips in traffic.

* `max_upscale_factor` (default: 10): The maximum factor by which to scale up the API on a single scaling event. For example, if `max_upscale_factor` is 10 and there are 5 running replicas, the autoscaler will not recommend more than 50 replicas. Increasing this number will allow the cluster to grow more quickly in response to dramatic spikes in traffic.

* `downscale_tolerance` (default: 0.1): Any recommendation falling within this factor below the current number of replicas will not trigger a scale down event. For example, if `downscale_tolerance` is 0.1 and there are 20 running replicas, a recommendation of 18 or 19 replicas will not be acted on, and the API will remain at 20 replicas. Increasing this value will prevent thrashing, but setting it too high will prevent the cluster from maintaining it's optimal size.

* `upscale_tolerance` (default: 0.1): Any recommendation falling within this factor above the current number of replicas will not trigger a scale up event. For example, if `upscale_tolerance` is 0.1 and there are 20 running replicas, a recommendation of 21 or 22 replicas will not be acted on, and the API will remain at 20 replicas. Increasing this value will prevent thrashing, but setting it too high will prevent the cluster from maintaining it's optimal size.

## Autoscaling Nodes

Cortex spins up and down nodes based on the aggregate resource requests of all APIs. The number of nodes will be at least `min_instances` and no more than `max_instances` (configured during installation and modifiable via `cortex cluster update` or the [AWS console](https://docs.aws.amazon.com/autoscaling/ec2/userguide/as-manual-scaling.html)).
