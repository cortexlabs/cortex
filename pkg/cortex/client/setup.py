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


from setuptools import setup, find_packages
from setuptools.command.install import install

import sys


class InstallBinary(install):
    def run(self):
        install.run(self)

        import requests
        from pathlib import Path
        import sys
        import os
        import stat
        import shutil

        dest_dir = os.path.join(self.install_lib, "cortex", "binary")

        zip_file_path = os.path.join(dest_dir, "cli.zip")
        cli_file_path = os.path.join(dest_dir, "cli")

        if not os.path.exists(cli_file_path):
            platform = sys.platform

            # recommended way to check platform: https://docs.python.org/3/library/sys.html#sys.platform
            if sys.platform.startswith("darwin"):
                platform = "darwin"
            if sys.platform.startswith("linux"):
                platform = "linux"

            if platform != "darwin" and platform != "linux":
                raise Exception(
                    f"platform {platform} is not supported; cortex is only supported on mac and linux"
                )

            cortex_version = self.config_vars["dist_version"]

            if "dev" in cortex_version:
                cortex_version = "master"

            download_url = f"https://s3-us-west-2.amazonaws.com/get-cortex/{cortex_version}/cli/{platform}/cortex.zip"

            print("downloading cortex cli...")
            with requests.get(download_url, stream=True) as r:
                with open(zip_file_path, "wb") as f:
                    shutil.copyfileobj(r.raw, f)

            zip_dir = os.path.join(dest_dir, "cli_dir")

            print("unzipping cortex cli...")
            shutil.unpack_archive(zip_file_path, zip_dir)
            shutil.move(os.path.join(zip_dir, "cortex"), cli_file_path)
            shutil.rmtree(zip_dir)
            os.remove(zip_file_path)

            f = Path(cli_file_path)
            f.chmod(f.stat().st_mode | stat.S_IEXEC)


with open("README.md") as f:
    long_description = f.read()

setup(
    name="cortex",
    version="0.28.0",  # CORTEX_VERSION
    description="Run inference at scale",
    author="cortex.dev",
    author_email="dev@cortex.dev",
    license="Apache License 2.0",
    long_description_content_type="text/markdown",
    long_description=long_description,
    url="https://www.cortex.dev",
    setup_requires=(["setuptools", "requests", "wheel"]),
    packages=find_packages(),
    package_data={"cortex.binary": ["cli"]},
    entry_points={
        "console_scripts": [
            "cortex = cortex.binary:run",
        ],
    },
    install_requires=(
        [
            "importlib-resources; python_version < '3.7'",
            "pyyaml>=5.3.0",
            "dill==0.3.2",  # lines up with dill package version used in cortex serving code
        ]
    ),
    python_requires=">=3.6",
    cmdclass={
        "install": InstallBinary,
    },
    classifiers=[
        "Operating System :: MacOS",
        "Operating System :: POSIX :: Linux",
        "Programming Language :: Python :: 3.6",
        "Programming Language :: Python :: 3.7",
        "Programming Language :: Python :: 3.8",
        "Intended Audience :: Developers",
    ],
    project_urls={
        "Bug Reports": "https://github.com/cortexlabs/cortex/issues",
        "Community": "https://gitter.im/cortexlabs/cortex",
        "Docs": "https://docs.cortex.dev",
        "Source Code": "https://github.com/cortexlabs/cortex",
    },
)
