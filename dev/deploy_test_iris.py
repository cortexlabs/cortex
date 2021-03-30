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

import cortex

cx = cortex.client()

api_spec = {
    "name": "iris-classifier",
    "kind": "RealtimeAPI",
    "predictor": {
        "type": "python",
        "path": "predictor.py",
        "config": {
            "model": "s3://cortex-examples/pytorch/iris-classifier/weights.pth",
        },
    },
}

print(cx.create_api(api_spec, project_dir="test/apis/pytorch/iris-classifier"))

# cx.delete_api("iris-classifier")
