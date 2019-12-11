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

from setuptools import setup, find_packages

setup(
    name="cortex",
    version="0.11.1",  # CORTEX_VERSION
    description="",
    author="Cortex Labs",
    author_email="dev@cortexlabs.com",
    install_requires=["dill>=0.3.1.1", "requests>=2.20.0", "msgpack>=0.6.0"],
    setup_requires=["setuptools"],
    packages=find_packages(),
)
