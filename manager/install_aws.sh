#!/bin/bash

# Copyright 2019 Cortex Labs, Inc.
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#     http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

set -e

function setup_bucket() {
  if ! aws s3api head-bucket --bucket $CORTEX_BUCKET --output json 2>/dev/null; then
    if aws s3 ls "s3://$CORTEX_BUCKET" --output json 2>&1 | grep -q 'NoSuchBucket'; then
      echo -e "\n✓ Creating S3 bucket: $CORTEX_BUCKET"
      aws s3api create-bucket --bucket $CORTEX_BUCKET \
                              --region $CORTEX_REGION \
                              --create-bucket-configuration LocationConstraint=$CORTEX_REGION \
                              >/dev/null
    else
      echo -e "\nA bucket named \"${CORTEX_BUCKET}\" already exists, but you do not have access to it"
      exit 1
    fi
  else
    echo -e "\n✓ Using existing S3 bucket: $CORTEX_BUCKET"
  fi
}

function setup_cloudwatch_logs() {
  if ! aws logs list-tags-log-group --log-group-name $CORTEX_LOG_GROUP --region $CORTEX_REGION --output json 2>&1 | grep -q "\"tags\":"; then
    echo -e "\n✓ Creating CloudWatch log group: $CORTEX_LOG_GROUP"
    aws logs create-log-group --log-group-name $CORTEX_LOG_GROUP --region $CORTEX_REGION
  else
    echo -e "\n✓ Using existing CloudWatch log group: $CORTEX_LOG_GROUP"
  fi
}

function setup_configmap() {
  kubectl -n=$CORTEX_NAMESPACE create configmap 'cortex-config' \
    --from-literal='LOG_GROUP'=$CORTEX_LOG_GROUP \
    --from-literal='BUCKET'=$CORTEX_BUCKET \
    --from-literal='REGION'=$CORTEX_REGION \
    --from-literal='NAMESPACE'=$CORTEX_NAMESPACE \
    --from-literal='IMAGE_OPERATOR'=$CORTEX_IMAGE_OPERATOR \
    --from-literal='IMAGE_SPARK'=$CORTEX_IMAGE_SPARK \
    --from-literal='IMAGE_TF_TRAIN'=$CORTEX_IMAGE_TF_TRAIN \
    --from-literal='IMAGE_TF_SERVE'=$CORTEX_IMAGE_TF_SERVE \
    --from-literal='IMAGE_TF_API'=$CORTEX_IMAGE_TF_API \
    --from-literal='IMAGE_PYTHON_PACKAGER'=$CORTEX_IMAGE_PYTHON_PACKAGER \
    --from-literal='IMAGE_TF_TRAIN_GPU'=$CORTEX_IMAGE_TF_TRAIN_GPU \
    --from-literal='IMAGE_TF_SERVE_GPU'=$CORTEX_IMAGE_TF_SERVE_GPU \
    --from-literal='ENABLE_TELEMETRY'=$CORTEX_ENABLE_TELEMETRY \
    -o yaml --dry-run | kubectl apply -f - >/dev/null
}

function setup_secrets() {
  kubectl -n=$CORTEX_NAMESPACE create secret generic 'aws-credentials' \
    --from-literal='AWS_ACCESS_KEY_ID'=$AWS_ACCESS_KEY_ID \
    --from-literal='AWS_SECRET_ACCESS_KEY'=$AWS_SECRET_ACCESS_KEY \
    -o yaml --dry-run | kubectl apply -f - >/dev/null
}

echo -e "\nSpinning up a cluster ... (this will about 15 minutes)"

echo
eksctl create cluster --name=$CORTEX_CLUSTER --asg-access --node-type=$CORTEX_NODE_TYPE --nodes-min=$CORTEX_NODES_MIN --nodes-max=$CORTEX_NODES_MAX

echo -e "\n✓ Spun up a cluster"

setup_bucket
setup_cloudwatch_logs

setup_configmap
setup_secrets
