# Upgrade notes

## `eksctl`

1. Update the version in the manager Dockerfile
1. Update `eks.yaml` as necessary (make sure to maintain all Cortex environment variables)
1. Check that `eksctl utils write-kubeconfig` log filter still behaves as desired

## Kubernetes

1. Find the latest version of Kubernetes supported by eksctl ([source code](https://github.com/weaveworks/eksctl/blob/master/pkg/apis/eksctl.io/v1alpha5/types.go))
1. Update the version in `eks.yaml`

## AWS CNI

1. Update the version in `install_cortex.sh`

## Go

1. Search the codebase for the current minor version (e.g. `1.11`), update versions as appropriate
1. Update go modules as necessary
1. Update your local version and alert developers

## Go modules

### Non-versioned modules

1. `go clean -modcache`
1. `rm go.mod go.sum`
1. `go mod tidy`
1. Replace these lines in go.mod:
    github.com/cortexlabs/yaml v2.2.4
    k8s.io/client-go v12.0.0
    k8s.io/api 7525909cc6da
    k8s.io/apimachinery 1799e75a0719
1. `go mod tidy`
1. Check the diff in go.mod

### Kubernetes client

1. Find the commit for the [client-go](https://github.com/kubernetes/client-go) release and browse to Godeps/Godeps.json to find the SHAs for k8s.io/api and k8s.io/apimachinery
1. Follow the "Update non-versioned modules" instructions, inserting the applicable versions for `k8s.io/*`

### `cortexlabs/yaml`

1. Follow the "Update non-versioned modules" instructions, inserting the desired version for `cortexlabs/yaml`

## TensorFlow / TensorFlow Serving / Python / Python base operating system

The Python version in the base images for tf-api and onnx-serve-gpu determines the Python version used thorughout Cortex.

1. Update the `tensorflow/tensorflow` base image in the tf-api Dockerfile to the desired version
1. Update the `nvidia/cuda` base image in the onnx-serve-gpu Dockerfile to the latest
1. Run `docker run --rm -it tensorflow/tensorflow:X.XX.X-py3`, and in the container run `python3 --version` and `cat /etc/lsb-release`
1. Run `docker run --rm -it nvidia/cuda:XX.X-cudnnX-devel`, and in the container run `python3 --version` and `cat /etc/lsb-release`
1. The Ubuntu and Python versions must match; if they do not, downgrade whichever one is too advanced
1. Search the codebase for the current minor TensorFlow version (e.g. `1.13`), update versions as appropriate
1. Search the codebase for the minor Python version (e.g. `3.6`) and update versions as appropriate
1. Search the codebase for ubuntu and update versions as appropriate

Note: it's ok if example training notebooks aren't upgraded, as long as the exported model still works\

## ONNX runtime

1. Update `ONNXRUNTIME_VERSION` in the onnx-serve and onnx-serve-gpu Dockerfiles

## Nvidia device plugin

1. Update the version in the nvidia Dockerfile
1. In the [GitHub Repo](https://github.com/NVIDIA/k8s-device-plugin), find the latest release and go to this file (replacing the version number): <https://github.com/NVIDIA/k8s-device-plugin/blob/1.0.0-beta/nvidia-device-plugin.yml>
1. Copy the contents to `nvidia.yaml`
1. In `nvidia.yaml`, update this line of config:
   `- image: $CORTEX_IMAGE_NVIDIA`
1. In `nvidia.yaml`, update the link at the top of the file to the URL you copied from
1. Confirm GPUs work for TensorFlow and ONNX models

## Misc Python packages

1. Update versions in `lib/requirements.txt`, `tf_api/requirements.txt`, and `onnx_serve/requirements.txt`

## Istio

1. Update the version in all `istio-*` dockerfiles
1. Update the version in the manager Dockerfile
1. Update `istio-values.yaml`, `apis.yaml`, and `operator.yaml` as necessary (make sure to maintain all Cortex environment variables)
1. Update `setup_istio()` in `install_cortex.sh` as necessary

## Metrics server

1. Update the version in the metrics-server Dockerfile
1. In the [GitHub Repo](https://github.com/kubernetes-incubator/metrics-server), find the latest release and go to this diectory (replacing the version number): <https://github.com/kubernetes-incubator/metrics-server/tree/v0.3.3/deploy/1.8+>
1. Copy the contents of all of the files in that directory into `metrics-server.yaml`
1. In `metrics-server.yaml`, update this line of config:
   `image: $CORTEX_IMAGE_METRICS_SERVER`
1. In `metrics-server.yaml`, update the link at the top of the file to the URL you copied from
1. You can confirm the metric server is running by showing the logs of the metrics-server pod, or via `kubectl get deployment metrics-server -n kube-system` and `kubectl get apiservice v1beta1.metrics.k8s.io -o yaml`

Note: overriding horizontal-pod-autoscaler-sync-period on EKS is currently not supported (<https://github.com/awslabs/amazon-eks-ami/issues/176>)

## Cluster autoscaler

1. In the [GitHub Repo](https://github.com/kubernetes/autoscaler/blob/cluster-autoscaler-1.15.1/cluster-autoscaler/cloudprovider/aws), find the latest release and go to this file (replacing the version number): <https://github.com/kubernetes/autoscaler/blob/cluster-autoscaler-1.15.1/cluster-autoscaler/cloudprovider/aws/examples/cluster-autoscaler-autodiscover.yaml>
1. Copy the contents to `cluster-autoscaler.yaml`
1. In `cluster-autoscaler.yaml`, update these two lines of config:
   `image: $CORTEX_IMAGE_CLUSTER_AUTOSCALER`
   `- --node-group-auto-discovery=asg:tag=k8s.io/cluster-autoscaler/enabled,k8s.io/cluster-autoscaler/$CORTEX_CLUSTER`
1. In `cluster-autoscaler.yaml`, update the link at the top of the file to the URL you copied from
1. Check your diff to make sure it makes sense
1. Update the version in the Dockerfile

## Fluentd

1. Update the version in the fluentd Dockerfile
1. Update `fluentd.yaml` as necessary (make sure to maintain all Cortex environment variables)

## Statsd

1. Update the version in the statsd Dockerfile
1. In this [GitHub Repo](https://github.com/aws-samples/amazon-cloudwatch-container-insights), open [k8s-yaml-templates/cwagent-statsd/cwagent-statsd-daemonset.yaml](https://github.com/aws-samples/amazon-cloudwatch-container-insights/blob/master/k8s-yaml-templates/cwagent-statsd/cwagent-statsd-daemonset.yaml) and [k8s-yaml-templates/cwagent-statsd/cwagent-statsd-configmap.yaml](https://github.com/aws-samples/amazon-cloudwatch-container-insights/blob/master/k8s-yaml-templates/cwagent-statsd/cwagent-statsd-configmap.yaml), and update `statsd.yaml` as necessary
1. Update the datadog client version in `lib/requirements.txt`

## `aws-iam-authenticator`

1. Update the version in the manager Dockerfile

## `kubectl`

1. Update the version in the manager Dockerfile
1. Update your local version and alert developers

## `helm`

1. Update the version in the manager Dockerfile

## Alpine base images

1. Search the codebase for alpine, update accordingly
