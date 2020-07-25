# Parallelism

_WARNING: you are on the master branch, please refer to the docs on the branch that matches your `cortex version`_

## Concurrency

Replica parallelism can be configured with the following fields in the `predictor` configuration:

* `processes_per_replica` (default: 1): Each replica runs a web server with `processes_per_replica` processes. For APIs running with multiple CPUs per replica, using 1-3 processes per unit of CPU generally leads to optimal throughput. For example, if `cpu` is 2, a value between 2 and 6 `processes_per_replica` is reasonable. The optimal number will vary based on the workload's characteristics and the CPU compute request for the API.

* `threads_per_process` (default: 1): Each process uses a thread pool of size `threads_per_process` to process requests. For applications that are not CPU intensive such as high I/O (e.g. downloading files), GPU-based inference, or Inferentia-based inference, increasing the number of threads per process can increase throughput. For CPU-bound applications such as running your model inference on a CPU, using 1 thread per process is recommended to avoid unnecessary context switching. Some applications are not thread-safe, and therefore must be run with 1 thread per process.

`processes_per_replica` * `threads_per_process` represents the total number of requests that your replica can work on concurrently. For example, if `processes_per_replica` is 2 and `threads_per_process` is 2, and the replica was hit with 5 concurrent requests, 4 would immediately begin to be processed, and 1 would be waiting for a thread to become available. If the replica were hit with 3 concurrent requests, all three would begin processing immediately.

## Batching

Server-side batching is the process of aggregating multiple requests into one. Once a threshold is hit, it runs the inference on the received samples and then returns the responses back to the clients. The threshold can either be based on a maximum number of requests (or samples) that can be aggregated together or can be interval-based. This process is totally transparent to the user. The benefit of server-side batching is that it increases the throughput at the expense of latency.

The [TensorFlow Predictor](predictors.md#tensorflow-predictor) also allows for the use of the following 2 fields in the `server_side_batching` configuration:

* `max_batch_size`: Maximum batch size when aggregating multiple requests in one. Looking at the TensorFlow Predictor's implementation, it still appears like a single prediction is made. This is an instrument for controlling the throughput. The maximum size can be achieved if `batch_interval` is big enough to collect all requests.

* `batch_interval`: How many seconds the API replica should wait to fill a batch with requests. If the batch's size `max_batch_size` isn't fulfilled in `batch_interval` seconds, then just run the predictions on what it's got at that moment. This is an instrument for controlling the latency.

In order for batches to be enabled on your API, the model(s)'(s) graph must be built such that batches can be accepted as input/output. The following is an example of how the input `x` and the output `y` of the graph would have to be shaped to be compatible with batching on Cortex.

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

When optimizing for maximum throughput, a good rule of thumb is to follow these steps:

1. Find the maximum throughput of one API replica when the `max_batch_size` is not set (same as if `max_batch_size` were set to 1).
1. Determine the highest `batch_interval` with which you are still comfortable for your application. Keep in mind that the batch timeout is not the only component of the overall latency - the inference on the batch also has to take place.
1. Multiply the maximum throughput you got with `batch_interval`. The result is a number which you can assign to `max_batch_size`. If the model fails operating with that batch size (when it runs out of video or RAM memory), then reduce it to a level that's still works. Reduce `batch_interval` by as many times as you had to reduce `max_batch_size` from its initial calculation.

<!-- CORTEX_VERSION_MINOR x1 -->
A batching example for the TensorFlow Predictor that has been benchmarked is found in [ResNet50 in TensorFlow](https://github.com/cortexlabs/cortex/tree/master/examples/tensorflow/image-classifier-resnet50#throughput-test).

#### Throughput/latency trade-off

To effectively determine what's the average batch size while serving, test an API replica for its throughput and multiply it with `batch_interval`. If the determined average batch size coincides with `max_batch_size`, then it might mean that the throughput could still be further increased by increasing `max_batch_size`. If it's lower, then it means `batch_interval` is reining in the tail latency. If modifying both fields `max_batch_size` and `batch_interval` doesn't improve the throughput, then it might mean that the service is bottlenecked by something else (i.e. CPU, network IO, `processes_per_replica`, `threads_per_process`, etc).

When optimizing for both throughput and latency, try to keep the `max_batch_size` to small values. Even though a higher `max_batch_size` (i.e. 256) with a low `batch_interval` (in case there are many requests coming in) can offer a significantly higher throughput, the overall latency would be quite big. The reason is that for a request to get back a response, it has to wait until the whole batch is processed (i.e. 256 samples), which means that the value of `batch_interval` can pale in comparison. For instance, let's assume that a single prediction takes 50ms. When the batch size is set to a high value such as 128, the processing time per prediction within a batch can be 10ms, whereas the whole batch comes in at 1280ms. So when the throughput has been tweaked to be 50ms/10ms times higher, you have to wait 1280ms instead of 50ms to get back a response. This is the trade-off with batching.
