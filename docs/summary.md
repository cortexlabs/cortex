# Table of contents

* [Get started](tutorials/realtime.md)
* [Community](https://gitter.im/cortexlabs/cortex)
* [Contributing](contributing.md)

## CLI

* [Install](cli/install.md)
* [Commands](cli/commands.md)
* [Python client](cli/python-client.md)
* [Environments](cli/environments.md)
* [Telemetry](cli/telemetry.md)
* [Uninstall](cli/uninstall.md)

## Tutorials

* [Realtime API](tutorials/realtime.md)
* [Batch API](tutorials/batch.md)
* [Multi-model API](tutorials/multi-model.md)
* [Traffic splitter](tutorials/traffic-splitter.md)
* [Project directory](tutorials/project.md)

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
  * [Predictor implementation](workloads/realtime/predictors.md)
  * [API configuration](workloads/realtime/configuration.md)
  * [API statuses](workloads/realtime/statuses.md)
  * [Models](workloads/realtime/models.md)
  * [Parallelism](workloads/realtime/parallelism.md)
  * [Autoscaling](workloads/realtime/autoscaling.md)
  * [Traffic Splitter](workloads/realtime/traffic-splitter.md)
* Batch
  * [Predictor implementation](workloads/batch/predictors.md)
  * [API configuration](workloads/batch/configuration.md)
  * [Endpoints](workloads/batch/endpoints.md)
  * [Job statuses](workloads/batch/statuses.md)
* [Python packages](workloads/python-packages.md)
* [System packages](workloads/system-packages.md)

## Troubleshooting

* [API is stuck updating](troubleshooting/stuck-updating.md)
* [404/503 API responses](troubleshooting/api-request-errors.md)
* [NVIDIA runtime not found](troubleshooting/nvidia-container-runtime-not-found.md)
* [TF session in predict()](troubleshooting/tf-session-in-predict.md)
* [Server-side batching errors](troubleshooting/server-side-batching-errors.md)

## Guides

* [Exporting models](guides/exporting.md)
* [Multi-model endpoints](guides/multi-model.md)
* [View API metrics](guides/metrics.md)
* [Setting up kubectl](guides/kubectl.md)
* [Self-hosted Docker images](guides/self-hosted-images.md)
* [Private docker registry](guides/private-docker.md)
