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

        dest_path = os.path.join(self.install_lib, "cortex", "binary", "cli")

        if not os.path.exists(dest_path):
            platform = sys.platform

            if platform != "darwin" and platform != "linux":
                raise Exception("cortex is only on mac and linux")

            cortex_version = self.config_vars["dist_version"]

            if "dev" in cortex_version:
                cortex_version = "master"

            download_url = f"https://s3-us-west-2.amazonaws.com/get-cortex/{cortex_version}/cli/{platform}/cortex"

            with open(dest_path, "wb") as f:
                r = requests.get(download_url, allow_redirects=True)
                f.write(r.content)

            f = Path(dest_path)
            f.chmod(f.stat().st_mode | stat.S_IEXEC)


setup(
    name="cortex",
    version="0.20.0.dev3",  # CORTEX_VERSION
    description="Model serving at scale",
    author="cortex.dev",
    author_email="dev@cortexlabs.com",
    license="Apache License 2.0",
    long_description_content_type="text/markdown",
    long_description=open("README.md").read(),
    url="https://github.com/cortexlabs/cortex",
    setup_requires=(["setuptools"]),
    packages=find_packages(),
    package_data={"cortex.binary": ["cli"]},
    entry_points={
        "console_scripts": [
            "cortex = cortex.binary:run",
        ],
    },
    python_requires=">=3.6.1",
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
        "Chat with us": "https://gitter.im/cortexlabs/cortex",
        "Documentation": "https://docs.cortex.dev",
        "Source Code": "https://github.com/cortexlabs/cortex",
    },
)
