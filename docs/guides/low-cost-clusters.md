# Low-cost clusters

Here are some tips for keeping costs down when running small clusters:

* Consider using [spot instances](../cluster-management/spot-instances.md).
* CPUs are cheaper than GPUs, so if there is low request volume and low latency is not critical, running on CPU instances will be more cost effective.
* If traffic is low and you have multiple models, you may be able to save cost by serving all of your models from a single API, rather than using a separate API per model. This can be especially useful for GPU-based models. See our guide for [multi-model endpoints](multi-model.md).
* If you need to have your cluster scale down to 0 API instances \(the Cortex operator instance cannot be terminated\), you must have `min_instances` set to 0 for your cluster, and no APIs can be running. Use `cortex get` to list your APIs, and `cortex delete <api_name>` to delete each one. After ~10 minutes, your cluster should scale down to 0 API instances.
* By default, Cortex performs rolling updates on APIs, which means that during an update, additional instances may be required. If downtime during an update is acceptable, you can disable rolling updates. See [here](../troubleshooting/stuck-updating.md#disabling-rolling-updates) for instructions.

