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

import pathlib
import pkg_resources
from setuptools import setup, find_packages

with pathlib.Path("cortex_internal.requirements.txt").open() as requirements_txt:
    install_requires = [
        str(requirement) for requirement in pkg_resources.parse_requirements(requirements_txt)
    ]

setup(
    name="cortex-internal",
    version="0.31.1",  # CORTEX_VERSION
    description="Internal package for Cortex containers",
    author="cortex.dev",
    author_email="dev@cortex.dev",
    license="Apache License 2.0",
    url="https:/github.com/cortexlabs/cortex",
    setup_requires=(["setuptools", "wheel"]),
    packages=find_packages(),
    install_requires=install_requires,
    python_requires=">=3.6",
    classifiers=[
        "Operating System :: MacOS",
        "Operating System :: POSIX :: Linux",
        "Programming Language :: Python :: 3.6",
        "Intended Audience :: Developers",
    ],
    project_urls={
        "Bug Reports": "https://github.com/cortexlabs/cortex/issues",
        "Chat with us": "https://gitter.im/cortexlabs/cortex",
        "Documentation": "https://docs.cortex.dev",
        "Source Code": "https://github.com/cortexlabs/cortex",
    },
)
