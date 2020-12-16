# Table of contents

* [Get started](workloads/realtime/deploy.md)
* [Community](https://gitter.im/cortexlabs/cortex)
* [Contributing](contributing.md)

## CLI

* [Install](cli/install.md)
* [Commands](cli/commands.md)
* [Python client](cli/python-client.md)
* [Environments](cli/environments.md)
* [Telemetry](cli/telemetry.md)
* [Uninstall](cli/uninstall.md)

## Running on AWS

* [Install](aws/install.md)
* [Credentials](aws/credentials.md)
* [Security](aws/security.md)
* [Spot instances](aws/spot.md)
* [GPUs](aws/gpu.md)
* [Inferentia](aws/inferentia.md)
* [Networking](aws/networking.md)
* [VPC peering](aws/vpc-peering.md)
* [Custom domain](aws/custom-domain.md)
* [SSH into instances](aws/ssh.md)
* [REST API Gateway](aws/rest-api-gateway.md)
* [Update](aws/update.md)
* [Uninstall](aws/uninstall.md)

## Running on GCP

* [Install](gcp/install.md)
* [Credentials](gcp/credentials.md)
* [Uninstall](gcp/uninstall.md)

## Workloads

* Realtime
  * [Deploy](workloads/realtime/deploy.md)
  * [Predictor](workloads/realtime/predictors.md)
  * [Configuration](workloads/realtime/configuration.md)
  * [Statuses](workloads/realtime/statuses.md)
  * [Models](workloads/realtime/models.md)
  * [Parallelism](workloads/realtime/parallelism.md)
  * [Autoscaling](workloads/realtime/autoscaling.md)
* Batch
  * [Deploy](workloads/batch/deploy.md)
  * [Predictor](workloads/batch/predictors.md)
  * [Configuration](workloads/batch/configuration.md)
  * [Statuses](workloads/batch/statuses.md)
  * [Endpoints](workloads/batch/endpoints.md)
* Traffic Splitter
  * [Deploy](workloads/traffic-splitter/deploy.md)
  * [Python packages](workloads/traffic-splitter/configuration.md)
* Dependencies
  * [Deploy](workloads/dependencies/deploy.md)
  * [Python packages](workloads/dependencies/python-packages.md)
  * [System packages](workloads/dependencies/system-packages.md)
* [Exporting models](workloads/exporting.md)

## Troubleshooting

* [API is stuck updating](troubleshooting/stuck-updating.md)
* [404/503 API responses](troubleshooting/api-request-errors.md)
* [NVIDIA runtime not found](troubleshooting/nvidia-container-runtime-not-found.md)
* [TF session in predict()](troubleshooting/tf-session-in-predict.md)
* [Server-side batching errors](troubleshooting/server-side-batching-errors.md)

## Guides

* [View API metrics](guides/metrics.md)
* [Setting up kubectl](guides/kubectl.md)
* [Self-hosted Docker images](guides/self-hosted-images.md)
* [Private docker registry](guides/private-docker.md)
