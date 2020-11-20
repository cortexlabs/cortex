# Docker rate-limiting

_WARNING: you are on the master branch, please refer to the docs on the branch that matches your `cortex version`_

Docker Hub's [annoucement](https://www.docker.com/increase-rate-limits) of rate-limiting the docker pulls for anonymous and authenticated users out has had a negative effect on the usability/reliability of using images served from this registry. Since spinning up a Cortex cluster for production use assumes the image pull operation to be done thousands of times, an alternative has to be found, to ensure that images can still be pulled any number of times.

## Paid subscription of Docker Hub

One option is to pay for the Docker Hub subscription to remove the limit on the number of pulls.

The updated pricing model on Docker Hub allows unlimited pulls to be done on a _Pro_ subscription for individuals as described [here](https://www.docker.com/pricing). Assuming the Cortex user is okay with this cost, the Cortex cluster can be configured to use the Docker credentials that grant access to an unlimited number of pulls from Docker Hub.

By default, the Cortex cluster pulls the images as an anonymous user. Follow [this guide](private-docker.md) to get your Cortex cluster to pull the images as an authenticated user (which has access to an unlimited number of pulls).

## Push to AWS ECR (Elastic Container Registry)

You can configure the Cortex cluster to use images from a different registry. A good alternative is ECR. When an ECR repository is within the same region as a Cortex cluster, there are no costs incurred on the ingress traffic.

### Step 1

Make sure you have the [aws](https://docs.aws.amazon.com/cli/latest/userguide/install-cliv1.html) and [docker](https://docs.docker.com/get-docker/) utilities installed.

### Step 2

Export the `AWS_ACCESS_KEY_ID` and `AWS_SECRET_ACCESS_KEY` variables in your current shell. These should be the same credentials that you use to spin up a cluster. These will be used by the cluster to pull the ECR images.

### Step 3

Decide on the region of your cluster. In this guide, let's assume the region for our cluster will be `us-west-2`. This also means our ECR repositories have to be in the `us-west-2` region.

Also, make a note of the AWS user account ID. The user account ID can be found in the _My Account_ section of your AWS console.

### Step 4

Push the images from Docker Hub to your ECR registry. Change the `ecr_region` to your desired region (in our case it's `us-west-2`). Set the right account ID for the `aws_account_id` variable. And also set the desired [version of Cortex](https://github.com/cortexlabs/cortex/releases) with `cortex_version` to use.

Now, run the following script - it runs on both a Mac or a Linux OS.

```bash
#!/bin/bash
set -euo pipefail

# user set variables
ecr_region="us-west-2"
aws_account_id="620970939130" # example account ID
cortex_version="0.22.0"

registry_name="cortexlabs"
source_registry="docker.io/$registry_name"
destination_registry="$aws_account_id.dkr.ecr.$ecr_region.amazonaws.com/$registry_name"

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

# images for the APIs
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
)
images=( "${cluster_images[@]}" "${api_images[@]}" )

extra_tags_for_slim_python_predictor=(
    "cuda10.0-cudnn7"
    "cuda10.1-cudnn7"
    "cuda10.1-cudnn8"
    "cuda10.2-cudnn7"
    "cuda10.2-cudnn8"
    "cuda11.0-cudnn8"
    "cuda11.1-cudnn8"
)

# create the image repositories
for image in "${images[@]}"; do
    aws ecr create-repository --repository-name=cortexlabs/$image --region=$ecr_region || true
done

# pull the images from the source registry
# and then push them to the destination registry (to ECR)
aws ecr get-login-password --region $ecr_region | docker login --username AWS --password-stdin $destination_registry
for image in "${images[@]}"; do
    docker image pull "$source_registry/$image:$cortex_version"
    docker image tag "$source_registry/$image:$cortex_version" "$destination_registry/$image:$cortex_version"
    docker image push "$destination_registry/$image:$cortex_version"
    if [ "$api_image" = "python-predictor-gpu-slim" ]; then
        for extra_tag in "${extra_tags_for_slim_python_predictor[@]}"; do
            docker image pull "$source_registry/$image:$cortex_version-$extra_tag"
            docker image tag "$source_registry/$image:$cortex_version" "$destination_registry/$image:$cortex_version-$extra_tag"
            docker image push "$destination_registry/$image:$cortex_version-$extra_tag"
        done
    fi
done

echo "add the following images to your cortex cluster config"
echo "-----------------------------------------------"
for cluster_image in "${cluster_images[@]}"; do
    echo "image_$cluster_image: $destination_registry/$cluster_image:$cortex_version"
done
echo -e "-----------------------------------------------\n"

echo "use the following images for your API specs"
echo "-----------------------------------------------"
for api_image in "${api_images[@]}"; do
    echo "$destination_registry/$api_image:$cortex_version"
    if [ "$api_image" = "python-predictor-gpu-slim" ]; then
        for extra_tag in "${extra_tags_for_slim_python_predictor[@]}"; do
            echo "$destination_registry/$api_image:$cortex_version-$extra_tag"
        done
    fi
done
echo "-----------------------------------------------"
```

The first list of images that were printed for the cluster config can be directly copy-pasted in your [`cluster.yaml` config](../cluster-management/config.md).

The API images can then be used for your API specs. The images would have to be set for the `predictor.image`/`predictor.tensorflow_serving_image` fields as described in the [API configuration template](../deployments/realtime-api/api-configuration.md) - also applicable to the [batch APIs](../deployments/batch-api/api-configuration.md). Be advised that by default, the Docker Hub images are used for your predictors, so you will need to specify the ECR versions.

## Step 5 - Optional

Push the [slim predictor images](../deployments/system-packages.md#custom-docker-image) as well. To the `api_images` array from the previous step, also add the following images:

```text
python-predictor-cpu-slim
python-predictor-inf-slim
tensorflow-predictor-slim
onnx-predictor-cpu-slim
onnx-predictor-gpu-slim
```

## Step 6

Spin up your Cortex cluster using the ECR-configured `cluster.yaml` config.