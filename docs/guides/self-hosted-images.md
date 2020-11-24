# Self-hosted Docker images

Self-hosted Docker images can be useful for reducing the ingress costs, for accelerating image pulls, or for eliminating the dependency on Cortex's public container registry.

In this guide, we'll use [ECR](https://aws.amazon.com/ecr/) as the destination container registry. When an ECR repository resides in the same region as your Cortex cluster, there are no costs incurred when pulling images.

## Step 1

Make sure you have the [aws](https://docs.aws.amazon.com/cli/latest/userguide/install-cliv1.html), [docker](https://docs.docker.com/get-docker/) and [skopeo](https://github.com/containers/skopeo/blob/master/install.md) utilities installed.

## Step 2

Export the `AWS_ACCESS_KEY_ID` and `AWS_SECRET_ACCESS_KEY` environment variables in your current shell, or run `aws configure`. These credentials must have access to push to ECR.

## Step 3

Choose a region for your cluster and ECR repositories. In this guide, we'll assume the region is `us-west-2`.

Also, take note of your AWS account ID. The account ID can be found in the _My Account_ section of your AWS console.

## Step 4

You can use the script below to push the Cortex images to your ECR registry. Make sure to update the `ecr_region`, `aws_account_id`, and `cortex_version` variables at the top of the file. Copy-paste the contents into a new file (e.g. `ecr.sh`), and then run `chmod +x ecr.sh`, followed by `./ecr.sh`. It is recommended to run this from an EC2 instance in the same region as your ECR repository, since it will be much faster.

```bash
#!/bin/bash
set -euo pipefail

# user set variables
ecr_region="us-west-2"
aws_account_id="620970939130"  # example account ID
cortex_version="0.22.1"

source_registry="quay.io/cortexlabs"
destination_ecr_prefix="cortexlabs"

destination_registry="${aws_account_id}.dkr.ecr.${ecr_region}.amazonaws.com/${destination_ecr_prefix}"
aws ecr get-login-password --region $ecr_region | docker login --username AWS --password-stdin $destination_registry

# images for the cluster
cluster_images=(
  "manager"
  "request-monitor"
  "downloader"
  "operator"
  "cluster-autoscaler"
  "metrics-server"
  "inferentia"
  "neuron-rtd"
  "nvidia"
  "fluentd"
  "statsd"
  "istio-proxy"
  "istio-pilot"
)

# images for the APIs (you may delete any images that your APIs don't use)
api_images=(
  "python-predictor-cpu"
  "python-predictor-gpu"
  "python-predictor-inf"
  "tensorflow-serving-cpu"
  "tensorflow-serving-gpu"
  "tensorflow-serving-inf"
  "tensorflow-predictor"
  "onnx-predictor-cpu"
  "onnx-predictor-gpu"
  "python-predictor-cpu-slim"
  "python-predictor-gpu-slim"
  "python-predictor-inf-slim"
  "tensorflow-predictor-slim"
  "onnx-predictor-cpu-slim"
  "onnx-predictor-gpu-slim"
)
images=( "${cluster_images[@]}" "${api_images[@]}" )

extra_tags_for_slim_python_predictor=(
    "cuda10.0"
    "cuda10.1"
    "cuda10.2"
    "cuda11.0"
)

# create the image repositories
for image in "${images[@]}"; do
    aws ecr create-repository --repository-name=$destination_ecr_prefix/$image --region=$ecr_region || true
done
echo

# pull the images from Docker Hub and push them to ECR
for image in "${images[@]}"; do
    if [ "$image" = "python-predictor-gpu-slim" ]; then
        for extra_tag in "${extra_tags_for_slim_python_predictor[@]}"; do
            echo "copying $image:$cortex_version-$extra_tag from $source_registry to $destination_registry"
            skopeo copy --src-no-creds "docker://$source_registry/$image:$cortex_version-$extra_tag" "docker://$destination_registry/$image:$cortex_version-$extra_tag"
            echo
        done
    else
        echo "copying $image:$cortex_version from $source_registry to $destination_registry"
        skopeo copy --src-no-creds "docker://$source_registry/$image:$cortex_version" "docker://$destination_registry/$image:$cortex_version"
        echo
    fi
done

echo "###############################################"
echo
echo "add the following images to your cortex cluster configuration file (e.g. cluster.yaml):"
echo "-----------------------------------------------"
for cluster_image in "${cluster_images[@]}"; do
    cluster_image_underscores=${cluster_image//-/_}
    echo "image_$cluster_image_underscores: $destination_registry/$cluster_image:$cortex_version"
done
echo -e "-----------------------------------------------\n"

echo "use the following images in your API configuration files (e.g. cortex.yaml):"
echo "-----------------------------------------------"
for api_image in "${api_images[@]}"; do
    if [ "$api_image" = "python-predictor-gpu-slim" ]; then
        for extra_tag in "${extra_tags_for_slim_python_predictor[@]}"; do
            echo "$destination_registry/$api_image:$cortex_version-$extra_tag"
        done
    else
      echo "$destination_registry/$api_image:$cortex_version"
    fi
done
echo "-----------------------------------------------"
```

The first list of images that were printed (the cluster images) can be directly copy-pasted in your [cluster configuration file](../aws/install.md) before spinning up your cluster.

The second list of images that were printed (the API images) can be used in your [API configuration files](../deployments/realtime-api/api-configuration.md). The image paths are specified in `predictor.image` (and `predictor.tensorflow_serving_image` for APIs with `kind: tensorflow`). Be advised that by default, the public images offered by Cortex are used for your predictors, so you will need to specify your ECR image paths for all of your APIs.

## Step 5

Spin up your Cortex cluster using your updated cluster configuration file (e.g. `cortex cluster up --config cluster.yaml`).

## Cleanup

You can delete your ECR images from the [AWS ECR dashboard](https://console.aws.amazon.com/ecr/repositories) (set your region in the upper right corner). Make sure all of your Cortex clusters have been deleted before deleting any ECR images.
