# Private Docker registry

## Install and configure kubectl

Follow the instructions for [AWS](../cloud/aws/kubectl.md) or [GCP](../cloud/gcp/kubectl.md).

## Setting credentials

```bash
DOCKER_USERNAME=***
DOCKER_PASSWORD=***

kubectl create secret docker-registry registry-credentials \
  --namespace default \
  --docker-username=$DOCKER_USERNAME \
  --docker-password=$DOCKER_PASSWORD

kubectl patch serviceaccount default --namespace default \
  -p "{\"imagePullSecrets\": [{\"name\": \"registry-credentials\"}]}"
```

## Deleting credentials

```bash
kubectl delete secret --namespace default registry-credentials

kubectl patch serviceaccount default --namespace default \
  -p "{\"imagePullSecrets\": []}"
```
