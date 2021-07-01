#!/usr/bin/env bash

# usage: build-neuron-tf-serving.sh [REGISTRY] [--skip-push]
#   REGISTRY defaults to $CORTEX_DEV_DEFAULT_IMAGE_REGISTRY; e.g. 764403040460.dkr.ecr.us-west-2.amazonaws.com/cortexlabs or quay.io/cortexlabs-test

image_name="neuron-tf-serving"

aws ecr get-login-password --region us-east-1 | docker login --username AWS --password-stdin 763104351884.dkr.ecr.us-east-1.amazonaws.com

"$(dirname "${BASH_SOURCE[0]}")"/../../../utils/build.sh $(realpath "${BASH_SOURCE[0]}") "$image_name" "$@"
