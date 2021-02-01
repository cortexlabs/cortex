# Install Cortex on your Kuberenetes cluster

## Prerequisites

* kubectl
* gsutil
* helm 3
* GKE cluster
    * kubernetes version >= 1.17
    * at least 2 e2.medium (2 vCPU, 4 GB mem) (with monitoring and logging disabled)

Note that installing Cortex on your kubernetes cluster will not provide some of the cluster level features such as cluster autoscaling and spot instances with on-demand backup.

## Download cortex charts

<!-- CORTEX_VERSION -->
```
wget https://s3-us-west-2.amazonaws.com/get-cortex/master/manifests/cortex-master.tar.gz
tar -xzf cortex-master.tar.gz
```

## Create a bucket in S3

The Cortex operator will use this bucket to store API states and dependencies.

```yaml
gsutil mb gs://CORTEX_GCS_BUCKET
```

## Provide credentials

The credentials need to have at least these [permissions](credentials.md).

```yaml
export $CORTEX_GOOGLE_APPLICATION_CREDENTIALS=

kubectl create secret generic 'gcp-credentials' --namespace default --from-file=key.json=$CORTEX_GOOGLE_APPLICATION_CREDENTIALS
```

## Install cortex

Define a `values.yaml` with the following information provided:

```yaml
# values.yaml

cortex:
  project: <CORTEX_REGION>
  zone: <CORTEX_ZONE>
  bucket: <CORTEX_GCS_BUCKET>
  cluster_name: <CORTEX_CLUSTER_NAME>
global:
  provider: "gcp"
```

## Install with your configuration

```bash
helm install cortex manifests/ --namespace default -f values.yaml
```

## Configure Cortex client

Get the operator endpoint
```bash
kubectl get svc ingress-operator ingressgateway-operator -n=default | grep "ip"
```

Wait for AWS to provision the loadbalancers and connect to your cluster. You can use the curl command below to verify loadbalancer set up.

It can take between 5 to 10 minutes for the setup to complete. You can expect to encounter `Could not resolve host` or timeouts when running the verification request below during the loadbalancer initialization.

```bash
curl http://<CORTEX_OPERATOR_ENDPOINT>/verifycortex -m 5
```

A successful response looks like this:

```bash
$ curl http://<CORTEX_OPERATOR_ENDPOINT>/verifycortex -m 5

{"provider":"gcp"}
```

## Using GPU resources on your cluster

The following tolerations are added to Deployments and Jobs orchestrated by Cortex.

```yaml
  - effect: NoSchedule
    key: nvidia.com/gpu
    operator: Exists
```