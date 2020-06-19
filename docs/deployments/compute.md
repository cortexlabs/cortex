# Compute

_WARNING: you are on the master branch, please refer to the docs on the branch that matches your `cortex version`_

Compute resource requests in Cortex follow the syntax and meaning of [compute resources in Kubernetes](https://kubernetes.io/docs/concepts/configuration/manage-compute-resources-container).

For example:

```yaml
- name: my-api
  ...
  compute:
    cpu: 1
    gpu: 1
    mem: 1G
```

CPU, GPU, Inf, and memory requests in Cortex correspond to compute resource requests in Kubernetes. In the example above, the API will only be scheduled once 1 CPU, 1 GPU, and 1G of memory are available on any instance, and it will be guaranteed to have access to those resources throughout its execution. In some cases, resource requests can be (or may default to) `Null`.

## CPU

One unit of CPU corresponds to one virtual CPU on AWS. Fractional requests are allowed, and can be specified as a floating point number or via the "m" suffix (`0.2` and `200m` are equivalent).

## GPU

One unit of GPU corresponds to one virtual GPU. Fractional requests are not allowed.

See [GPU documentation](gpus.md) for more information.

## Memory

One unit of memory is one byte. Memory can be expressed as an integer or by using one of these suffixes: `K`, `M`, `G`, `T` (or their power-of two counterparts: `Ki`, `Mi`, `Gi`, `Ti`). For example, the following values represent roughly the same memory: `128974848`, `129e6`, `129M`, `123Mi`.

## Inf

One unit of Inf corresponds to one Inferentia ASIC with 4 NeuronCores *(not the same thing as `cpu`)* and 8GB of cache memory *(not the same thing as `mem`)*. Fractional requests are not allowed.
