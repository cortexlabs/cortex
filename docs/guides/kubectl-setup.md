# Set up kubectl

_WARNING: you are on the master branch, please refer to the docs on the branch that matches your `cortex version`_

Although it is not necessary to use `kubectl` to interact with Cortex clusters, advanced users can use `kubectl` to get more granular visibility into the cluster (since Cortex is built on top of Kubernetes).

Here's how to set up `kubectl` and connect it to your existing Cortex cluster:

## Step 1

Install `kubectl` by following these [instructions](https://kubernetes.io/docs/tasks/tools/install-kubectl/).

## Step 2

Install `eksctl` by following these [instructions](https://eksctl.io/introduction/#installation).

## Step 3

Make sure that your AWS credentials are available in your current shell environment. `eksctl` will search for credentials in the `AWS_ACCESS_KEY_ID` and `AWS_SECRET_ACCESS_KEY` environment variables, or your `aws` CLI's credentials file (which can be created with `aws configure`).

## Step 4

Run the following command:

```bash
$ eksctl utils write-kubeconfig --cluster=<cluster_name> --region=<region>
```

Where `<cluster_name>` is the name of your cluster and `<region>` is the region of the cluster. These were specified when your cluster was created, either via command line prompts or your cluster configuration file (e.g. `cluster.yaml`). The default cluster name is `cortex`, and the default region is `us-east-1`.

## Step 4

Test `kubectl` against the existing Cortex cluster by running a command like the following. Your output will be different.

```bash
$ kubectl get pods

NAME                            READY   STATUS    RESTARTS   AGE
cloudwatch-agent-statsd-flwmv   1/1     Running   0          6m16s
fluentd-bv8xl                   1/1     Running   0          6m20s
fluentd-vrwhw                   1/1     Running   0          6m20s
operator-dc489b4f9-mmwkz        1/1     Running   0          6m14s
```

`kubectl` is now configured!
