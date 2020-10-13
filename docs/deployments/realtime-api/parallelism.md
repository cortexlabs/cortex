# Parallelism

_WARNING: you are on the master branch, please refer to the docs on the branch that matches your `cortex version`_

## Concurrency

Replica parallelism can be configured with the following fields in the `predictor` configuration:

* `processes_per_replica` (default: 1): Each replica runs a web server with `processes_per_replica` processes. For APIs running with multiple CPUs per replica, using 1-3 processes per unit of CPU generally leads to optimal throughput. For example, if `cpu` is 2, a value between 2 and 6 `processes_per_replica` is reasonable. The optimal number will vary based on the workload's characteristics and the CPU compute request for the API.

* `threads_per_process` (default: 1): Each process uses a thread pool of size `threads_per_process` to process requests. For applications that are not CPU intensive such as high I/O (e.g. downloading files), GPU-based inference, or Inferentia-based inference, increasing the number of threads per process can increase throughput. For CPU-bound applications such as running your model inference on a CPU, using 1 thread per process is recommended to avoid unnecessary context switching. Some applications are not thread-safe, and therefore must be run with 1 thread per process.

`processes_per_replica` * `threads_per_process` represents the total number of requests that your replica can work on concurrently. For example, if `processes_per_replica` is 2 and `threads_per_process` is 2, and the replica was hit with 5 concurrent requests, 4 would immediately begin to be processed, and 1 would be waiting for a thread to become available. If the replica were hit with 3 concurrent requests, all three would begin processing immediately.

## Server-side batching

Server-side batching is the process of aggregating multiple real-time requests into a single batch inference, which increases throughput at the expense of latency. Inference is triggered when either a maximum number of requests have been received, or when a certain amount of time has passed since receiving the first request, whichever comes first. Once a threshold is reached, inference is run on the received requests and responses are returned individually back to the clients. This process is transparent to the clients.

The [TensorFlow Predictor](predictors.md#tensorflow-predictor) allows for the use of the following 2 fields in the `server_side_batching` configuration:

* `max_batch_size`: The maximum number of requests to aggregate before running inference. This is an instrument for controlling throughput. The maximum size can be achieved if `batch_interval` is long enough to collect `max_batch_size` requests.

* `batch_interval`: The maximum amount of time to spend waiting for additional requests before running inference on the batch of requests. If fewer than `max_batch_size` requests are received after waiting the full `batch_interval`, then inference will run on the requests that have been received. This is an instrument for controlling latency.

In order to use server-side batching, the model's graph must be built such that batches can be accepted as input/output. The following is an example of how the input `x` and the output `y` of the graph could be shaped to be compatible with server-side batching:

```python
batch_size = None
sample_shape = [340, 240, 3] # i.e. RGB image
output_shape = [1000] # i.e. image labels

with graph.as_default():
    # ...
    x = tf.placeholder(tf.float32, shape=[batch_size] + sample_shape, name="input")
    y = tf.placeholder(tf.float32, shape=[batch_size] + output_shape, name="output")
    # ...
```

### Optimization

When optimizing for both throughput and latency, you will likely want keep the `max_batch_size` to a relatively small value. Even though a higher `max_batch_size` with a low `batch_interval` (when there are many requests coming in) can offer a significantly higher throughput, the overall latency could be quite large. The reason is that for a request to get back a response, it has to wait until the entire batch is processed, which means that the added latency due to the `batch_interval` can pale in comparison. For instance, let's assume that a single prediction takes 50ms, and that when the batch size is set to 128, the processing time for a batch is 1280ms (i.e. 10ms per sample). So while the throughput is now 5 times higher, it takes 1280ms + `batch_interval` to get back a response (instead of 50ms). This is the trade-off with server-side batching.

When optimizing for maximum throughput, a good rule of thumb is to follow these steps:

1. Determine the maximum throughput of one API replica when `server_side_batching` is not enabled (same as if `max_batch_size` were set to 1). This can be done with a load test (make sure to set `max_replicas` to 1 to disable autoscaling).
1. Determine the highest `batch_interval` with which you are still comfortable for your application. Keep in mind that the batch interval is not the only component of the overall latency - the inference on the batch and the pre/post processing also have to occur.
1. Multiply the maximum throughput from step 1 by the `batch_interval` from step 2. The result is a number which you can assign to `max_batch_size`.
1. Run the load test again. If the inference fails with that batch size (e.g. due to running out of GPU or RAM memory), then reduce `max_batch_size` to a level that works (reduce `batch_interval` by the same factor).
1. Use the load test to determine the peak throughput of the API replica. Multiply the observed throughput by the `batch_interval` to calculate the average batch size. If the average batch size coincides with `max_batch_size`, then it might mean that the throughput could still be further increased by increasing `max_batch_size`. If it's lower, then it means that `batch_interval` is triggering the inference before `max_batch_size` requests have been aggregated. If modifying both `max_batch_size` and `batch_interval` doesn't improve the throughput, then the service may be bottlenecked by something else (e.g. CPU, network IO, `processes_per_replica`, `threads_per_process`, etc).

<!-- CORTEX_VERSION_MINOR x1 -->
An example of server-side batching for the TensorFlow Predictor that has been benchmarked is found in [ResNet50 in TensorFlow](https://github.com/cortexlabs/cortex/tree/master/examples/tensorflow/image-classifier-resnet50#throughput-test).
