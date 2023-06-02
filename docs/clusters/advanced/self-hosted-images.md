# Self-hosted Docker images

Self-hosting the Cortex cluster's system Docker images can be useful for reducing the ingress costs, for accelerating image pulls, or for eliminating the dependency on Cortex's public container registry.

In this guide, we'll use [ECR](https://aws.amazon.com/ecr/) as the destination container registry. When an ECR repository resides in the same region as your Cortex cluster, there are no costs incurred when pulling images.

## Step 1

Make sure you have the [aws](https://docs.aws.amazon.com/cli/latest/userguide/install-cliv1.html), [docker](https://docs.docker.com/get-docker/), and [skopeo](https://github.com/containers/skopeo/blob/master/install.md) utilities installed.

## Step 2

Export the `AWS_ACCESS_KEY_ID` and `AWS_SECRET_ACCESS_KEY` environment variables in your current shell, or run `aws configure`. These credentials must have access to push to ECR.

## Step 3

Clone the Cortex repo using the release tag corresponding to your version (which you can check by running `cortex version`):

<!-- CORTEX_VERSION_README -->

```bash
export CORTEX_VERSION=0.42.2
git clone --depth 1 --branch v$CORTEX_VERSION https://github.com/cortexlabs/cortex.git
```

## Step 4

Run the script below to export images to ECR in the same region and account as your cluster.

The script will automatically create ECR Repositories with prefix `cortexlabs` if they don't already exist.

Feel free to modify the script if you would like to export the images to a different registry such as a private docker hub.

```bash
./cortex/dev/export_images.sh <AWS_REGION> <AWS_ACCOUNT_ID>
```

You can now configure Cortex to use your images when creating a cluster (see [here](../management/create.md) for instructions).

## Cleanup

You can delete your ECR images from the [AWS ECR dashboard](https://console.aws.amazon.com/ecr/repositories) (set your region in the upper right corner). Make sure all of your Cortex clusters have been deleted before deleting any ECR images.
