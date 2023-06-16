# Upgrade notes

## eksctl

1. Find the latest release on [GitHub](https://github.com/weaveworks/eksctl/releases) and check the changelog
1. Search the code base for the old version to find where to update it (e.g. `manager/Dockerfile`)
1. Update `generate_eks.py` if necessary
1. Check that `eksctl utils write-kubeconfig` log filter still behaves as desired, and logs in `cortex cluster up` look good.
1. Update eksctl on your dev
   machine: `curl --location "https://github.com/weaveworks/eksctl/releases/download/v0.143.0/eksctl_$(uname -s)_amd64.tar.gz" | tar xz -C /tmp && sudo mv -f /tmp/eksctl /usr/local/bin`
1. Check if eksctl iam polices changed by comparing the previous version of the eksctl policy docs to the new version's and update `./dev/minimum_aws_policy.json` and `docs/clusters/management/auth.md` accordingly. https://github.com/weaveworks/eksctl/blob/v0.143.0/userdocs/src/usage/minimum-iam-policies.md

## Kubernetes

1. Find the latest version of Kubernetes supported by
   eksctl ([source code](https://github.com/weaveworks/eksctl/blob/master/pkg/apis/eksctl.io/v1alpha5/types.go))
1. Update the version in `generate_eks.py`
1. Update `ami.json` (see release checklist for instructions)
1. See instructions for upgrading the Kubernetes client below

## kube-proxy (IPVS mode)

1. Before spinning up a Cortex cluster with the new eksctl/kubernetes/eks updates, make sure to have the `setup_ipvs` functional call commented out in the manager.
1. Once the cluster is up, run the `cat /var/lib/kube-proxy-config/config` command on any of the kube-proxy pods of the cluster. Compare the output of that with what the `upgrade_kube_proxy_mode.py` script is applying and make sure it's still applicable, if not, check out the spec of the [KubeProxyConfiguration](https://kubernetes.io/docs/reference/config-api/kube-proxy-config.v1alpha1/) and upgrade `upgrade_kube_proxy_mode.py`.
1. Compare the spec of the `kube-proxy.patch.yaml` patch with the current spec of the kube-proxy daemoset and make sure it's still applicable. You can either inspect the `kube-proxy` command helper by exec-ing into the pod or by looking at the [kube-proxy](https://kubernetes.io/docs/reference/command-line-tools-reference/kube-proxy/) documentation for the respective version of Kubernetes.
1. Once both config map and the daemonset are updated and the kube-proxy pod(s) has/have started, make sure you notice the `Using ipvs Proxier` log.

## aws-iam-authenticator

1. Find the latest release [here](https://docs.aws.amazon.com/eks/latest/userguide/install-aws-iam-authenticator.html)
1. Update the version in `images/manager/Dockerfile`

## kubectl

1. Find the latest release [here](https://storage.googleapis.com/kubernetes-release/release/stable.txt)
1. Update the version in `images/manager/Dockerfile` and `images/operator/Dockerfile`
1. Update your local version and alert developers
    * Linux:

     ```shell
        mkdir -p $HOME/temp && \
        cd $HOME/temp && \
        curl -LO https://storage.googleapis.com/kubernetes-release/release/$(curl -s https://storage.googleapis.com/kubernetes-release/release/stable.txt)/bin/linux/amd64/kubectl && \
        chmod +x ./kubectl && \
        sudo mv -f ./kubectl /usr/local/bin/kubectl && \
        if [ -f $HOME/.bash_profile ]; then source $HOME/.bash_profile; else source $HOME/.bashrc; fi && \
        cd - && \
        kubectl version
     ```

    * Mac:
        1. `brew upgrade kubernetes-cli`
        1. refresh shell
        1. `kubectl version`

## Istio

1. Find the latest [release](https://istio.io/latest/news/releases) and check the release notes (here are
   the [latest IstioOperator Options](https://istio.io/latest/docs/reference/config/istio.operator.v1alpha1/))
1. Update the version in `images/manager/Dockerfile`.
1. Update the version in all `images/istio-*` Dockerfiles.
1. Update `istio.yaml.j2`, `apis.yaml.j2`, `operator.yaml.j2`, and `pkg/lib/k8s` as necessary.
1. Update `install.sh` as necessary.

## AWS CNI

1. Update the CNI version in `generate_eks.py` ([CNI releases](https://github.com/aws/amazon-vpc-cni-k8s/releases))
1. Update the go module version (see `Go > Non-versioned modules` section below)
1. Check if new instance types were added by running the script below (update the two env vars at the top).
   1. If there are new instance types, check if any changes need to be made to `servicequotas.go` or `validateInstanceType()`.

```bash
PREV_RELEASE=1.10.1
NEW_RELEASE=1.11.0
wget -q -O cni_supported_instances_prev.txt https://raw.githubusercontent.com/aws/amazon-vpc-cni-k8s/v${PREV_RELEASE}/pkg/awsutils/vpc_ip_resource_limit.go; wget -q -O cni_supported_instances_new.txt https://raw.githubusercontent.com/aws/amazon-vpc-cni-k8s/v${NEW_RELEASE}/pkg/awsutils/vpc_ip_resource_limit.go; git diff --no-index cni_supported_instances_prev.txt cni_supported_instances_new.txt; rm -rf cni_supported_instances_prev.txt; rm -rf cni_supported_instances_new.txt
```

## Go

1. Find the latest release on Golang's [release page](https://golang.org/doc/devel/release.html) (
   or [downloads page](https://golang.org/dl/)) and check the changelog
1. Search the codebase for the current minor version (e.g. `1.17`), update versions as appropriate
1. Update your local version and alert developers:
    * Linux:

     ```shell
        mkdir -p $HOME/temp
        cd $HOME/temp
        wget https://dl.google.com/go/go1.20.4.linux-amd64.tar.gz && \
        tar -xvf go1.20.4.linux-amd64.tar.gz && \
        sudo rm -rf /usr/local/go && \
        sudo mv -f go /usr/local && \
        rm go1.20.4.linux-amd64.tar.gz && \
        if [ -f $HOME/.bash_profile ]; then source $HOME/.bash_profile; else source $HOME/.bashrc; fi && \
        cd - && \
        go version
     ```

    * Mac:
        1. `brew upgrade go` or `brew install go@1.17`
        1. refresh shell
        1. `go version`
1. Update go modules as necessary

## Go modules

### Kubernetes client

1. Find the latest patch [release](https://github.com/kubernetes/client-go) for the minor kubernetes version that we use (e.g. for k8s 1.21, use `client-go` version `v0.21.X`, where `X` is the latest available patch release)
1. Follow the "Update non-versioned modules" instructions using the updated version for `k8s.io/client-go`

### Istio client

1. Find the version of istio that we use in `images/manager/Dockerfile`
1. Follow the "Update non-versioned modules" instructions using the updated version for `istio.io/client-go`

### docker/engine/client

1. Find the latest tag from [here](https://github.com/docker/engine/tags)
1. Follow the "Update non-versioned modules" instructions using the updated version for `docker/engine`

_note: docker client installation may be able to be improved,
see https://github.com/moby/moby/issues/39302#issuecomment-639687466_

### PEAT-AI/yaml

1. Check [go-yaml/yaml](https://github.com/go-yaml/yaml/commits/v2) to see if there were new releases
   since [PEAT-AI/yaml](https://github.com/PEAT-AI/yaml/commits/v2)
1. `git clone git@github.com:PEAT-AI/yaml.git && cd yaml`
1. `git remote add upstream https://github.com/go-yaml/yaml && git fetch upstream`
1. `git merge upstream/v2`
1. `git push origin v2`
1. Follow the "Update non-versioned modules" instructions using the desired commit sha for `PEAT-AI/yaml`

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
1. `go get k8s.io/client-go@v0.20.15 && go get k8s.io/apimachinery@v0.20.15 && go get k8s.io/api@v0.20.15`
1. `go get istio.io/client-go@v1.11.8 && go get istio.io/api@1.11.8`
1. `go get github.com/aws/amazon-vpc-cni-k8s/pkg/awsutils@v1.13.0`
1. `go get github.com/PEAT-AI/yaml@31e52ba8433b683c471ef92cf1711fe67671dac5`
1. `go get github.com/cortexlabs/go-input@8b67a7a7b28d1c45f5c588171b3b50148462b247`
1. `go get github.com/xlab/treeprint@v1.1.0`
1. `go get -u sigs.k8s.io/controller-runtime@v0.14.6`
1. `echo -e '\nreplace github.com/docker/docker => github.com/docker/engine v19.03.13' >> go.mod`
1. `go get -u github.com/docker/distribution`
1. `go mod tidy`
1. Potentially skip these steps
   1. For every non-indirect, non-hardcoded dependency in go.mod, update with `go get -u <path>`
   1. `go mod tidy`
   1. Re-run the relevant hardcoded `go get` commands above
   1. `go mod tidy`
   1. `make test`
   1. `go mod tidy`
1. Check that the diff in `go.mod` is reasonable

## Nvidia device plugin

1. Update the version in `images/nvidia-device-plugin/Dockerfile` ([releases](https://github.com/NVIDIA/k8s-device-plugin/releases)
   , [Dockerhub](https://hub.docker.com/r/nvidia/k8s-device-plugin))
1. In the [GitHub Repo](https://github.com/NVIDIA/k8s-device-plugin), find the latest release and go to this file (
   replacing the version number): <https://github.com/NVIDIA/k8s-device-plugin/blob/v0.14.0/nvidia-device-plugin.yml>
1. Copy the contents to `manager/manifests/nvidia.yaml`
    1. Update the link at the top of the file to the URL you copied from
    1. Check that your diff is reasonable (and put back any of our modifications, e.g. the image path, rolling update
       strategy, resource requests, tolerations, node selector, priority class, etc)

## Neuron device plugin and scheduler

1. Update `images/neuron-device-plugin/Dockerfile` if necessary (see [here](https://gallery.ecr.aws/neuron/neuron-device-plugin) for the latest tag)
1. Update `images/neuron-scheduler/Dockerfile` if necessary (see [here](https://gallery.ecr.aws/neuron/neuron-scheduler) for the latest tag)
1. Copy the contents of `k8s-neuron-*` in [this folder](https://github.com/aws/aws-neuron-sdk/tree/master/src/k8) into `manager/manifests/inferentia.yaml`
    1. Update the link at the top of the file to the URL you copied from
    1. Check that your diff is reasonable (and put back any of our modifications)

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

## Cluster autoscaler

1. Find the latest patch release for our current version of k8s (e.g. k8s v1.17 -> cluster-autocluster v1.20.4)
   on [GitHub](https://github.com/kubernetes/autoscaler/releases) and check the changelog
1. In the [GitHub Repo](https://github.com/kubernetes/autoscaler/blob/master/cluster-autoscaler/cloudprovider/aws),
   set the tree to the tag for the chosen release, and open `cloudprovider/aws/examples/cluster-autoscaler-autodiscover.yaml`
   (e.g. <https://github.com/kubernetes/autoscaler/blob/cluster-autoscaler-1.20.0/cluster-autoscaler/cloudprovider/aws/examples/cluster-autoscaler-autodiscover.yaml>)
1. Resolve merge conflicts with the template in `manager/manifests/cluster-autoscaler.yaml.j2`.
1. Clone our fork: `git clone git@github.com:cortexlabs/autoscaler.git`
1. Checkout our updated branch: `git checkout cluster-autoscaler-1.21.1-cortex`
1. List the most recent commit: `git log`
1. Reset the latest commit (use the SHA of the last non-cortex commit): `git reset <SHA>`
1. `git add *`
1. `git stash`
1. `git remote add upstream https://github.com/kubernetes/autoscaler.git`
1. `git fetch upstream`
1. Checkout the appropriate version tag, e.g. `git checkout cluster-autoscaler-1.22.2 -b cluster-autoscaler-1.22.2-cortex`
1. `git stash pop`
1. Resolve any merge conflicts
1. Unstage and check the diff
1. `git add *; git commit -am "Add rate limiter"`
1. `git push origin cluster-autoscaler-1.22.2-cortex`
1. Update `images/cluster-autoscaler/Dockerfile` to use the new branch name (e.g. "cluster-autoscaler-1.22.2") in the `-b` flag's value from `git clone`.
1. Match the Go version of the builder in `images/cluster-autoscaler/Dockerfile` with that of the [cluster autoscaler's Dockerfile](https://github.com/kubernetes/autoscaler/blob/master/builder/Dockerfile).

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

1. Run `helm template` on the kube-state-metrics charts from https://github.com/kubernetes/kube-state-metrics#helm-chart and save the output somewhere temporarily.
1. Update the base image version in `images/prometheus-kube-state-metrics/Dockerfile`.
1. Update `prometheus-kube-state-metrics.yaml` as necessary, if that's the case. Keep in mind that in our k8s template, the `ServiceMonitor` was changed to a `PodMonitor`. Remove any unnecessary labels. The update can also include adjusting the resource requests.

## Prometheus Kubelet Exporter

1. Check if https://github.com/prometheus-operator/kube-prometheus/blob/main/manifests/kubernetes-serviceMonitorKubelet.yaml has changed when compared to `manager/manifests/prometheus-kubelet-exporter`.

## Prometheus Node Exporter

1. Find the latest release in the Kube Prometheus [GitHub Repo](https://github.com/prometheus-operator/kube-prometheus/blob/main/manifests/).
1. Copy the `node-exporter-*.yaml` files contents into `prometheus-node-exporter.yaml`, but keep the prometheus rules resource.
1. Replace the image in the Deployment resource with a cortex env var.
1. Update the base image version in `images/prometheus-node-exporter/Dockerfile`
1. Update the base branch version in `images/kube-rbac-proxy/Dockerfile` (as well as the rest of the contents if necessary).

## Grafana

1. Find the latest release
   on [Docker Hub](https://registry.hub.docker.com/r/grafana/grafana/tags?page=1&ordering=last_updated).
1. Update the base image version in `images/grafana/Dockerfile`.
1. Update `grafana.yaml` as necessary, if that's the case.

## Event Exporter

1. Find the latest release on [GitHub](https://github.com/opsgenie/kubernetes-event-exporter) / [GitHub Container Registry](https://github.com/opsgenie/kubernetes-event-exporter/pkgs/container/kubernetes-event-exporter).
1. Update the base image version in `images/event-exporter/Dockerfile`.
1. Update `event-exporter.yaml` as necessary, if that's the case.

## Alpine base images

1. Find the latest release on [Dockerhub](https://hub.docker.com/_/alpine)
1. Search the codebase for `alpine` and update accordingly

## Python client dependencies

1. Update package versions in `install_requires` in `python/client/setup.py`.
