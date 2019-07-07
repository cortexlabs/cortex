# Compute

Compute resource requests in Cortex follow the syntax and meaning of [compute resources in Kubernetes](https://kubernetes.io/docs/concepts/configuration/manage-compute-resources-container).

For example:

```yaml
- kind: api
  ...
  compute:
    cpu: "2"
    mem: "1Gi"
    gpu: 1
```

CPU and memory requests in Cortex correspond to compute resource requests in Kubernetes. In the example above, the training job will only be scheduled once 2 CPUs and 1Gi of memory are available, and the job will be guaranteed to have access to those resources throughout it's execution. In some cases, a Cortex compute resource request can be (or may default to) `Null`.

## CPU

One unit of CPU corresponds to one virtual CPU on AWS. Fractional requests are allowed, and can be specified as a floating point number or via the "m" suffix (`0.2` and `200m` are equivalent).

## Memory

One unit of memory is one byte. Memory can be expressed as an integer or by using one of these suffixes: `K`, `M`, `G`, `T` (or their power-of two counterparts: `Ki`, `Mi`, `Gi`, `Ti`). For example, the following values represent roughly the same memory: `128974848`, `129e6`, `129M`, `123Mi`.

## GPU

1. Please make sure your AWS account is subscribed to the [EKS-optimized AMI with GPU Support](https://aws.amazon.com/marketplace/pp/B07GRHFXGM).
2. Set CORTEX_NODE_TYPE to an AWS GPU instance (e.g. p2.xlarge) before installing Cortex.
3. Note that one unit of GPU corresponds to one virtual GPU on AWS. Fractional requests are not allowed.
