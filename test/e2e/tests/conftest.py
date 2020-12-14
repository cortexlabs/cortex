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


def pytest_addoption(parser):
    parser.addoption(
        "--aws-env",
        action="store",
        default=None,
        help="set cortex AWS environment, to test on an existing AWS cluster",
    )
    parser.addoption(
        "--aws-config",
        action="store",
        default=None,
        help="set cortex AWS cluster config, to test on a new AWS cluster",
    )
    parser.addoption(
        "--gcp-env",
        action="store",
        default=None,
        help="set cortex GCP environment, to test on an existing GCP cluster",
    )
    parser.addoption(
        "--gcp-config",
        action="store",
        default=None,
        help="set cortex GCP cluster config, to test on a new GCP cluster",
    )
