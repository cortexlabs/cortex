# Table of contents

* [Build machine learning APIs](../README.md)
* [Install](cluster-management/install.md)
* [Tutorial](../examples/pytorch/text-generator/README.md)
* [GitHub](https://github.com/cortexlabs/cortex)
* [Examples](https://github.com/cortexlabs/cortex/tree/master/examples)  <!-- CORTEX_VERSION_MINOR -->
* [Chat with us](https://gitter.im/cortexlabs/cortex)
* [Email us](mailto:hello@cortex.dev)
* [We're hiring](https://angel.co/cortex-labs-inc/jobs)

## Deployments

* [Sync API](deployments/syncapi.md)
  * [Predictor implementation](deployments/syncapi/predictors.md)
  * [API configuration](deployments/syncapi/api-configuration.md)
  * [API deployment](deployments/syncapi/deployment.md)
  * [API statuses](deployments/syncapi/statuses.md)
  * [Parallelism](deployments/syncapi/parallelism.md)
  * [Autoscaling](deployments/syncapi/autoscaling.md)
  * [Prediction monitoring](deployments/syncapi/prediction-monitoring.md)
  * [Tutorial](../examples/pytorch/text-generator/README.md)
  * [API Splitter](deployments/syncapi/apisplitter.md)
* [Batch API](deployments/batchapi.md)
  * [Predictor implementation](deployments/batchapi/predictors.md)
  * [API configuration](deployments/batchapi/api-configuration.md)
  * [API deployment](deployments/batchapi/deployment.md)
  * [Endpoints](deployments/batchapi/endpoints.md)
  * [Job statuses](deployments/batchapi/statuses.md)
  * [Tutorial](../examples/batch/image-classifier/README.md)

## Advanced

* [Compute](deployments/compute.md)
* [Using GPUs](deployments/gpus.md)
* [Using Inferentia](deployments/inferentia.md)
* [Python packages](deployments/python-packages.md)
* [System packages](deployments/system-packages.md)

## Cluster management

* [Cluster configuration](cluster-management/config.md)
* [AWS credentials](cluster-management/aws-credentials.md)
* [EC2 instances](cluster-management/ec2-instances.md)
* [Spot instances](cluster-management/spot-instances.md)
* [Update](cluster-management/update.md)
* [Uninstall](cluster-management/uninstall.md)

## Miscellaneous

* [CLI commands](miscellaneous/cli.md)
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

## Guides

* [Exporting models](guides/exporting.md)
* [Multi-model endpoints](guides/multi-model.md)
* [View API metrics](guides/metrics.md)
* [Set up a custom domain](guides/custom-domain.md)
* [Set up VPC peering](guides/vpc-peering.md)
* [SSH into worker instance](guides/ssh-instance.md)
* [Single node deployment](guides/single-node-deployment.md)

## Contributing

* [Development](contributing/development.md)
