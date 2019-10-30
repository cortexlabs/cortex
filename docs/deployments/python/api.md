# APIs

A python implementation to enable flexible model serving at scale. 

If you have an Tensorflow exported model or an ONNX model go here.

## Config

```yaml
- kind: api
  name: <string>  # API name (required)
  python:
    inference: <string>  # path to the inference implementation python file, relative to the cortex root (required)
  tracker:
    key: <string>  # key to track (required if the response payload is a JSON object)
    model_type: <string>  # model type, must be "classification" or "regression"
  compute:
    min_replicas: <int>  # minimum number of replicas (default: 1)
    max_replicas: <int>  # maximum number of replicas (default: 100)
    init_replicas: <int>  # initial number of replicas (default: <min_replicas>)
    target_cpu_utilization: <int>  # CPU utilization threshold (as a percentage) to trigger scaling (default: 80)
    cpu: <string | int | float>  # CPU request per replica (default: 200m)
    gpu: <int>  # GPU request per replica (default: 0)
    mem: <string>  # memory request per replica (default: Null)
  metadata: <string: value>  # dictionary that can be used to configure custom values    
```

## Example

```yaml
- kind: api
  name: my-api
  python:
    inference: inference.py
  compute:
    gpu: 1
```

## Inference

An Python implementation that defines how to initialize a model and use it to make predictions on requests. An `init` function can be defined to perform setup such as downloading a model or configuring a tokenizer. The `predict` function definition is required to make a prediction on the data passed in the request. The `predict` function can be used to 

See [inference](./inference.md) for a detailed guide.


## Prediction Monitoring

`tracker` can be configured to collect API prediction metrics and display real-time stats in `cortex get <api_name>`. The tracker looks for scalar values in the response payload (after the execution of the `post_inference` request handler, if provided). If the response payload is a JSON object, `key` can be set to extract the desired scalar value. For regression models, the tracker should be configured with `model_type: regression` to collect float values and display regression stats such as min, max and average. For classification models, the tracker should be configured with `model_type: classification` to collect integer or string values and display the class distribution.

## Debugging

You can log information about each request by adding a `?debug=true` parameter to your requests. This will print:

1. The raw sample
2. The value after running the `predict` function

## Autoscaling Replicas

Cortex adjusts the number of replicas that are serving predictions by monitoring the compute resource usage of each API. The number of replicas will be at least `min_replicas` and no more than `max_replicas`.

## Autoscaling Nodes

Cortex spins up and down nodes based on the aggregate resource requests of all APIs. The number of nodes will be at least `min_instances` and no more than `max_instances` (configured during installation and modifiable via the [AWS console](https://docs.aws.amazon.com/autoscaling/ec2/userguide/as-manual-scaling.html)).
