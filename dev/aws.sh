#!/bin/bash
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")"/.. >/dev/null && pwd)"

source $ROOT/dev/config/build.sh
source $ROOT/dev/config/k8s.sh
source $ROOT/dev/util.sh

if [ "$1" = "delete-cache" ]; then
  aws s3 rm --recursive --quiet s3://$K8S_BUCKET/apps
  aws s3 rm --recursive --quiet s3://$K8S_BUCKET/contexts
  aws s3 rm --recursive --quiet s3://$K8S_BUCKET/model_implementations
  aws s3 rm --recursive --quiet s3://$K8S_BUCKET/transformations
else
  echo "Command $1 not found"
fi
