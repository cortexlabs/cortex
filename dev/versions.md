# Upgrade notes

## eksctl

1. Find the latest release on [GitHub](https://github.com/weaveworks/eksctl/releases) and check the changelog
1. Update the version in `manager/Dockerfile`
1. Update eks configuration file as necessary (make sure to maintain all Cortex environment variables)
1. Check that `eksctl utils write-kubeconfig` log filter still behaves as desired
1. Update eksctl on your dev machine: `curl --location "https://github.com/weaveworks/eksctl/releases/download/0.19.0/eksctl_$(uname -s)_amd64.tar.gz" | tar xz -C /tmp && sudo mv -f /tmp/eksctl /usr/local/bin`

## Kubernetes

1. Find the latest version of Kubernetes supported by eksctl ([source code](https://github.com/weaveworks/eksctl/blob/master/pkg/apis/eksctl.io/v1alpha5/types.go))
1. Update the version in `generate_eks.py`
1. See instructions for upgrading the Kubernetes client below

## AWS CNI

1. Check which version of the CNI is used by default
1. Update the go module version (see `Go > Non-versioned modules` section below)
1. If new instances types were added, check if `pkg/lib/aws/servicequotas.go` needs to be updated for the new instances

### AWS CNI (depreciated since we stopped setting it manually)

1. Find the latest release on [GitHub](https://github.com/aws/amazon-vpc-cni-k8s/releases) and check the changelog
1. Update the version in `install.sh`
1. Update the go module version (see `Go > Non-versioned modules` section below)
1. If new instances types were added, check if `pkg/lib/aws/servicequotas.go` needs to be updated for the new instances

## Go

1. Find the latest release on Golang's [release page](https://golang.org/doc/devel/release.html) (or [downloads page](https://golang.org/dl/)) and check the changelog
1. Search the codebase for the current minor version (e.g. `1.14`), update versions as appropriate
1. Update your local version and alert developers:
   * Linux:
     1. `wget https://dl.google.com/go/go1.14.2.linux-amd64.tar.gz`
     1. `tar -xvf go1.14.2.linux-amd64.tar.gz`
     1. `sudo rm -rf /usr/local/go`
     1. `sudo mv -f go /usr/local`
     1. `rm go1.14.2.linux-amd64.tar.gz`
     1. refresh shell
     1. `go version`
   * Mac:
     1. `brew upgrade go`
     1. refresh shell
     1. `go version`
1. Update go modules as necessary

## Go modules

### Kubernetes client

Note: check their [install.md](https://github.com/kubernetes/client-go/blob/master/INSTALL.md) for the latest instructions. These apply for k8s versions before v1.17.0:

1. Find the latest patch release for the minor kubernetes version that EKS uses by default, e.g. `kubernetes-1.16.9` (here are [their versions](https://docs.aws.amazon.com/eks/latest/userguide/kubernetes-versions.html))
1. Follow the "Update non-versioned modules" instructions using the updated version for `k8s.io/client-go`

### docker/engine/client

1. Find the latest tag from [releases](https://github.com/docker/engine/releases)
1. Follow the "Update non-versioned modules" instructions using the updated version for `docker/engine`

_note: docker client installation may be able to be improved, see https://github.com/moby/moby/issues/39302#issuecomment-639687466_

### cortexlabs/yaml

1. Check [go-yaml/yaml](https://github.com/go-yaml/yaml) to see if there were new releases since [cortexlabs/yaml](https://github.com/cortexlabs/yaml)
1. `git clone git@github.com:cortexlabs/yaml.git && cd yaml`
1. `git remote add upstream https://github.com/go-yaml/yaml && git fetch upstream`
1. `git merge upstream/v2`
1. `git push origin v2`
1. Follow the "Update non-versioned modules" instructions using the desired commit sha for `cortexlabs/yaml`

### cortexlabs/go-input

1. Check [tcnksm/go-input](https://github.com/tcnksm/go-input) to see if there were new releases since [cortexlabs/go-input](https://github.com/cortexlabs/go-input)
1. `git clone git@github.com:cortexlabs/go-input.git && cd go-input`
1. `git remote add upstream https://github.com/tcnksm/go-input && git fetch upstream`
1. `git merge upstream/master`
1. `git push origin master`
1. Follow the "Update non-versioned modules" instructions using the desired commit sha for `cortexlabs/go-input`

### Non-versioned modules

1. `rm -rf go.mod go.sum && go mod init && go clean -modcache`
1. `go get k8s.io/client-go@kubernetes-1.16.9 && go get k8s.io/apimachinery@kubernetes-1.16.9 && go get k8s.io/api@kubernetes-1.16.9`
1. `go get github.com/aws/amazon-vpc-cni-k8s/pkg/awsutils@v1.6.0`
1. `go get github.com/cortexlabs/yaml@581aea36a2e4db10f8696587e48cac5248d64f4d`
1. `go get github.com/cortexlabs/go-input@8b67a7a7b28d1c45f5c588171b3b50148462b247`
1. `echo -e '\nreplace github.com/docker/docker => github.com/docker/engine v19.03.8' >> go.mod`
1. `go get -u github.com/docker/distribution`
1. `go mod tidy`
1. For every non-indirect, non-hardcoded dependency in go.mod, update with `go get -u <path>`
1. `go mod tidy`
1. `make test-go`
1. `go mod tidy`
1. Check that the diff in `go.mod` is reasonable

### request-monitor

1. `cd images/request-monitor/`
1. `rm -rf go.mod go.sum && go mod init && go clean -modcache`
1. `go mod tidy`
1. Check that the diff in `go.mod` is reasonable

## Python

The same Python version should be used throughout Cortex (e.g. search for `3.6` and update all accordingly).

It's probably safest to use the minor version of Python that you get when you run `apt-get install python3` ([currently that's what TensorFlow's Docker image does](https://github.com/tensorflow/tensorflow/blob/master/tensorflow/tools/dockerfiles/dockerfiles/cpu.Dockerfile)), or what you get by default in Google CoLab. In theory, it should be safe to use the lowest of the maximum supported python versions in our pip dependencies (e.g. [tensorflow](https://pypi.org/project/tensorflow), [Keras](https://pypi.org/project/Keras), [numpy](https://pypi.org/project/numpy), [pandas](https://pypi.org/project/pandas), [scikit-learn](https://pypi.org/project/scikit-learn), [scipy](https://pypi.org/project/scipy), [torch](https://pypi.org/project/torch), [xgboost](https://pypi.org/project/xgboost))

## TensorFlow / TensorFlow Serving

1. Find the latest release on [GitHub](https://github.com/tensorflow/tensorflow/releases)
1. Search the codebase for the current minor TensorFlow version (e.g. `2.1`) and update versions as appropriate

Note: it's ok if example training notebooks aren't upgraded, as long as the exported model still works

## CUDA

1. Update the `nvidia/cuda` base image in `images/python-predictor-gpu/Dockerfile` and `images/onnx-predictor-gpu/Dockerfile` (as well as `libnvinfer` in `images/python-predictor-gpu/Dockerfile` and `images/tensorflow-serving-gpu/Dockerfile`) to the desired version based on [TensorFlow's documentation](https://www.tensorflow.org/install/gpu#ubuntu_1804_cuda_101) / [TensorFlow's compatability table](https://www.tensorflow.org/install/source#gpu) ([Dockerhub](https://hub.docker.com/r/nvidia/cuda)) (it's possible these versions will diverge depending on ONNX runtime support)

## ONNX runtime

1. Update the version in `images/onnx-predictor-cpu/Dockerfile` and `images/onnx-predictor-gpu/Dockerfile` ([releases](https://github.com/microsoft/onnxruntime/releases))
1. Update the version listed for `onnxruntime` in "Pre-installed Packages" in `onnx.md`
1. Search the codebase for the previous ONNX runtime version

## Nvidia device plugin

1. Update the version in `images/nvidia/Dockerfile` ([releases](https://github.com/NVIDIA/k8s-device-plugin/releases), [Dockerhub](https://hub.docker.com/r/nvidia/k8s-device-plugin))
1. In the [GitHub Repo](https://github.com/NVIDIA/k8s-device-plugin), find the latest release and go to this file (replacing the version number): <https://github.com/NVIDIA/k8s-device-plugin/blob/1.0.0-beta/nvidia-device-plugin.yml>
1. Copy the contents to `manager/manifests/nvidia.yaml`
   1. Update the link at the top of the file to the URL you copied from
   1. Check that your diff is reasonable (and put back any of our modifications, e.g. the image path, rolling update strategy, resource requests, tolerations, node selector, priority class, etc)
1. Confirm GPUs work for PyTorch, TensorFlow, and ONNX models

## Inferentia device plugin

1. Check if [k8s-neuron-device-plugin](https://github.com/aws/aws-neuron-sdk/blob/master/docs/neuron-container-tools/k8s-neuron-device-plugin.yml) has been updated since the last time (we're running the version listed here: https://github.com/aws/aws-neuron-sdk/issues/102). If so, then update `images/inferentia/Dockerfile` and update `manager/manifests/inferentia.yaml` with the latest and replace the container's image with `$CORTEX_IMAGE_INFERENTIA`. Currently, all device versions are residing at [robertlucian/cortexlabs-inferentia](https://hub.docker.com/repository/docker/robertlucian/cortexlabs-inferentia), because the ECR repo that was hosting the image seems to have been taken down. See https://github.com/cortexlabs/cortex/issues/1133.

## Python packages

1. Update versions in `images/python-predictor-*/Dockerfile`, `images/tensorflow-predictor/Dockerfile`, and `images/onnx-predictor-*/Dockerfile`
1. To determine the versions used in `images/python-predictor-inf/Dockerfile`, run `pip install --extra-index-url https://pip.repos.neuron.amazonaws.com neuron-cc tensorflow-neuron torch-neuron` from a clean environment and check what versions of all the dependencies are installed.
1. Update versions in `pkg/workloads/cortex/serve/requirements.txt` and `pkg/workloads/cortex/downloader/requirements.txt`
1. Update the versions listed in "Pre-installed packages" in `realtime-api/predictors.md` and `batch-api/predictors.md` (look at the diff carefully since some packages are not shown, and e.g. `tensorflow-cpu` -> `tensorflow`)
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
   1. Check that your diff is reasonable (there may have been other modifications to the file which should be preserved, like resource requests)
1. You can confirm the metric server is running by showing the logs of the metrics-server pod, or via `kubectl get deployment metrics-server -n kube-system` and `kubectl get apiservice v1beta1.metrics.k8s.io -o yaml`

Note: overriding horizontal-pod-autoscaler-sync-period on EKS is currently not supported (<https://github.com/awslabs/amazon-eks-ami/issues/176>)

## Neuron RTD

1. Run a `cortexlabs/neuron-rtd` container and check if there are newer versions of `aws-neuron-tools` and `aws-neuron-runtime` with `yum info <package>` command.
1. Set this version in `images/neuron-rtd/Dockerfile`, `images/python-predictor-inf/Dockerfile`, and `images/tensorflow-serving-inf/Dockerfile`.
1. Rebuild `images/neuron-rtd/Dockerfile`, `images/python-predictor-inf/Dockerfile`, `images/tensorflow-serving-inf/Dockerfile` images and test Inferentia examples.

## Inferentia temporary workarounds

To make the Inferentia work with Cortex, 3 unofficial solutions are still required:

1. Custom version of the [cluster-autoscaler](https://github.com/kubernetes/autoscaler) as built on [robertlucian/cortexlabs-cluster-autoscaler:v1.16.6-6c98931](https://hub.docker.com/repository/docker/robertlucian/cortexlabs-cluster-autoscaler).
1. Custom AMI for `inf1` instances. The currently used AMI image `ami-07a7b48058cfe1a73` has been built off of `ami-011c865bf7da41a9d` image (which is an EKS-optimized AMI version for EKS 1.6). [These](https://github.com/aws/aws-neuron-sdk/blob/master/docs/neuron-runtime/nrt_start.md) instructions have been used to build the image. Alongside that, one more thing that has to be done and is not mentioned in the instructions is to set `vm.nr_hugepages` in `/etc/sysctl.conf` to `0` and to disable the `neuron-rtd` service. Be sure that `neuron-discovery` service is still left enabled.
1. The Inferentia device plugin points to `robertlucian/cortexlabs-inferentia` (see above).

## Cluster autoscaler

1. Find the latest patch release for our current version of k8s (e.g. k8s v1.16 -> cluster-autocluster v1.16.5) on [GitHub](https://github.com/kubernetes/autoscaler/releases) and check the changelog
1. In the [GitHub Repo](https://github.com/kubernetes/autoscaler/blob/master/cluster-autoscaler/cloudprovider/aws), set the tree to the tag for the chosen release, and open `cloudprovider/aws/examples/cluster-autoscaler-autodiscover.yaml` (e.g. <https://github.com/kubernetes/autoscaler/blob/cluster-autoscaler-1.16.5/cluster-autoscaler/cloudprovider/aws/examples/cluster-autoscaler-autodiscover.yaml>)
1. Resolve merge conflicts with the template in `manager/manifests/cluster-autoscaler.yaml.j2`
1. Update the base image in `images/cluster-autoscaler/Dockerfile` to the repository URL shown in the GitHub release

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
1. Update the datadog client version in `pkg/workloads/cortex/serve/requirements.txt`

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

1. Find the latest 2.X release on [GitHub](https://github.com/helm/helm/releases) (Istio does not work with helm 3)
1. Update the version in `images/manager/Dockerfile`

## Ubuntu base images

1. Search the codebase for `ubuntu` and update versions as appropriate

## Alpine base images

1. Find the latest release on [Dockerhub](https://hub.docker.com/_/alpine)
1. Search the codebase for `alpine` and update accordingly
