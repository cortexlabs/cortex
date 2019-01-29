#!/bin/bash
set -euo pipefail

DIR=$1
IMAGE=$2
CORTEX_VERSION=master

docker build . -f $DIR/Dockerfile -t cortexlabs/$IMAGE \
                                  -t cortexlabs/$IMAGE:$CORTEX_VERSION

echo "$DOCKER_PASSWORD" | docker login -u "$DOCKER_USERNAME" --password-stdin

docker push cortexlabs/$IMAGE:$CORTEX_VERSION
