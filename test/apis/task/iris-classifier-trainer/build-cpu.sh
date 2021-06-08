#!/usr/bin/env bash

# usage: build-cpu.sh [REGISTRY] [--skip-push]
#   REGISTRY defaults to $CORTEX_DEV_DEFAULT_IMAGE_REGISTRY; e.g. 764403040460.dkr.ecr.us-west-2.amazonaws.com/cortexlabs or quay.io/cortexlabs-test

image_name="task-iris-classifier-trainer-cpu"

"$(dirname "${BASH_SOURCE[0]}")"/../../../utils/build.sh $(realpath "${BASH_SOURCE[0]}") "$image_name" "$@"
