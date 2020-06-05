# Copyright 2020 Cortex Labs, Inc.
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


class Model:
    def __init__(self, name=None, model=None, signature_key=None, base_path=None):
        self.name = name
        self.model = model
        self.signature_key = signature_key
        self.base_path = base_path


def get_model_signature_map(models):
    signature_keys = {}
    for model in models:
        signature_keys[model.name] = model.signature_key

    return signature_keys


def get_model_names(models):
    return [model.name for model in models]
