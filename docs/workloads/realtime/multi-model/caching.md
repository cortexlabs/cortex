# Multi-model caching

Multi-model caching allows each replica to serve more models than would fit into its memory by keeping a specified number of models in memory (and disk) at a time. When the in-memory model limit is reached, the least recently accessed model is evicted from the cache. This can be useful when you have many models, and some models are frequently accessed while a larger portion of them are rarely used, or when running on smaller instances to control costs.

The model cache is a two-layer cache, configured by the following parameters in the `predictor.models` configuration:

* `cache_size` sets the number of models to keep in memory
* `disk_cache_size` sets the number of models to keep on disk (must be greater than or equal to `cache_size`)

Both of these fields must be specified, in addition to either the `dir` or `paths` field (which specifies the model paths, see [models](../../realtime/models.md) for documentation). Multi-model caching is only supported if `predictor.processes_per_replica` is set to 1 (the default value).

## Out of memory errors

Cortex runs a background process every 10 seconds that counts the number of models in memory and on disk, and evicts the least recently used models if the count exceeds `cache_size` / `disk_cache_size`. If many new models are requested between executions of the process, there may be more models in memory and/or on disk than the configured `cache_size` or `disk_cache_size` limits which could lead to out of memory errors.
