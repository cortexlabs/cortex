# Set up kubectl

_WARNING: you are on the master branch, please refer to the docs on the branch that matches your `cortex version`_

Since Cortex is built on top of Kubernetes, you can get more granular control over your Cortex cluster by interacting with the `kubectl` CLI.

The following are the steps required to get `kubectl` set up for your existing Cortex cluster. This will only apply to Linux/Mac machines as the Cortex CLI was built for these OSes.

## Step 1

Install `eksctl` by following these [instructions](https://eksctl.io/introduction/#installation). Don't forget to have the AWS credentials set too.

## Step 2

Install `kubectl` by following these [instructions](https://kubernetes.io/docs/tasks/tools/install-kubectl/).

## Step 3

Run the following command.

```bash
eksctl utils write-kubeconfig --cluster=<cluster_name> --region=<region>
```

Where `<cluster_name>` is the name of your cluster as specified in `cluster.yaml` and with `<region>` being the region of the cluster, still as it was specified in the `cluster.yaml` config.

## Step 4

Test `kubectl` against the existing Cortex cluster by running a command like the following.

```bash
kubectl get pods


```
