# Parallelism

_WARNING: you are on the master branch, please refer to the docs on the branch that matches your `cortex version`_

Replica parallelism can be tweaked with the following fields:

* `workers_per_replica` (default: 1): Each replica runs a web server with `workers_per_replica` workers, each of which runs in it's own process. For APIs running with multiple CPUs per replica, using 1-3 workers per unit of CPU generally leads to optimal throughput. For example, if `cpu` is 2, a value between 2 and 6 `workers_per_replica` is reasonable. The optimal number will vary based on the workload and the CPU request for the API.

* `threads_per_worker` (default: 1): Each worker uses a thread pool of size `threads_per_worker` to process requests. For applications that are not CPU intensive such as high I/O (e.g. downloading files), GPU-based inference or Inferentia ASIC-based inference, increasing the number of threads per worker can increase throughput. For CPU-bound applications such as running your model inference on a CPU, using 1 thread per worker is recommended to avoid unnecessary context switching. Some applications are not thread-safe, and therefore must be run with 1 thread per worker.

`workers_per_replica` * `threads_per_worker` represents the number of requests that your replica can work in parallel. For example, if `workers_per_replica` is 2 and `threads_per_worker` is 2, and the replica was hit with 5 concurrent requests, 4 would immediately begin to be processed, 1 would be waiting for a thread to become available, and the concurrency for the replica would be 5. If the replica was hit with 3 concurrent requests, all three would begin processing immediately, and the replica concurrency would be 3.
