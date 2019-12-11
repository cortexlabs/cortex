# Compute

_WARNING: you are on the master branch, please refer to the docs on the branch that matches your `cortex version`_

Compute resource requests in Cortex follow the syntax and meaning of [compute resources in Kubernetes](https://kubernetes.io/docs/concepts/configuration/manage-compute-resources-container).

For example:

```yaml
- kind: api
  ...
  compute:
    cpu: 1
    gpu: 1
    mem: 1G
```

CPU, GPU, and memory requests in Cortex correspond to compute resource requests in Kubernetes. In the example above, the API will only be scheduled once 1 CPU, 1GPU, and 1G of memory are available on any instance, and the deployment will be guaranteed to have access to those resources throughout its execution. In some cases, resource requests can be (or may default to) `Null`.

## CPU

One unit of CPU corresponds to one virtual CPU on AWS. Fractional requests are allowed, and can be specified as a floating point number or via the "m" suffix (`0.2` and `200m` are equivalent).

## Memory

One unit of memory is one byte. Memory can be expressed as an integer or by using one of these suffixes: `K`, `M`, `G`, `T` (or their power-of two counterparts: `Ki`, `Mi`, `Gi`, `Ti`). For example, the following values represent roughly the same memory: `128974848`, `129e6`, `129M`, `123Mi`.

## GPU

1. Make sure your AWS account is subscribed to the [EKS-optimized AMI with GPU Support](https://aws.amazon.com/marketplace/pp/B07GRHFXGM).
2. You may need to [file an AWS support ticket](https://console.aws.amazon.com/support/cases#/create?issueType=service-limit-increase&limitType=ec2-instances) to increase the limit for your desired instance type.
3. Set instance type to an AWS GPU instance (e.g. p2.xlarge) when installing Cortex.
4. Note that one unit of GPU corresponds to one virtual GPU on AWS. Fractional requests are not allowed.
