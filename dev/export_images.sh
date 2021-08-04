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
cortex_version=0.40.0

# user set variables
ecr_region=$1
aws_account_id=$2

source_registry=quay.io/cortexlabs  # this can also be docker.io/cortexlabs

destination_ecr_prefix="cortexlabs"
destination_registry="${aws_account_id}.dkr.ecr.${ecr_region}.amazonaws.com/${destination_ecr_prefix}"
aws ecr get-login-password --region $ecr_region | docker login --username AWS --password-stdin $destination_registry

source build/images.sh

# create the image repositories
for image in "${all_images[@]}"; do
    aws ecr create-repository --repository-name=$destination_ecr_prefix/$image --region=$ecr_region || true
done
echo

# pull the images from source registry and push them to ECR
for image in "${all_images[@]}"; do
    echo "copying $image:$cortex_version from $source_registry to $destination_registry"
    skopeo copy --src-no-creds "docker://$source_registry/$image:$cortex_version" "docker://$destination_registry/$image:$cortex_version"
    echo
done
echo "done ✓"
