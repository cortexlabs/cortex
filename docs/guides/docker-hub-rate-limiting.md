# Docker Hub rate limiting

_WARNING: you are on the master branch, please refer to the docs on the branch that matches your `cortex version`_

*Note: This guide is only relevant for Cortex version `0.22.1` and earlier. Starting in version `0.23.0`, we've migrated from Docker Hub to Quay (which allows for unlimited image pulls for unauthenticated users). If you upgrade to version >= `0.23.0`, you can disregard this guide.*

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

There are three ways to avoid this issue:

## Use Cortex images from quay.io

In response to Docker Hub's new image pull policy, we have migrated our images to [quay.io](https://quay.io). This registry allows for unlimited image pulls for unauthenticated users.

It is possible to configure Cortex to use the images from Quay instead of Docker Hub:

### Update your cluster configuration file

Add the following to your [cluster configuration file](../aws/install.md) (e.g. `cluster.yaml`). In the image paths below, make sure to set `<VERSION>` to your cluster's version.

```yaml
# cluster.yaml

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

For cluster version <= `0.20.0`, also add the following two images:

```yaml
image_istio_galley: quay.io/cortexlabs/istio-galley:<VERSION>
image_istio_citadel: quay.io/cortexlabs/istio-citadel:<VERSION>
```

For Cortex cluster version < `0.16.0`, please upgrade your cluster to the latest version. If you upgrade to version >= `0.23.0`, you can disregard this guide.

Once you've updated your cluster configuration file, you can spin up your cluster (e.g. `cortex cluster up --config cluster.yaml`).

### Update your API configuration file(s)

To configure your APIs to use the Quay images, you cna update your [API configuration files](../deployments/realtime-api/api-configuration.md). The image paths are specified in `predictor.image` (and `predictor.tensorflow_serving_image` for APIs with `kind: tensorflow`). Be advised that by default, the Docker Hub images are used for your predictors, so you will need to specify the Quay image paths for all of your APIs.

Here is a list of available images (make sure to set `<VERSION>` to your cluster's version):

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

The advantage of this approach is that there's no need to do a `cortex cluster down`/`cortex cluster up` to authenticate with your Docker Hub account.

By default, the Cortex cluster pulls the images as an anonymous user. To configure your Cortex cluster to pull the images as an authenticated user, follow these steps:

### Step 1

Install and configure kubectl ([instructions](kubectl-setup.md)).

### Step 2

Set the following environment variables, replacing the placeholders with your docker username and password:

```bash
DOCKER_USERNAME=***
DOCKER_PASSWORD=***
```

Run the following commands:

```bash
kubectl create secret docker-registry registry-credentials \
  --namespace default \
  --docker-username=$DOCKER_USERNAME \
  --docker-password=$DOCKER_PASSWORD

kubectl create secret docker-registry registry-credentials \
  --namespace kube-system \
  --docker-username=$DOCKER_USERNAME \
  --docker-password=$DOCKER_PASSWORD

kubectl patch serviceaccount default --namespace default \
  -p "{\"imagePullSecrets\": [{\"name\": \"registry-credentials\"}]}"

kubectl patch serviceaccount operator --namespace default \
  -p "{\"imagePullSecrets\": [{\"name\": \"registry-credentials\"}]}"

kubectl patch serviceaccount fluentd --namespace default \
  -p "{\"imagePullSecrets\": [{\"name\": \"registry-credentials\"}]}"

kubectl patch serviceaccount default --namespace kube-system \
  -p "{\"imagePullSecrets\": [{\"name\": \"registry-credentials\"}]}"

kubectl patch serviceaccount cluster-autoscaler --namespace kube-system \
  -p "{\"imagePullSecrets\": [{\"name\": \"registry-credentials\"}]}"

kubectl patch serviceaccount metrics-server --namespace kube-system \
  -p "{\"imagePullSecrets\": [{\"name\": \"registry-credentials\"}]}"

# Only if you are using cortex version <= 0.20.0:
kubectl patch serviceaccount istio-cni --namespace kube-system \
  -p "{\"imagePullSecrets\": [{\"name\": \"registry-credentials\"}]}"

# Only if you are using Inferentia:
kubectl patch serviceaccount neuron-device-plugin --namespace kube-system \
  -p "{\"imagePullSecrets\": [{\"name\": \"registry-credentials\"}]}"
```

### Updating your credentials

First remove your old docker credentials from the cluster:

```bash
kubectl delete secret --namespace default registry-credentials
kubectl delete secret --namespace kube-system registry-credentials
```

Then repeat step 2 above with your updated credentials.

### Removing your credentials

To remove your docker credentials from the cluster, run the following commands:

```bash
kubectl delete secret --namespace default registry-credentials
kubectl delete secret --namespace kube-system registry-credentials

kubectl patch serviceaccount default --namespace default -p "{\"imagePullSecrets\": []}"
kubectl patch serviceaccount operator --namespace default -p "{\"imagePullSecrets\": []}"
kubectl patch serviceaccount fluentd --namespace default -p "{\"imagePullSecrets\": []}"
kubectl patch serviceaccount default --namespace kube-system -p "{\"imagePullSecrets\": []}"
kubectl patch serviceaccount cluster-autoscaler --namespace kube-system -p "{\"imagePullSecrets\": []}"
kubectl patch serviceaccount metrics-server --namespace kube-system -p "{\"imagePullSecrets\": []}"
# Only if you are using cortex version <= 0.20.0:
kubectl patch serviceaccount istio-cni --namespace kube-system -p "{\"imagePullSecrets\": []}"
# Only if you are using Inferentia:
kubectl patch serviceaccount neuron-device-plugin --namespace kube-system -p "{\"imagePullSecrets\": []}"
```

## Push to AWS ECR (Elastic Container Registry)

You can also push the Cortex images to ECR on your AWS account, and pull from your ECR repository in your cluster. Follow [this guide](self-hosted-images.md) to do this.
