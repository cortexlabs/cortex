#!/bin/bash
# Copyright 2021 Cortex Labs, Inc.
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

set -euo pipefail

# usage: ./dev/export_images.sh <region> <aws account id>
# e.g. ./dev/export_images.sh us-east-1 123456789

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")"/.. >/dev/null && pwd)"

# CORTEX_VERSION
cortex_version=master

# user set variables
ecr_region=$1
aws_account_id=$2

source_registry=quay.io/cortexlabs  # this can also be docker.io/cortexlabs

destination_ecr_prefix="cortexlabs"
destination_registry="${aws_account_id}.dkr.ecr.${ecr_region}.amazonaws.com/${destination_ecr_prefix}"

need_login=true
if [[ -f $HOME/.docker/config.json && $(cat $HOME/.docker/config.json | grep "ecr-login" | wc -l) -ne 0 ]]; then
	need_login=false
	echo "skipping docker login because you are using ecr-login with Amazon ECR Docker Credential Helper"
fi

if [[ $need_login = true ]]; then
	aws ecr get-login-password --region $ecr_region | docker login --username AWS --password-stdin $destination_registry
fi

source $ROOT/build/images.sh

# create the image repositories
for image in "${all_images[@]}"; do
	repository_name=$destination_ecr_prefix/$image
	if [[ $(aws ecr describe-repositories | grep repositoryName | grep $image | wc -l) -ne 0 ]]; then
		echo "repository '$repository_name' already exists. skip to next"
	else
		aws ecr create-repository --repository-name=$repository_name --region=$ecr_region | cat
	fi
done
echo

# pull the images from source registry and push them to ECR
for image in "${all_images[@]}"; do
    echo "copying $image:$cortex_version from $source_registry to $destination_registry"
    skopeo copy --src-no-creds "docker://$source_registry/$image:$cortex_version" "docker://$destination_registry/$image:$cortex_version"
    echo
done
echo "done ✓"
