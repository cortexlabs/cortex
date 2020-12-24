# Setting up kubectl

## Install kubectl

Follow these [instructions](https://kubernetes.io/docs/tasks/tools/install-kubectl).

## Install gcloud

Follow these [instructions](https://cloud.google.com/sdk/docs/install).

## Update kubeconfig

```bash
gcloud container clusters get-credentials <cluster_name> --zone <zone> --project <project_id>
```

## Test kubectl

```bash
kubectl get pods
```
