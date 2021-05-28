# Upgrade notes

## eksctl

1. Find the latest release on [GitHub](https://github.com/weaveworks/eksctl/releases) and check the changelog
1. Search the code base for the old version to find where to update it (e.g. `manager/Dockerfile`)
1. Update `generate_eks.py` if necessary
1. Check that `eksctl utils write-kubeconfig` log filter still behaves as desired, and logs in `cortex cluster up` look good.
1. Update eksctl on your dev
   machine: `curl --location "https://github.com/weaveworks/eksctl/releases/download/0.50.0/eksctl_$(uname -s)_amd64.tar.gz" | tar xz -C /tmp && sudo mv -f /tmp/eksctl /usr/local/bin`
1. Check if eksctl iam polices changed by comparing the previous version of the eksctl policy docs to the new version's and update `./dev/minimum_aws_policy.json` and `docs/clusters/management/auth.md` accordingly. https://github.com/weaveworks/eksctl/blob/v0.50.0/userdocs/src/usage/minimum-iam-policies.md

## Kubernetes

1. Find the latest version of Kubernetes supported by
   eksctl ([source code](https://github.com/weaveworks/eksctl/blob/master/pkg/apis/eksctl.io/v1alpha5/types.go))
1. Update the version in `generate_eks.py`
1. See instructions for upgrading the Kubernetes client below

## AWS CNI

1. Update the CNI version in `eks_cluster.yaml` ([CNI releases](https://github.com/aws/amazon-vpc-cni-k8s/releases))
1. Update the go module version (see `Go > Non-versioned modules` section below)
1. Check if new instance types were added by running the script below (update the two env vars at the top).
   1. If there are new instance types, check if any changes need to be made to `servicequotas.go` or `validateInstanceType()`.

```bash
PREV_RELEASE=1.7.5
NEW_RELEASE=1.7.10
wget -q -O cni_supported_instances_prev.txt https://raw.githubusercontent.com/aws/amazon-vpc-cni-k8s/v${PREV_RELEASE}/pkg/awsutils/vpc_ip_resource_limit.go; wget -q -O cni_supported_instances_new.txt https://raw.githubusercontent.com/aws/amazon-vpc-cni-k8s/v${NEW_RELEASE}/pkg/awsutils/vpc_ip_resource_limit.go; git diff --no-index cni_supported_instances_prev.txt cni_supported_instances_new.txt; rm -rf cni_supported_instances_prev.txt; rm -rf cni_supported_instances_new.txt
```

## Go

1. Find the latest release on Golang's [release page](https://golang.org/doc/devel/release.html) (
   or [downloads page](https://golang.org/dl/)) and check the changelog
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

1. Find the latest patch release for the minor kubernetes version that EKS uses by default (here
   are [their versions](https://docs.aws.amazon.com/eks/latest/userguide/kubernetes-versions.html))
1. Follow the "Update non-versioned modules" instructions using the updated version for `k8s.io/client-go`

### Istio client

1. Find the version of istio that we use in `images/manager/Dockerfile`
1. Follow the "Update non-versioned modules" instructions using the updated version for `istio.io/client-go`

### docker/engine/client

1. Find the latest tag from [releases](https://github.com/docker/engine/releases)
1. Follow the "Update non-versioned modules" instructions using the updated version for `docker/engine`

_note: docker client installation may be able to be improved,
see https://github.com/moby/moby/issues/39302#issuecomment-639687466_

### cortexlabs/yaml

1. Check [go-yaml/yaml](https://github.com/go-yaml/yaml/commits/v2) to see if there were new releases
   since [cortexlabs/yaml](https://github.com/cortexlabs/yaml/commits/v2)
1. `git clone git@github.com:cortexlabs/yaml.git && cd yaml`
1. `git remote add upstream https://github.com/go-yaml/yaml && git fetch upstream`
1. `git merge upstream/v2`
1. `git push origin v2`
1. Follow the "Update non-versioned modules" instructions using the desired commit sha for `cortexlabs/yaml`

### cortexlabs/go-input

1. Check [tcnksm/go-input](https://github.com/tcnksm/go-input/commits/master) to see if there were new releases
   since [cortexlabs/go-input](https://github.com/cortexlabs/go-input/commits/master)
1. `git clone git@github.com:cortexlabs/go-input.git && cd go-input`
1. `git remote add upstream https://github.com/tcnksm/go-input && git fetch upstream`
1. `git merge upstream/master`
1. `git push origin master`
1. Follow the "Update non-versioned modules" instructions using the desired commit sha for `cortexlabs/go-input`

### Non-versioned modules

1. `rm -rf go.mod go.sum && go mod init && go clean -modcache`
1. `go get k8s.io/client-go@v0.17.6 && go get k8s.io/apimachinery@v0.17.6 && go get k8s.io/api@v0.17.6`
1. `go get istio.io/client-go@1.7.3 && go get istio.io/api@1.7.3`
1. `go get github.com/aws/amazon-vpc-cni-k8s/pkg/awsutils@v1.7.10`
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

## Nvidia device plugin

1. Update the version in `images/nvidia/Dockerfile` ([releases](https://github.com/NVIDIA/k8s-device-plugin/releases)
   , [Dockerhub](https://hub.docker.com/r/nvidia/k8s-device-plugin))
1. In the [GitHub Repo](https://github.com/NVIDIA/k8s-device-plugin), find the latest release and go to this file (
   replacing the version number): <https://github.com/NVIDIA/k8s-device-plugin/blob/v0.6.0/nvidia-device-plugin.yml>
1. Copy the contents to `manager/manifests/nvidia.yaml`
    1. Update the link at the top of the file to the URL you copied from
    1. Check that your diff is reasonable (and put back any of our modifications, e.g. the image path, rolling update
       strategy, resource requests, tolerations, node selector, priority class, etc)
1. Confirm GPUs work for PyTorch, TensorFlow, and ONNX models

## Inferentia device plugin

1. Check if the image
   in [k8s-neuron-device-plugin.yml](https://github.com/aws/aws-neuron-sdk/blob/master/docs/neuron-container-tools/k8s-neuron-device-plugin.yml)
   has been updated (also check the readme in the parent directory to see if anything has changed). To check what the
   latest tag currently points to,
   run `aws ecr list-images --region us-west-2 --registry-id 790709498068 --repository-name neuron-device-plugin`, and
   then see which version has the same imageDigest as `latest`.
1. Copy the contents
   of [k8s-neuron-device-plugin.yml](https://github.com/aws/aws-neuron-sdk/blob/master/docs/neuron-container-tools/k8s-neuron-device-plugin.yml)
   and [k8s-neuron-device-plugin-rbac.yml](https://github.com/aws/aws-neuron-sdk/blob/master/docs/neuron-container-tools/k8s-neuron-device-plugin-rbac.yml)
   to `manager/manifests/inferentia.yaml`
    1. Update the links at the top of the file to the URL you copied from
    1. Check that your diff is reasonable (and put back any of our modifications)

## Istio

1. Find the latest [release](https://istio.io/latest/news/releases) and check the release notes (here are
   the [latest IstioOperator Options](https://istio.io/latest/docs/reference/config/istio.operator.v1alpha1/))
1. Update the version in `images/manager/Dockerfile`
1. Update the version in all `images/istio-*` Dockerfiles
1. Update `istio.yaml.j2`, `apis.yaml.j2`, `operator.yaml.j2`, and `pkg/lib/k8s` as necessary
1. Update `install.sh` as necessary

## Metrics server

1. Find the latest release on [GitHub](https://github.com/kubernetes-incubator/metrics-server/releases) and check the
   changelog
1. Update the version in `images/metrics-server/Dockerfile`
1. Download the manifest referenced in the latest release in changelog
1. Copy the contents of the manifest into `manager/manifests/metrics-server.yaml`
    1. Update accordingly (e.g. image, pull policy, resource request, etc):
    1. Check that your diff is reasonable
1. You can confirm the metric server is running by showing the logs of the metrics-server pod, or
   via `kubectl get deployment metrics-server -n kube-system`
   and `kubectl get apiservice v1beta1.metrics.k8s.io -o yaml`

Note: overriding horizontal-pod-autoscaler-sync-period on EKS is currently not
supported (<https://github.com/awslabs/amazon-eks-ami/issues/176>)

## Cluster autoscaler

1. Find the latest patch release for our current version of k8s (e.g. k8s v1.17 -> cluster-autocluster v1.17.3)
   on [GitHub](https://github.com/kubernetes/autoscaler/releases) and check the changelog
1. Update the base image in `images/cluster-autoscaler/Dockerfile` to the repository URL shown in the GitHub release
1. In the [GitHub Repo](https://github.com/kubernetes/autoscaler/blob/master/cluster-autoscaler/cloudprovider/aws), set
   the tree to the tag for the chosen release, and
   open `cloudprovider/aws/examples/cluster-autoscaler-autodiscover.yaml` (
   e.g. <https://github.com/kubernetes/autoscaler/blob/cluster-autoscaler-1.16.5/cluster-autoscaler/cloudprovider/aws/examples/cluster-autoscaler-autodiscover.yaml>)
1. Resolve merge conflicts with the template in `manager/manifests/cluster-autoscaler.yaml.j2`

## FluentBit

1. Find the latest release
   on [Docker Hub](https://hub.docker.com/r/amazon/aws-for-fluent-bit/tags?page=1&ordering=last_updated)
1. Update the base image version in `images/fluent-bit/Dockerfile`
1. Update `fluent-bit.yaml` as necessary (make sure to maintain all Cortex environment variables)

## Prometheus Operator / Prometheus Config Reloader

1. Find the latest release in the [GitHub Repo](https://github.com/prometheus-operator/prometheus-operator).
1. Copy the `bundle.yaml` file contents into `prometheus-operator.yaml`.
1. Replace the image in the Deployment resource with a cortex env var.
1. Update the base image versions in `images/prometheus-operator/Dockerfile`
   and `images/prometheus-config-reloader/Dockerfile`.

## Prometheus

1. Find the latest release on [Docker Hub](https://hub.docker.com/r/prom/prometheus/tags?page=1&ordering=last_updated),
   compatible to the current version of Prometheus Operator.
1. Update the base image version in `images/prometheus/Dockerfile`.
1. Update `prometheus-monitoring.yaml` as necessary, if that's the case.

## Prometheus StatsD Exporter

1. Find the latest release
   on [Docker Hub](https://registry.hub.docker.com/r/prom/statsd-exporter/tags?page=1&ordering=last_updated).
1. Update the base image version in `images/prometheus-statsd-exporter/Dockerfile`.
1. Update `prometheus-statsd-exporter.yaml` as necessary, if that's the case.

## Prometheus DCGM Exporter

1. Run `helm template` on the DCGM charts https://github.com/NVIDIA/gpu-monitoring-tools/tree/master/deployment/dcgm-exporter and save the output somewhere temporarily.
1. Update the base image version in `images/prometheus-dcgm-exporter/Dockerfile`.
1. Update `prometheus-dcgm-exporter.yaml` as necessary, if that's the case. Keep in mind that in our k8s template, the `ServiceMonitor` was changed to a `PodMonitor`. Remove any unnecessary labels.

## Prometheus kube-state-metrics Exporter

1. Run `helm template` on the kube-state-metrics charts https://github.com/kubernetes/kube-state-metrics/tree/master/charts/kube-state-metrics and save the output somewhere temporarily.
1. Update the base image version in `images/prometheus-kube-state-metrics/Dockerfile`.
1. Update `prometheus-kube-state-metrics.yaml` as necessary, if that's the case. Keep in mind that in our k8s template, the `ServiceMonitor` was changed to a `PodMonitor`. Remove any unnecessary labels. The update can also include adjusting the resource requests.

## Prometheus Kubelet Exporter

1. Check if https://github.com/prometheus-operator/kube-prometheus/blob/main/manifests/kubernetes-serviceMonitorKubelet.yaml has changed when compared to `manager/manifests/prometheus-kubelet-exporter`.

## Prometheus Node Exporter

1. Find the latest release in the Kube Prometheus [GitHub Repo](https://github.com/prometheus-operator/kube-prometheus/blob/main/manifests/).
1. Copy the `node-exporter-*.yaml` files contents into `prometheus-node-exporter.yaml`, but keep the prometheus rules resource.
1. Replace the image in the Deployment resource with a cortex env var.
1. Update the base image versions in `images/prometheus-node-exporter/Dockerfile`
   and `images/kube-rbac-proxy/Dockerfile`.

## Grafana

1. Find the latest release
   on [Docker Hub](https://registry.hub.docker.com/r/grafana/grafana/tags?page=1&ordering=last_updated).
1. Update the base image version in `images/grafana/Dockerfile`.
1. Update `grafana.yaml` as necessary, if that's the case.

## Event Exporter

1. Find the latest release
   on [GitHub](https://github.com/opsgenie/kubernetes-event-exporter).
1. Update the base image version in `images/event-exporter/Dockerfile`.
1. Update `event-exporter.yaml` as necessary, if that's the case.

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

## Alpine base images

1. Find the latest release on [Dockerhub](https://hub.docker.com/_/alpine)
1. Search the codebase for `alpine` and update accordingly
