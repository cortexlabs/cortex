# Private docker registry

_WARNING: you are on the master branch, please refer to the docs on the branch that matches your `cortex version`_

Until [#1459](https://github.com/cortexlabs/cortex/issues/1459) is addressed, you can use a private docker registry for your Predictor images by following this guide.

Note: The following steps only apply for Cortex clusters. When running Cortex locally, simply run `docker login` and then `docker pull <your_image>` from the command line.

## Step 1

Install and configure kubectl ([instructions](kubectl-setup.md)).

## Step 2

Set the following environment variables, replacing the placeholders with your docker username and password:

```bash
DOCKER_USERNAME=***
DOCKER_PASSWORD=***
```

Run the following commands:

```bash
kubectl create secret docker-registry registry-credentials \
  --namespace default \
  --docker-username=$DOCKER_USERNAME \
  --docker-password=$DOCKER_PASSWORD

kubectl patch serviceaccount default \
  --namespace default \
  -p "{\"imagePullSecrets\": [{\"name\": \"registry-credentials\"}]}"
```

## Updating your credentials

To remove your docker credentials from the cluster, run this command, then repeat step 2:

```bash
kubectl delete secret --namespace default registry-credentials
```

## Removing your credentials

To remove your docker credentials from the cluster, run the following commands:

```bash
kubectl delete secret --namespace default registry-credentials

kubectl patch serviceaccount default \
  --namespace default \
  -p "{\"imagePullSecrets\": []}"
```
