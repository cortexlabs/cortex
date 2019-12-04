# Upgrade notes

## eksctl

1. Find the latest release on [GitHub](https://github.com/weaveworks/eksctl/releases) and check the changelog
1. Update the version in `manager/Dockerfile`
1. Update `eks.yaml` as necessary (make sure to maintain all Cortex environment variables)
1. Check that `eksctl utils write-kubeconfig` log filter still behaves as desired
1. Update eksctl on your dev machine: `curl --location "https://github.com/weaveworks/eksctl/releases/download/0.5.3/eksctl_$(uname -s)_amd64.tar.gz" | tar xz -C /tmp && sudo mv -f /tmp/eksctl /usr/local/bin`

## Kubernetes

1. Find the latest version of Kubernetes supported by eksctl ([source code](https://github.com/weaveworks/eksctl/blob/master/pkg/apis/eksctl.io/v1alpha5/types.go))
1. Update the version in `eks.yaml`

## Go

1. Find the latest release on Golang's [release page](https://golang.org/doc/devel/release.html) (or [downloads page](https://golang.org/dl/)) and check the changelog
1. Search the codebase for the current minor version (e.g. `1.12`), update versions as appropriate
1. Update your local version and alert developers:
   * Linux:
     1. `wget https://dl.google.com/go/go1.12.13.linux-amd64.tar.gz`
     1. `tar -xvf go1.12.13.linux-amd64.tar.gz`
     1. `sudo rm -rf /usr/local/go`
     1. `sudo mv -f go /usr/local`
     1. `rm go1.12.13.linux-amd64.tar.gz`
     1. refresh shell
     1. `go version`
   * Mac:
     1. `brew upgrade go`
     1. refresh shell
     1. `go version`
1. Update go modules as necessary

## Go modules

### Kubernetes client

1. Find the minor version of the latest stable release from the [README](https://github.com/kubernetes/client-go), and/or find the latest tagged patch version from [releases](https://github.com/kubernetes/client-go/releases)
1. Follow the "Update non-versioned modules" instructions using the updated version for `k8s.io/client-go`

### docker/engine/client

1. Find the latest tag from [releases](https://github.com/docker/engine/releases)
1. Follow the "Update non-versioned modules" instructions using the updated version for `docker/engine`

### cortexlabs/yaml

1. Follow the "Update non-versioned modules" instructions using the desired version for `cortexlabs/yaml`

### Non-versioned modules

1. `rm go.mod go.sum`
1. `go mod init`
1. `go clean -modcache`
1. `go get k8s.io/client-go@v12.0.0`
1. `go get github.com/cortexlabs/yaml@v2.2.4`
1. `echo -e '\nreplace github.com/docker/docker => github.com/docker/engine v19.03.4' >> go.mod`
1. `go mod tidy`
1. Check that the diff in `go.mod` is reasonable

## TensorFlow / TensorFlow Serving / Python / Python base operating system

The Python version in the base images for `tf-api` and `onnx-serve-gpu`/`predictor-serve-gpu` determines the Python version used throughout Cortex.

1. Update the `tensorflow/tensorflow` base image in `images/tf-api/Dockerfile` to the desired version ([Dockerhub](https://hub.docker.com/r/tensorflow/tensorflow))
1. Update the `nvidia/cuda` base image in `images/onnx-serve-gpu/Dockerfile` to the desired version ([Dockerhub](https://hub.docker.com/r/nvidia/cuda))
1. Run `docker run --rm -it tensorflow/tensorflow:***`, and in the container run `python3 --version` and `cat /etc/lsb-release`
1. Run `docker run --rm -it nvidia/cuda:***`, and in the container run `python3 --version` and `cat /etc/lsb-release`
1. The Ubuntu and Python versions must match; if they do not, downgrade whichever one is too advanced
1. Search the codebase for the current minor TensorFlow version (e.g. `1.14`) and update versions as appropriate
1. Search the codebase for the minor Python version (e.g. `3.6`) and update versions as appropriate
1. Search the codebase for `ubuntu` and update versions as appropriate

Note: it's ok if example training notebooks aren't upgraded, as long as the exported model still works

## ONNX runtime

1. Update `ONNXRUNTIME_VERSION` in `images/onnx-serve/Dockerfile` and `images/onnx-serve-gpu/Dockerfile` ([releases](https://github.com/microsoft/onnxruntime/releases))
1. Update the version listed for `onnxruntime` in "Pre-installed Packages" in `request-handlers.py`

## Nvidia device plugin

1. Update the version in `images/nvidia/Dockerfile` ([releases](https://github.com/NVIDIA/k8s-device-plugin/releases), [Dockerhub](https://hub.docker.com/r/nvidia/k8s-device-plugin))
1. In the [GitHub Repo](https://github.com/NVIDIA/k8s-device-plugin), find the latest release and go to this file (replacing the version number): <https://github.com/NVIDIA/k8s-device-plugin/blob/1.0.0-beta/nvidia-device-plugin.yml>
1. Copy the contents to `manager/manifests/nvidia.yaml`
   1. Update this line of config:

       ```yaml
       - image: $CORTEX_IMAGE_NVIDIA
       ```

   1. Update the link at the top of the file to the URL you copied from
   1. Check that your diff is reasonable
1. Confirm GPUs work for TensorFlow and ONNX models

## Python packages

1. Update versions in `pkg/workloads/cortex/lib/requirements.txt`, `pkg/workloads/cortex/tf_api/requirements.txt`, `pkg/workloads/cortex/onnx_serve/requirements.txt`, and `pkg/workloads/cortex/predictor_serve/requirements.txt`
1. Update the versions listed in "Pre-installed packages" in `request-handlers.md` and `predictor.md`
1. Rerun all examples and check their logs

## Istio

1. Find the latest [release](https://github.com/istio/istio/releases/) and check the [changelog](https://istio.io/about/notes/) and [option changes](https://istio.io/docs/reference/config/installation-options-changes/) (here are the [latest configuration options](https://istio.io/docs/reference/config/installation-options/))
1. Update the version in all `images/istio-*` Dockerfiles
1. Update the version in `images/manager/Dockerfile`
1. Update `istio-values.yaml`, `apis.yaml`, and `operator.yaml` as necessary (make sure to maintain all Cortex environment variables)
1. Update `setup_istio()` in `install.sh` as necessary

## Metrics server

1. Find the latest release on [GitHub](https://github.com/kubernetes-incubator/metrics-server/releases) and check the changelog
1. Update the version in `images/metrics-server/Dockerfile`
1. In the [GitHub Repo](https://github.com/kubernetes-incubator/metrics-server), find the latest release and go to this directory (replacing the version number): <https://github.com/kubernetes-incubator/metrics-server/tree/v0.3.4/deploy/1.8+>
1. Copy the contents of all of the files in that directory into `manager/manifests/metrics-server.yaml`
   1. Update this line of config:

       ```yaml
       image: $CORTEX_IMAGE_METRICS_SERVER
       ```

   1. Update the link at the top of the file to the URL you copied from
   1. Check that your diff is reasonable
1. You can confirm the metric server is running by showing the logs of the metrics-server pod, or via `kubectl get deployment metrics-server -n kube-system` and `kubectl get apiservice v1beta1.metrics.k8s.io -o yaml`

Note: overriding horizontal-pod-autoscaler-sync-period on EKS is currently not supported (<https://github.com/awslabs/amazon-eks-ami/issues/176>)

## Cluster autoscaler

1. Find the latest patch release for our current version of k8s (e.g. k8s v1.14 -> cluster-autocluster v1.14.5) on [GitHub](https://github.com/kubernetes/autoscaler/releases) and check the changelog
1. In the [GitHub Repo](https://github.com/kubernetes/autoscaler/blob/master/cluster-autoscaler/cloudprovider/aws), set the tree to the tag for the chosen release, and open `cloudprovider/aws/examples/cluster-autoscaler-autodiscover.yaml` (e.g. <https://github.com/kubernetes/autoscaler/blob/cluster-autoscaler-1.14.5/cluster-autoscaler/cloudprovider/aws/examples/cluster-autoscaler-autodiscover.yaml>)
1. Copy the contents to `manager/manifests/cluster-autoscaler.yaml`
   1. Update this line of config:

       ```yaml
       - image: $CORTEX_IMAGE_CLUSTER_AUTOSCALER
       ```

   1. Replace `<YOUR CLUSTER NAME>` with `$CORTEX_CLUSTER_NAME`
   1. Update the link at the top of the file to the URL you copied from
   1. Check that your diff is reasonable
1. Update the version of the base image in `images/cluster-autoscaler/Dockerfile` to the tag of the chosen release

## Fluentd

1. Find the latest release on [Dockerhub](https://hub.docker.com/r/fluent/fluentd-kubernetes-daemonset/)
1. Update the base image version in `images/fluentd/Dockerfile`
1. Update record-modifier in `images/fluentd/Dockerfile` to the latest version [here](https://github.com/repeatedly/fluent-plugin-record-modifier/blob/master/VERSION)
1. Update `fluentd.yaml` as necessary (make sure to maintain all Cortex environment variables)

## Statsd

1. Find the latest release on [Dockerhub](https://hub.docker.com/r/amazon/cloudwatch-agent/tags)
1. Update the version in `images/statsd/Dockerfile`
1. In this [GitHub Repo](https://github.com/aws-samples/amazon-cloudwatch-container-insights), set the tree to `master` and open [k8s-yaml-templates/cwagent-statsd/cwagent-statsd-daemonset.yaml](https://github.com/aws-samples/amazon-cloudwatch-container-insights/blob/master/k8s-yaml-templates/cwagent-statsd/cwagent-statsd-daemonset.yaml) and [k8s-yaml-templates/cwagent-statsd/cwagent-statsd-configmap.yaml](https://github.com/aws-samples/amazon-cloudwatch-container-insights/blob/master/k8s-yaml-templates/cwagent-statsd/cwagent-statsd-configmap.yaml)
1. Update `statsd.yaml` as necessary (this wasn't copy-pasted, so you may need to check the diff intelligently)
1. Update the datadog client version in `lib/requirements.txt`

## aws-iam-authenticator

1. Find the latest release [here](https://docs.aws.amazon.com/eks/latest/userguide/install-aws-iam-authenticator.html)
1. Update the version in `images/manager/Dockerfile`

## kubectl

1. Find the latest release [here](https://storage.googleapis.com/kubernetes-release/release/stable.txt)
1. Update the version in `images/manager/Dockerfile`
1. Update your local version and alert developers
   * Linux:
     1. `curl -LO https://storage.googleapis.com/kubernetes-release/release/$(curl -s https://storage.googleapis.com/kubernetes-release/release/stable.txt)/bin/linux/amd64/kubectl`
     1. `chmod +x ./kubectl`
     1. `sudo mv -f ./kubectl /usr/local/bin/kubectl`
     1. refresh shell
     1. `kubectl version`
   * Mac:
     1. `brew upgrade kubernetes-cli`
     1. refresh shell
     1. `kubectl version`

## helm

1. Find the latest release on [GitHub](https://github.com/helm/helm/releases)
1. Update the version in `images/manager/Dockerfile`

## Alpine base images

1. Find the latest release on [Dockerhub](https://hub.docker.com/_/alpine)
1. Search the codebase for `alpine` and update accordingly
