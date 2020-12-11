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

import e2e


def pytest_addoption(parser):
    parser.addoption("--aws-env", action="store", default=None)
    parser.addoption("--aws-config", action="store", default=None)


def pytest_configure(config):
    aws_config = config.getoption("--aws-config")
    if aws_config:
        e2e.create_cluster(aws_config)


def pytest_unconfigure(config):
    aws_config = config.getoption("--aws-config")
    if aws_config:
        e2e.delete_cluster(aws_config)
