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

from pathlib import Path

from setuptools import setup, find_packages

root = Path(__file__).parent.absolute()
cortex_client_dir = root.parent.parent / "pkg" / "cortex" / "client"

if not cortex_client_dir.exists():
    raise ModuleNotFoundError(f"cortex client not found in {cortex_client_dir}")

setup(
    name="e2e",
    version="0.31.1",  # CORTEX_VERSION
    packages=find_packages(exclude=["tests"]),
    url="https://github.com/cortexlabs/cortex",
    license="Apache License 2.0",
    python_requires=">=3.6",
    install_requires=[
        "requests==2.24.0",
        "jsonschema==3.2.0",
        "pytest==6.1.*",
        "python-dotenv==0.15.0",
        "pyyaml>=5.3.1",
        "cortex",
    ],
    dependency_links=[f"file://{cortex_client_dir}#egg=cortex"],
    author="Cortex Labs",
    author_email="dev@cortex.dev",
    description="Cortex E2E tests package",
)
