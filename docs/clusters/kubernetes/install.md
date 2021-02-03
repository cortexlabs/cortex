# Install

Cortex currently relies on cloud provider specific functionality such as load balancers and storage. Kubernetes clusters in the following cloud providers are supported:

* [AWS](#aws)
* [GCP](#gcp)

Cortex uses helm to install the Cortex operator and its dependencies on your Kubernetes cluster.

## AWS

### Prerequisites

* kubectl
* aws cli
* helm 3
* EKS cluster
  * at least 3 t3.medium (2 vCPU, 4 GB mem) instances

You may install Cortex in any namespace in your cluster. In the guide that follows, the "default" namespace is assumed; if you're using a different namespace, replace all occurrences of "default" with the name of your namespace.

Note that installing Cortex on your Kubernetes cluster will not provide some of the cluster-level features such as cluster autoscaling and spot instances with on-demand backup.

### Download Cortex charts

<!-- CORTEX_VERSION_BRANCH_STABLE x3 -->
```bash
wget https://s3-us-west-2.amazonaws.com/get-cortex/master/charts/cortex-master.tar.gz
tar -xzf cortex-master.tar.gz
```

### Create a bucket in S3

The Cortex operator will use this bucket to store API state and dependencies.

```yaml
aws s3api create-bucket --bucket <CORTEX_S3_BUCKET> --region <CORTEX_REGION>
```

### Credentials

The credentials need to have at least these [permissions](../aws/security.md#operator).

```yaml
export CORTEX_AWS_ACCESS_KEY_ID=<AWS_ACCESS_KEY_ID>
export CORTEX_AWS_SECRET_ACCESS_KEY=<AWS_SECRET_ACCESS_KEY>

kubectl --namespace default create secret generic 'aws-credentials' \
    --from-literal='AWS_ACCESS_KEY_ID'=$CORTEX_AWS_ACCESS_KEY_ID \
    --from-literal='AWS_SECRET_ACCESS_KEY'=$CORTEX_AWS_SECRET_ACCESS_KEY \
    -o yaml --dry-run=client | kubectl apply -f - >/dev/null
```

### Install Cortex

Define a `values.yaml` with the following information provided:

```yaml
# values.yaml

cortex:
  region: <CORTEX_REGION>
  bucket: <CORTEX_S3_BUCKET>  # e.g. "my-cortex-bucket" (without s3://)
  cluster_name: <CORTEX_CLUSTER_NAME>
global:
  provider: "aws"
```

```bash
helm install cortex charts/ --namespace default -f values.yaml
```

### Configure Cortex client

Wait for the loadbalancers to be provisioned and connected to your cluster.

Get the Cortex operator endpoint:

```bash
kubectl get service --namespace default ingressgateway-operator -o jsonpath='{.status.loadBalancer.ingress[0].hostname}'
```

You can use the curl command below to verify that your load balancer is ready. It can take 5-10 minutes for the setup to complete. You can expect to encounter `Could not resolve host` or timeouts when running the verification request before the load balancer is initialized.

```bash
export CORTEX_OPERATOR_ENDPOINT=$(kubectl get service --namespace default ingressgateway-operator -o jsonpath='{.status.loadBalancer.ingress[0].hostname}')

curl http://$CORTEX_OPERATOR_ENDPOINT/verifycortex --max-time 5
```

A successful response looks like this:

```bash
{"provider":"aws"}
```

Once you receive a successful response, configure your Cortex client:

```bash
cortex env configure --operator-endpoint $CORTEX_OPERATOR_ENDPOINT
```

### Using GPU/Inf resources on your cluster

The following tolerations are added to Deployments and Jobs orchestrated by Cortex.

```yaml
  - effect: NoSchedule
    key: nvidia.com/gpu
    operator: Exists
  - effect: NoSchedule
    key: aws.amazon.com/neuron
    operator: Equal
    value: "true"
```

## GCP

### Prerequisites

* kubectl
* gsutil
* helm 3
* GKE cluster
  * at least 2 n1-standard-2 (2 vCPU, 8 GB mem) (with monitoring and logging disabled)

You may install Cortex in any namespace in your cluster. In the guide that follows, the "default" namespace is assumed; if you're using a different namespace, replace all occurrences of "default" with the name of your namespace.

Note that installing Cortex on your Kubernetes cluster will not provide some of the cluster-level features such as cluster autoscaling and preemptible instances.

### Download Cortex charts

<!-- CORTEX_VERSION_BRANCH_STABLE x3 -->
```bash
wget https://s3-us-west-2.amazonaws.com/get-cortex/master/charts/cortex-master.tar.gz
tar -xzf cortex-master.tar.gz
```

### Create a bucket in GCS

The Cortex operator will use this bucket to store API state and dependencies.

```yaml
gsutil mb gs://CORTEX_GCS_BUCKET
```

### Credentials

The credentials need to have at least these [permissions](../gcp/credentials.md).

```yaml
export CORTEX_GOOGLE_APPLICATION_CREDENTIALS=<PATH_TO_CREDENTIALS>

kubectl create secret generic 'gcp-credentials' --namespace default --from-file=key.json=$CORTEX_GOOGLE_APPLICATION_CREDENTIALS
```

### Install Cortex

Define a `values.yaml` with the following information provided:

```yaml
# values.yaml

cortex:
  project: <CORTEX_PROJECT>
  zone: <CORTEX_ZONE>
  bucket: <CORTEX_GCS_BUCKET>  # e.g. "my-cortex-bucket" (without gs://)
  cluster_name: <CORTEX_CLUSTER_NAME>
global:
  provider: "gcp"
```

```bash
helm install cortex charts/ --namespace default -f values.yaml
```

### Configure Cortex client

Wait for the loadbalancers to be provisioned and connected to your cluster.

Get the Cortex operator endpoint:

```bash
kubectl get service --namespace default ingressgateway-operator -o jsonpath='{.status.loadBalancer.ingress[0].ip}'
```

You can use the curl command below to verify that your load balancer is ready. It can take 5-10 minutes for the setup to complete. You can expect to encounter `Could not resolve host` or timeouts when running the verification request before the load balancer is initialized.

```bash
export CORTEX_OPERATOR_ENDPOINT=$(kubectl get service --namespace default ingressgateway-operator -o jsonpath='{.status.loadBalancer.ingress[0].ip}')

curl http://$CORTEX_OPERATOR_ENDPOINT/verifycortex --max-time 5
```

A successful response looks like this:

```bash
{"provider":"gcp"}
```

Once you receive a successful response, configure your Cortex client:

```bash
cortex env configure --operator-endpoint $CORTEX_OPERATOR_ENDPOINT
```

### Using GPU resources on your cluster

The following tolerations are added to Deployments and Jobs orchestrated by Cortex.

```yaml
  - effect: NoSchedule
    key: nvidia.com/gpu
    operator: Exists
```
