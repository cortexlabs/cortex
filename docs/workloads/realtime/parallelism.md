# Replica parallelism

Replica parallelism can be configured with the following fields in the `predictor` configuration:

* `processes_per_replica` (default: 1): Each replica runs a web server with `processes_per_replica` processes. For APIs running with multiple CPUs per replica, using 1-3 processes per unit of CPU generally leads to optimal throughput. For example, if `cpu` is 2, a value between 2 and 6 `processes_per_replica` is reasonable. The optimal number will vary based on the workload's characteristics and the CPU compute request for the API.

* `threads_per_process` (default: 1): Each process uses a thread pool of size `threads_per_process` to process requests. For applications that are not CPU intensive such as high I/O (e.g. downloading files), GPU-based inference, or Inferentia-based inference, increasing the number of threads per process can increase throughput. For CPU-bound applications such as running your model inference on a CPU, using 1 thread per process is recommended to avoid unnecessary context switching. Some applications are not thread-safe, and therefore must be run with 1 thread per process.

`processes_per_replica` * `threads_per_process` represents the total number of requests that your replica can work on concurrently. For example, if `processes_per_replica` is 2 and `threads_per_process` is 2, and the replica was hit with 5 concurrent requests, 4 would immediately begin to be processed, and 1 would be waiting for a thread to become available. If the replica were hit with 3 concurrent requests, all three would begin processing immediately.
