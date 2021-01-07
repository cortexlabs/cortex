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

# USAGE: python ./dev/deploy_test.py <env_name>
# e.g.: python ./dev/deploy_test.py aws

import os
import cortex
import sys
import requests

cx = cortex.client(sys.argv[1])
api_config = {
    "name": "text-generator",
    "kind": "RealtimeAPI",
}


class PythonPredictor:
    def __init__(self, config):
        from transformers import pipeline

        self.model = pipeline(task="text-generation")

    def predict(self, payload):
        return self.model(payload["text"])[0]


api = cx.create_api(
    api_config,
    predictor=PythonPredictor,
    requirements=["torch", "transformers"],
    wait=True,
)

response = requests.post(
    api["endpoint"],
    json={"text": "machine learning is great because"},
)

print(response.status_code)
print(response.text)

cx.delete_api(api_config["name"])
