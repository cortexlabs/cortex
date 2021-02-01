# Install Cortex on your Kuberenetes cluster

## Prerequisites

1. kubectl
1. helm 3
1. EKS cluster with k8s == 1.17
1. t3.medium or greater (2 CPU  4 GB mem)
1. ...

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
aws s3api create-bucket --bucket <CORTEX_S3_BUCKET> --region <CORTEX_REGION>
```

## Provide credentials

The credentials need to have at least these [permissions](security.md#operator).

```yaml
export CORTEX_AWS_ACCESS_KEY_ID=
export CORTEX_AWS_SECRET_ACCESS_KEY=

kubectl -n=default create secret generic 'aws-credentials' \
    --from-literal='AWS_ACCESS_KEY_ID'=$CORTEX_AWS_ACCESS_KEY_ID \
    --from-literal='AWS_SECRET_ACCESS_KEY'=$CORTEX_AWS_SECRET_ACCESS_KEY \
    -o yaml --dry-run=client | kubectl apply -f - >/dev/null
```

## Install cortex

Define a `values.yaml` with the following information provided:

```yaml
# values.yaml

region: <CORTEX_REGION>
bucket: <CORTEX_S3_BUCKET>
cluster_name:
global:
  provider: "aws"
```

## Install with your configuration

```bash
helm install cortex manifests -n default -f values.yaml
```

## Configure Cortex client

Get the operator endpoint
```bash
kubectl get svc ingress-operator ingressgateway-operator -n=default | grep "hostname"
```

Wait for AWS to provision the loadbalancers and connect to your cluster. You can use the curl command below to verify loadbalancer set up.

It can take between 5 to 10 minutes for the setup to complete. You can expect to encounter `Could not resolve host` or timeouts when running the verification request below during the loadbalancer initialization.

```bash
curl http://<CORTEX_OPERATOR_ENDPOINT>/verifycortex -m 5
```

A successful response looks like this:

```bash
$ curl http://<CORTEX_OPERATOR_ENDPOINT>/verifycortex -m 5

{"provider":"aws"}
```


## Use GPU/Inf resources on your cluster

Cortex adds the following tolerations to all workloads:

```yaml
  - effect: NoSchedule
    key: workload
    operator: Equal
    value: "true"
  - effect: NoSchedule
    key: nvidia.com/gpu
    operator: Exists
  - effect: NoSchedule
    key: aws.amazon.com/neuron
    operator: Equal
    value: "true"
```

Add the respective taints