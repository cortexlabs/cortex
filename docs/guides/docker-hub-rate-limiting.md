# Docker Hub rate limiting

_WARNING: you are on the master branch, please refer to the docs on the branch that matches your `cortex version`_

Docker Hub's [newly enforced rate-limiting policy](https://www.docker.com/increase-rate-limits) can negatively impact your cluster. This is much likelier to be an issue if you've set `subnet_visibility: private` in your cluster configuration file, since with private subnets, all requests from all nodes are routed through the NAT Gateway, and will therefore have the same IP address (docker imposes the rate limit per IP address). If you haven't specified `subnet_visibility` or have set `subnet_visibility: public`, this is less likely to be an issue for you, since each instance will have its own IP address.

If you are affected by Docker Hub's rate limiting, your may encounter issues such as:

* your APIs typically run as expected but new replicas (during scale up) or newly submitted batch jobs suddenly stop working for a period of time and then eventually they start working again
* you encounter scaling issues for Realtime APIs
* batch jobs are stuck in an in progress state for an unusually long period of time

Follow these steps to determine if this issue is affecting your cluster:

1. [Setup kubectl](./kubectl-setup.md)
2. `kubectl get pods --all-namespaces`
3. Check the pod status column for image pull failures such as `ErrImagePull`, `ImagePullBackoff`. If you don't see any, the rate limiting may not be affecting you currently.
4. Get the pod id and namespace of a pod encountering image pull failures
5. `kubectl describe pod <pod id> --namespace <pod namespace>`
6. Under the events section, if you see error events related to docker hub rate limiting, then your cluster is likely affected by the rate limiting

## Use Cortex images from quay.io

In response to Docker Hub's new image pull policy, we have migrated our recent images (the oldest version being `0.16.0`) to the [quay.io](https://quay.io) registry. This registry allows for unlimited image pulls for unauthenticated users.

You won't have to do any modification for >= `0.23.0` versions of Cortex. For the older versions, you need to add the `quay.io` images to your cluster config and API spec.

---

In the image paths below, make sure to set <VERSION> to your cluster's version.
For Cortex cluster version >= `0.21.0`:

```yaml
image_manager: quay.io/cortexlabs/manager:<VERSION>
image_operator: quay.io/cortexlabs/operator:<VERSION>
image_downloader: quay.io/cortexlabs/downloader:<VERSION>
image_request_monitor: quay.io/cortexlabs/request-monitor:<VERSION>
image_cluster_autoscaler: quay.io/cortexlabs/cluster-autoscaler:<VERSION>
image_metrics_server: quay.io/cortexlabs/metrics-server:<VERSION>
image_nvidia: quay.io/cortexlabs/nvidia:<VERSION>
image_inferentia: quay.io/cortexlabs/inferentia:<VERSION>
image_neuron_rtd: quay.io/cortexlabs/neuron-rtd:<VERSION>
image_fluentd: quay.io/cortexlabs/fluentd:<VERSION>
image_statsd: quay.io/cortexlabs/statsd:<VERSION>
image_istio_proxy: quay.io/cortexlabs/istio-proxy:<VERSION>
image_istio_pilot: quay.io/cortexlabs/istio-pilot:<VERSION>
```

For `0.16.0` <= cluster version <= `0.20.0`, add the following two images:

```yaml
image_istio_galley: quay.io/cortexlabs/istio-galley:<VERSION>
image_istio_citadel: quay.io/cortexlabs/istio-citadel:<VERSION>
```

For Cortex cluster version < `0.16.0`, please upgrade to the latest version.

---

In the image paths below, make sure to set to your APIs' version(s). This is required for the `predictor.image` and `predictor.tensorflow_serving_image` fields.

```text
quay.io/cortexlabs/python-predictor-cpu:<VERSION>
quay.io/cortexlabs/python-predictor-gpu:<VERSION>
quay.io/cortexlabs/python-predictor-inf:<VERSION>
quay.io/cortexlabs/tensorflow-serving-cpu:<VERSION>
quay.io/cortexlabs/tensorflow-serving-gpu:<VERSION>
quay.io/cortexlabs/tensorflow-serving-inf:<VERSION>
quay.io/cortexlabs/tensorflow-predictor:<VERSION>
quay.io/cortexlabs/onnx-predictor-cpu:<VERSION>
quay.io/cortexlabs/onnx-predictor-gpu:<VERSION>
quay.io/cortexlabs/python-predictor-cpu-slim:<VERSION>
quay.io/cortexlabs/python-predictor-gpu-slim:<VERSION>-cuda10.0
quay.io/cortexlabs/python-predictor-gpu-slim:<VERSION>-cuda10.1
quay.io/cortexlabs/python-predictor-gpu-slim:<VERSION>-cuda10.2
quay.io/cortexlabs/python-predictor-gpu-slim:<VERSION>-cuda11.0
quay.io/cortexlabs/python-predictor-inf-slim:<VERSION>
quay.io/cortexlabs/tensorflow-predictor-slim:<VERSION>
quay.io/cortexlabs/onnx-predictor-cpu-slim:<VERSION>
quay.io/cortexlabs/onnx-predictor-gpu-slim:<VERSION>
```

## Paid Docker Hub subscription

Another option is to pay for the Docker Hub subscription to remove the limit on the number of image pulls. Docker Hub's updated pricing model allows unlimited pulls on a _Pro_ subscription for individuals as described [here](https://www.docker.com/pricing).

By default, the Cortex cluster pulls the images as an anonymous user. Follow [this guide](private-docker.md) to configure your Cortex cluster to pull the images as an authenticated user.

The advantage to this is that there's no need to do a `cortex cluster down`/`cortex cluster up` to authenticate with your account.

## Push to AWS ECR (Elastic Container Registry)

You can configure the Cortex cluster to use images from a different registry. A good choice is ECR on AWS. When an ECR repository resides in the same region as your Cortex cluster, there are no costs incurred when pulling images. Follow [this guide](self-hosted-images.md) to do this.