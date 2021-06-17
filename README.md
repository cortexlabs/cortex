<img src='https://s3-us-west-2.amazonaws.com/cortex-public/logo.png' height='26'>

<br>

# Serverless containers on AWS

Deploy, manage, and scale containers without managing infrastructure.

### [Get started](https://docs.cortex.dev)

<br>

## Scale realtime, batch, and async workloads

**Realtime** - respond to requests in real-time and autoscale based on in-flight request volumes.

**Batch** - run distributed and fault-tolerant batch processing jobs on-demand.

**Async** - process requests asynchronously and autoscale based on request queue length.

<br>

```bash
$ cortex deploy

creating realtime text-generator
creating batch image-classifier
creating async video-analyzer
```

<br>

## Allocate CPU, GPU, and memory without limits

**No resource limits** - allocate as much CPU, GPU, and memory as each workload requires.

**No cold starts** - keep a minimum number of replicas running to ensure that requests are handled in real-time.

**No timeouts** - run workloads for as long as you want.

<br>

```bash
$ cortex get

WORKLOAD             TYPE         REPLICAS
text-generator       realtime     32
image-classifier     batch        64
video-analyzer       async        16
```

<br>

## Control your AWS spend

**Scale to zero** - optimize the autoscaling behavior of each workload to minimize idle resources.

**Multi-instance** - run different workloads on different EC2 instances to ensure efficient resource utilization.

**Spot instances** - run workloads on spot instances and fall back to on-demand instances to ensure reliability.

<br>

```bash
$ cortex cluster up

INSTANCE        PRICE     SPOT     SCALE
c5.xlarge       $0.17     yes      0-100
g4dn.xlarge     $0.53     yes      0-100
```
