# Setting up kubectl

## Install kubectl

Follow these [instructions](https://kubernetes.io/docs/tasks/tools/install-kubectl).

## Install the AWS CLI

Follow these [instructions](https://docs.aws.amazon.com/cli/latest/userguide/cli-chap-install.html).

## Configure the AWS CLI

```bash
aws --version  # should be >= 1.16

aws configure
```

## Update kubeconfig

```bash
aws eks update-kubeconfig --name=<cluster_name> --region=<region>
```

## Test kubectl

```bash
kubectl get pods
```
