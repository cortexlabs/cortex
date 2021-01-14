# Upgrade notes

## Things to check when updating versions

* cluster up / info / down (works, and logs look good)
* check metrics server pod logs
* check cluster autoscaler pod logs
* check pod -> cluster autoscaling on cpu or gpu or inferentia
* check cluster autoscaling on cpu and gpu and inferentia
* examples
  * check logs, predictions
  * check metrics, tracker
  * make sure to try all 8 base images (tf/onnx/py gpu/cpu, tf/py inferentia)
  * confirm GPUs are used when requested

## eksctl

1. Find the latest release on [GitHub](https://github.com/weaveworks/eksctl/releases) and check the changelog
1. Update the version in `manager/Dockerfile`
1. Update `generate_eks.py` as necessary
1. Check that `eksctl utils write-kubeconfig` log filter still behaves as desired
1. Update eksctl on your dev machine: `curl --location "https://github.com/weaveworks/eksctl/releases/download/0.27.0/eksctl_$(uname -s)_amd64.tar.gz" | tar xz -C /tmp && sudo mv -f /tmp/eksctl /usr/local/bin`

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
     1. `wget https://dl.google.com/go/go1.14.7.linux-amd64.tar.gz`
     1. `tar -xvf go1.14.7.linux-amd64.tar.gz`
     1. `sudo rm -rf /usr/local/go`
     1. `sudo mv -f go /usr/local`
     1. `rm go1.14.7.linux-amd64.tar.gz`
     1. refresh shell
     1. `go version`
   * Mac:
     1. `brew upgrade go` or `brew install go@1.14`
     1. refresh shell
     1. `go version`
1. Update go modules as necessary

## Go modules

### Kubernetes client

1. Find the latest patch release for the minor kubernetes version that EKS uses by default (here are [their versions](https://docs.aws.amazon.com/eks/latest/userguide/kubernetes-versions.html))
1. Follow the "Update non-versioned modules" instructions using the updated version for `k8s.io/client-go`

### Istio client

1. Find the version of istio that we use in `images/manager/Dockerfile`
1. Follow the "Update non-versioned modules" instructions using the updated version for `istio.io/client-go`

### docker/engine/client

1. Find the latest tag from [releases](https://github.com/docker/engine/releases)
1. Follow the "Update non-versioned modules" instructions using the updated version for `docker/engine`

_note: docker client installation may be able to be improved, see https://github.com/moby/moby/issues/39302#issuecomment-639687466_

### cortexlabs/yaml

1. Check [go-yaml/yaml](https://github.com/go-yaml/yaml/commits/v2) to see if there were new releases since [cortexlabs/yaml](https://github.com/cortexlabs/yaml/commits/v2)
1. `git clone git@github.com:cortexlabs/yaml.git && cd yaml`
1. `git remote add upstream https://github.com/go-yaml/yaml && git fetch upstream`
1. `git merge upstream/v2`
1. `git push origin v2`
1. Follow the "Update non-versioned modules" instructions using the desired commit sha for `cortexlabs/yaml`

### cortexlabs/go-input

1. Check [tcnksm/go-input](https://github.com/tcnksm/go-input/commits/master) to see if there were new releases since [cortexlabs/go-input](https://github.com/cortexlabs/go-input/commits/master)
1. `git clone git@github.com:cortexlabs/go-input.git && cd go-input`
1. `git remote add upstream https://github.com/tcnksm/go-input && git fetch upstream`
1. `git merge upstream/master`
1. `git push origin master`
1. Follow the "Update non-versioned modules" instructions using the desired commit sha for `cortexlabs/go-input`

### Non-versioned modules

1. `rm -rf go.mod go.sum && go mod init && go clean -modcache`
1. `go get k8s.io/client-go@v0.17.6 && go get k8s.io/apimachinery@v0.17.6 && go get k8s.io/api@v0.17.6`
1. `go get istio.io/client-go@1.7.3 && go get istio.io/api@1.7.3`
1. `go get github.com/aws/amazon-vpc-cni-k8s/pkg/awsutils@v1.7.1`
1. `go get github.com/cortexlabs/yaml@581aea36a2e4db10f8696587e48cac5248d64f4d`
1. `go get github.com/cortexlabs/go-input@8b67a7a7b28d1c45f5c588171b3b50148462b247`
1. `echo -e '\nreplace github.com/docker/docker => github.com/docker/engine v19.03.12' >> go.mod`
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
1. Search the codebase for the current minor TensorFlow version (e.g. `2.3`) and update versions as appropriate

Note: it's ok if example training notebooks aren't upgraded, as long as the exported model still works

## CUDA

1. Update the `nvidia/cuda` base image in `images/python-predictor-gpu/Dockerfile` and `images/onnx-predictor-gpu/Dockerfile` (as well as `libnvinfer` in `images/python-predictor-gpu/Dockerfile` and `images/tensorflow-serving-gpu/Dockerfile`) to the desired version based on [TensorFlow's documentation](https://www.tensorflow.org/install/gpu) / [TensorFlow's compatability table](https://www.tensorflow.org/install/source#gpu) ([Dockerhub](https://hub.docker.com/r/nvidia/cuda)) (it's possible these versions will diverge depending on ONNX runtime support)

## ONNX runtime

1. Update the version in `images/onnx-predictor-cpu/Dockerfile` and `images/onnx-predictor-gpu/Dockerfile` ([releases](https://github.com/microsoft/onnxruntime/releases))
1. Search the codebase for the previous ONNX runtime version

## Nvidia device plugin

1. Update the version in `images/nvidia/Dockerfile` ([releases](https://github.com/NVIDIA/k8s-device-plugin/releases), [Dockerhub](https://hub.docker.com/r/nvidia/k8s-device-plugin))
1. In the [GitHub Repo](https://github.com/NVIDIA/k8s-device-plugin), find the latest release and go to this file (replacing the version number): <https://github.com/NVIDIA/k8s-device-plugin/blob/v0.6.0/nvidia-device-plugin.yml>
1. Copy the contents to `manager/manifests/nvidia.yaml`
   1. Update the link at the top of the file to the URL you copied from
   1. Check that your diff is reasonable (and put back any of our modifications, e.g. the image path, rolling update strategy, resource requests, tolerations, node selector, priority class, etc)
1. Confirm GPUs work for PyTorch, TensorFlow, and ONNX models

## Inferentia device plugin

1. Check if the image in [k8s-neuron-device-plugin.yml](https://github.com/aws/aws-neuron-sdk/blob/master/docs/neuron-container-tools/k8s-neuron-device-plugin.yml) has been updated (also check the readme in the parent directory to see if anything has changed). To check what the latest tag currently points to, run `aws ecr list-images --region us-west-2 --registry-id 790709498068 --repository-name neuron-device-plugin`, and then see which version has the same imageDigest as `latest`.
1. Copy the contents of [k8s-neuron-device-plugin.yml](https://github.com/aws/aws-neuron-sdk/blob/master/docs/neuron-container-tools/k8s-neuron-device-plugin.yml) and [k8s-neuron-device-plugin-rbac.yml](https://github.com/aws/aws-neuron-sdk/blob/master/docs/neuron-container-tools/k8s-neuron-device-plugin-rbac.yml) to `manager/manifests/inferentia.yaml`
   1. Update the links at the top of the file to the URL you copied from
   1. Check that your diff is reasonable (and put back any of our modifications)

## Neuron

1. `docker run --rm -it amazonlinux:2`
1. Run the `echo $'[neuron] ...' > /etc/yum.repos.d/neuron.repo` command from [Dockerfile.neuron-rtd](https://github.com/aws/aws-neuron-sdk/blob/master/docs/neuron-container-tools/docker-example/Dockerfile.neuron-rtd) (it needs to be updated to work properly with the new lines)
1. Run `yum info aws-neuron-tools` and `yum info aws-neuron-runtime` to check the versions that were installed, and use those versions in `images/neuron-rtd/Dockerfile`
1. Check if there are any updates to [Dockerfile.neuron-rtd](https://github.com/aws/aws-neuron-sdk/blob/master/docs/neuron-container-tools/docker-example/Dockerfile.neuron-rtd) which should be brought in to `images/neuron-rtd/Dockerfile`
1. Set the version of `aws-neuron-tools` and `aws-neuron-runtime` in `images/python-predictor-inf/Dockerfile` and `images/tensorflow-serving-inf/Dockerfile`
1. Run `docker run --rm -it ubuntu:18.04`
1. Run the first `RUN` command used in `images/tensorflow-serving-inf/Dockerfile`, having omitted the version specified for `tensorflow-model-server-neuron` and the cleanup line at the end
1. Run `apt-cache policy tensorflow-model-server-neuron` to find the version that was installed, and update it in `images/tensorflow-serving-inf/Dockerfile`
1. Check if there are any updates to [Dockerfile.tf-serving](https://github.com/aws/aws-neuron-sdk/blob/master/docs/neuron-container-tools/docker-example/Dockerfile.tf-serving) which should be brought in to `images/tensorflow-serving-inf/Dockerfile`
1. Run `docker run --rm -it ubuntu:18.04`
1. Run `apt-get update && apt-get install -y curl python3.6 python3.6-distutils` (change the python version if necessary)
1. Run `curl https://bootstrap.pypa.io/get-pip.py -o get-pip.py && python3.6 get-pip.py && pip install --upgrade pip` (change the python version if necessary)
1. Run `pip install --extra-index-url https://pip.repos.neuron.amazonaws.com neuron-cc tensorflow-neuron torch-neuron`
1. Run `pip list` to show the versions of all installed dependencies, and update `images/python-predictor-inf/Dockerfile` and the docs accordingly (`realtime-api/predictors.md` and `batch-api/predictors.md`); latest versions of dependencies that aren't shown in `pip list` can be determined on pypi.org (for `torchvision`, go to its pypi page, and use the latest patch version of the minor version which is appropriate for the version of `torch` that's installed)
1. Take a deep breath, cross your fingers, rebuild all images, and confirm that the Inferentia examples work

## Python packages

1. Update versions in `images/python-predictor-*/Dockerfile`, `images/tensorflow-predictor/Dockerfile`, and `images/onnx-predictor-*/Dockerfile`
1. Update versions in `pkg/cortex/serve/*requirements.txt` and `pkg/cortex/downloader/requirements.txt`
1. Update the versions listed in "Pre-installed packages" in `realtime-api/predictors.md` and `batch-api/predictors.md`
    * look at the diff carefully since some packages are not shown, and e.g. `tensorflow-cpu` -> `tensorflow`
    * be careful not to update any of the versions for Inferentia that are not latest in `images/python-predictor-inf/Dockerfile`
1. Rerun all examples and check their logs

## S6-overlay supervisor

1. Locate the `s6-overlay` installation in `images/python-predictor-*/Dockerfile`, `images/tensorflow-predictor/Dockerfile` and `images/onnx-predictor-*/Dockerfile`
1. Update the version in each serving image with the newer one in https://github.com/just-containers/s6-overlay.

## Nginx

1. Run a base image of ubuntu that matches the version tag used for the serving images. The running command is `docker run -it --rm <base-image>`
1. Run `apt update && apt-cache policy nginx`. Notice the latest minor version of nginx (e.g. `1.14`)
1. Locate the `nginx` package in `images/python-predictor-*/Dockerfile`, `images/tensorflow-predictor/Dockerfile` and `images/onnx-predictor-*/Dockerfile`
1. Update the version for all `nginx` appearances using the minor version from step 2 and add an asterisk at the end to denote any version (e.g. `1.14.*`)

## Istio

1. Find the latest [release](https://istio.io/latest/news/releases) and check the release notes (here are the [latest IstioOperator Options](https://istio.io/latest/docs/reference/config/istio.operator.v1alpha1/))
1. Update the version in `images/manager/Dockerfile`
1. Update the version in all `images/istio-*` Dockerfiles
1. Update `istio.yaml.j2`, `apis.yaml.j2`, `operator.yaml.j2`, and `pkg/lib/k8s` as necessary
1. Update `install.sh` as necessary

## Google Pause

1. Find the version of google pause used in the nvidia device driver yaml file referenced [here](https://cloud.google.com/kubernetes-engine/docs/how-to/gpus#installing_drivers)
1. Update the version in `images/google-pause/Dockerfile`

## Metrics server

1. Find the latest release on [GitHub](https://github.com/kubernetes-incubator/metrics-server/releases) and check the changelog
1. Update the version in `images/metrics-server/Dockerfile`
1. In the [GitHub Repo](https://github.com/kubernetes-incubator/metrics-server), find the latest release and go to this directory (replacing the version number): <https://github.com/kubernetes-incubator/metrics-server/tree/v0.3.7/deploy/1.8+>
1. Copy the contents of all of the files in that directory into `manager/manifests/metrics-server.yaml`
   1. Update this line of config:

       ```yaml
       image: $CORTEX_IMAGE_METRICS_SERVER
       ```

   1. Update the link at the top of the file to the URL you copied from
   1. Check that your diff is reasonable (there may have been other modifications to the file which should be preserved, like resource requests)
1. You can confirm the metric server is running by showing the logs of the metrics-server pod, or via `kubectl get deployment metrics-server -n kube-system` and `kubectl get apiservice v1beta1.metrics.k8s.io -o yaml`

Note: overriding horizontal-pod-autoscaler-sync-period on EKS is currently not supported (<https://github.com/awslabs/amazon-eks-ami/issues/176>)

## Cluster autoscaler

1. Find the latest patch release for our current version of k8s (e.g. k8s v1.17 -> cluster-autocluster v1.17.3) on [GitHub](https://github.com/kubernetes/autoscaler/releases) and check the changelog
1. Update the base image in `images/cluster-autoscaler/Dockerfile` to the repository URL shown in the GitHub release
1. In the [GitHub Repo](https://github.com/kubernetes/autoscaler/blob/master/cluster-autoscaler/cloudprovider/aws), set the tree to the tag for the chosen release, and open `cloudprovider/aws/examples/cluster-autoscaler-autodiscover.yaml` (e.g. <https://github.com/kubernetes/autoscaler/blob/cluster-autoscaler-1.16.5/cluster-autoscaler/cloudprovider/aws/examples/cluster-autoscaler-autodiscover.yaml>)
1. Resolve merge conflicts with the template in `manager/manifests/cluster-autoscaler.yaml.j2`

## FluentBit

1. Find the latest release on [Dockerhub](https://hub.docker.com/r/amazon/aws-for-fluent-bit/tags?page=1&ordering=last_updated)
1. Update the base image version in `images/fluent-bit/Dockerfile`
1. Update `fluent-bit.yaml` as necessary (make sure to maintain all Cortex environment variables)

## Statsd

1. Find the latest release on [Dockerhub](https://hub.docker.com/r/amazon/cloudwatch-agent/tags)
1. Update the version in `images/statsd/Dockerfile`
1. In this [GitHub Repo](https://github.com/aws-samples/amazon-cloudwatch-container-insights), set the tree to `master` and open [k8s-yaml-templates/cwagent-statsd/cwagent-statsd-daemonset.yaml](https://github.com/aws-samples/amazon-cloudwatch-container-insights/blob/master/k8s-yaml-templates/cwagent-statsd/cwagent-statsd-daemonset.yaml) and [k8s-yaml-templates/cwagent-statsd/cwagent-statsd-configmap.yaml](https://github.com/aws-samples/amazon-cloudwatch-container-insights/blob/master/k8s-yaml-templates/cwagent-statsd/cwagent-statsd-configmap.yaml)
1. Update `statsd.yaml` as necessary (this wasn't copy-pasted, so you may need to check the diff intelligently)
1. Update the datadog client version in `pkg/cortex/serve/requirements.txt`

## aws-iam-authenticator

1. Find the latest release [here](https://docs.aws.amazon.com/eks/latest/userguide/install-aws-iam-authenticator.html)
1. Update the version in `images/manager/Dockerfile`

## kubectl

1. Find the latest release [here](https://storage.googleapis.com/kubernetes-release/release/stable.txt)
1. Update the version in `images/manager/Dockerfile` and `images/operator/Dockerfile`
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

## Ubuntu base images

1. Search the codebase for `ubuntu` and update versions as appropriate

## Alpine base images

1. Find the latest release on [Dockerhub](https://hub.docker.com/_/alpine)
1. Search the codebase for `alpine` and update accordingly
