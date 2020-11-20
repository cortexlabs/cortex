# Table of contents

* [Deploy machine learning models to production](../README.md)
* [Install](cluster-management/install.md)
* [Tutorial](https://docs.cortex.dev/v/0.22/deployments/realtime-api/text-generator)  <!-- CORTEX_VERSION_MINOR -->
* [GitHub](https://github.com/cortexlabs/cortex)
* [Examples](https://github.com/cortexlabs/cortex/tree/0.22/examples)  <!-- CORTEX_VERSION_MINOR -->
* [Contact us](miscellaneous/contact-us.md)

## Deployments

* [Realtime API](deployments/realtime-api.md)
  * [Predictor implementation](deployments/realtime-api/predictors.md)
  * [API configuration](deployments/realtime-api/api-configuration.md)
  * [API deployment](deployments/realtime-api/deployment.md)
  * [API statuses](deployments/realtime-api/statuses.md)
  * [Models](deployments/realtime-api/models.md)
  * [Parallelism](deployments/realtime-api/parallelism.md)
  * [Autoscaling](deployments/realtime-api/autoscaling.md)
  * [Prediction monitoring](deployments/realtime-api/prediction-monitoring.md)
  * [Traffic Splitter](deployments/realtime-api/traffic-splitter.md)
  * [Realtime API tutorial](../examples/pytorch/text-generator/README.md)
* [Batch API](deployments/batch-api.md)
  * [Predictor implementation](deployments/batch-api/predictors.md)
  * [API configuration](deployments/batch-api/api-configuration.md)
  * [API deployment](deployments/batch-api/deployment.md)
  * [Endpoints](deployments/batch-api/endpoints.md)
  * [Job statuses](deployments/batch-api/statuses.md)
  * [Batch API tutorial](../examples/batch/image-classifier/README.md)

## Advanced

* [Compute](deployments/compute.md)
* [Using GPUs](deployments/gpus.md)
* [Using Inferentia](deployments/inferentia.md)
* [Python packages](deployments/python-packages.md)
* [System packages](deployments/system-packages.md)
* [Networking](deployments/networking.md)

## Cluster management

* [Cluster configuration](cluster-management/config.md)
* [AWS credentials](cluster-management/aws-credentials.md)
* [EC2 instances](cluster-management/ec2-instances.md)
* [Spot instances](cluster-management/spot-instances.md)
* [Update](cluster-management/update.md)
* [Uninstall](cluster-management/uninstall.md)

## Miscellaneous

* [CLI commands](miscellaneous/cli.md)
* [Python client](miscellaneous/python-client.md)
* [Environments](miscellaneous/environments.md)
* [Architecture diagram](miscellaneous/architecture.md)
* [Security](miscellaneous/security.md)
* [Telemetry](miscellaneous/telemetry.md)

## Troubleshooting

* [API is stuck updating](troubleshooting/stuck-updating.md)
* [404/503 API responses](troubleshooting/api-request-errors.md)
* [NVIDIA runtime not found](troubleshooting/nvidia-container-runtime-not-found.md)
* [TF session in predict()](troubleshooting/tf-session-in-predict.md)
* [Serving-side batching errors](troubleshooting/server-side-batching-errors.md)
* [Cluster down failures](troubleshooting/cluster-down.md)

## Guides

* [Exporting models](guides/exporting.md)
* [Multi-model endpoints](guides/multi-model.md)
* [View API metrics](guides/metrics.md)
* [Running in production](guides/production.md)
* [Low-cost clusters](guides/low-cost-clusters.md)
* [Set up a custom domain](guides/custom-domain.md)
* [Set up VPC peering](guides/vpc-peering.md)
* [SSH into worker instance](guides/ssh-instance.md)
* [Single node deployment](guides/single-node-deployment.md)
* [Set up kubectl](guides/kubectl-setup.md)
* [Docker Hub rate limiting](guides/docker-hub-rate-limiting.md)
* [Private docker registry](guides/private-docker.md)
* [Set up REST API Gateway](guides/rest-api-gateway.md)
* [Install CLI on Windows](guides/windows-cli.md)

## Contributing

* [Development](contributing/development.md)
