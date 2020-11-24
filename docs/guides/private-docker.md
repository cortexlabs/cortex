# Private docker registry

_WARNING: you are on the master branch, please refer to the docs on the branch that matches your `cortex version`_

Until [#1459](https://github.com/cortexlabs/cortex/issues/1459) is addressed, you can use a private docker registry for your Predictor images by following this guide.

## Local

When running Cortex locally, you can use private Docker images by running `docker login` and then `docker pull <your_image>`. The Docker image will be present on your machine, and will be accessible by Cortex the next time you run `cortex deploy`.

## Cluster

### Step 1

Install and configure kubectl ([instructions](kubectl-setup.md)).

### Step 2

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

kubectl patch serviceaccount default --namespace default \
  -p "{\"imagePullSecrets\": [{\"name\": \"registry-credentials\"}]}"
```

### Updating your credentials

First remove your old docker credentials from the cluster:

```bash
kubectl delete secret --namespace default registry-credentials
```

Then repeat step 2 above with your updated credentials.

### Removing your credentials

To remove your docker credentials from the cluster, run the following commands:

```bash
kubectl delete secret --namespace default registry-credentials

kubectl patch serviceaccount default --namespace default \
  -p "{\"imagePullSecrets\": []}"
```
