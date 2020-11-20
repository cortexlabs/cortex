# Docker Hub rate limiting

_WARNING: you are on the master branch, please refer to the docs on the branch that matches your `cortex version`_

Docker Hub's [newly enforced rate-limiting policy](https://www.docker.com/increase-rate-limits) can negatively impact your cluster. This is much likelier to be an issue if you've set `subnet_visibility: private` in your cluster configuration file, since with private subnets, all requests from all nodes are routed through the NAT Gateway, and will therefore have the same IP address (docker imposes the rate limit per IP address). If you haven't specified `subnet_visibility` or have set `subnet_visibility: public`, this is less likely to be an issue for you, since each instance will have its own IP address.

We are actively working on a long term resolution to this problem. In the meantime, there are two ways to avoid this issue:

## Paid Docker Hub subscription

One option is to pay for the Docker Hub subscription to remove the limit on the number of image pulls. Docker Hub's updated pricing model allows unlimited pulls on a _Pro_ subscription for individuals as described [here](https://www.docker.com/pricing).

By default, the Cortex cluster pulls the images as an anonymous user. Follow [this guide](private-docker.md) to configure your Cortex cluster to pull the images as an authenticated user.

## Push to AWS ECR (Elastic Container Registry)

You can configure the Cortex cluster to use images from a different registry. A good choice is ECR on AWS. When an ECR repository resides in the same region as your Cortex cluster, there are no costs incurred when pulling images.

### Step 1

Make sure you have the [aws](https://docs.aws.amazon.com/cli/latest/userguide/install-cliv1.html) and [docker](https://docs.docker.com/get-docker/) CLIs installed.

### Step 2

Export the `AWS_ACCESS_KEY_ID` and `AWS_SECRET_ACCESS_KEY` environment variables in your current shell, or run `aws configure`. These credentials must have access to push to ECR.

### Step 3

Choose a region for your cluster and ECR repositories. In this guide, we'll assume the region is `us-west-2`.

Also, take note of your AWS account ID. The account ID can be found in the _My Account_ section of your AWS console.

### Step 4

You can use the script below to push the images from Docker Hub to your ECR registry. Make sure to update the `ecr_region`, `aws_account_id`, and `cortex_version` variables at the top of the file. Copy-paste the contents into a new file (e.g. `ecr.sh`), and then run `chmod +x ecr.sh`, followed by `./ecr.sh`. It is recommended to run this from an EC2 instance in the same region as your ECR repository, since it will be much faster.

```bash
#!/bin/bash
set -euo pipefail

# user set variables
ecr_region="us-west-2"
aws_account_id="620970939130"  # example account ID
cortex_version="0.22.1"

source_registry="docker.io/cortexlabs"
destination_registry="${aws_account_id}.dkr.ecr.${ecr_region}.amazonaws.com/cortexlabs"

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
    aws ecr create-repository --repository-name=cortexlabs/$image --region=$ecr_region || true
done

# pull the images from Docker Hub and push them to ECR
for image in "${images[@]}"; do
    if [ "$image" = "python-predictor-gpu-slim" ]; then
        for extra_tag in "${extra_tags_for_slim_python_predictor[@]}"; do
            docker image pull "$source_registry/$image:$cortex_version-$extra_tag"
            docker image tag "$source_registry/$image:$cortex_version-$extra_tag" "$destination_registry/$image:$cortex_version-$extra_tag"
            docker image push "$destination_registry/$image:$cortex_version-$extra_tag"
            echo
        done
    else
      docker image pull "$source_registry/$image:$cortex_version"
      docker image tag "$source_registry/$image:$cortex_version" "$destination_registry/$image:$cortex_version"
      docker image push "$destination_registry/$image:$cortex_version"
      echo
    fi
done

echo "###############################################"
echo
echo "add the following images to your cortex cluster configuration file (e.g. cluster.yaml):"
echo "-----------------------------------------------"
for cluster_image in "${cluster_images[@]}"; do
    echo "image_$cluster_image: $destination_registry/$cluster_image:$cortex_version"
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

The first list of images that were printed (the cluster images) can be directly copy-pasted in your [cluster configuration file](../cluster-management/config.md) before spinning up your cluster.

The second list of images that were printed (the API images) can be used in your [API configuration files](../deployments/realtime-api/api-configuration.md). The images are specified in `predictor.image` (and `predictor.tensorflow_serving_image` for APIs with `kind: tensorflow`). Be advised that by default, the Docker Hub images are used for your predictors, so you will need to specify your ECR image paths for all of your APIs.

## Step 6

Spin up your Cortex cluster using your updated cluster configuration file (e.g. `cortex cluster up --config cluster.yaml`).

## Cleanup

You can delete your ECR images from the [AWS ECR dashboard](https://console.aws.amazon.com/ecr/repositories) (set your region in the upper right corner). Make sure all of your Cortex clusters have been deleted before deleting any ECR images.
