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
from setuptools.command.install import install
import sys


class CustomInstallCommand(install):
    def run(self):
        install.run(self)

        import requests
        from pathlib import Path
        import sys
        import os
        import stat

        # for development purposes
        if os.environ.get("CORTEX_CLI_PATH") is not None:
            print("skipping cli download because environment variable `CORTEX_CLI_PATH` is set")
            return

        if os.path.exists("/usr/local/bin/cortex"):
            raise Exception(
                "it appears that a cortex binary is already installed on this machine using other means; please delete with `sudo rm /usr/local/bin/cortex` to avoid conflicts"
            )

        dest_path = os.path.join(self.install_lib, "cortex", "binary", "cli")

        if not os.path.exists(dest_path):
            platform = sys.platform

            if platform != "darwin" and platform != "linux":
                raise Exception("cortex is only on mac and linux")

            cortex_version = self.config_vars["dist_version"]

            download_url = f"https://s3-us-west-2.amazonaws.com/get-cortex/{cortex_version}/cli/{platform}/cortex"

            with open(dest_path, "wb") as f:
                r = requests.get(download_url, allow_redirects=True)
                f.write(r.content)

            f = Path(dest_path)
            f.chmod(f.stat().st_mode | stat.S_IEXEC)


setup(
    name="cortex",
    version="master",  # CORTEX_VERSION
    description="",
    author="Cortex Labs",
    author_email="dev@cortexlabs.com",
    install_requires=["dill>=0.3.0", "requests>=2.20.0", "msgpack>=0.6.0"],
    setup_requires=(["setuptools"]),
    packages=find_packages(),
    package_data={"cortex.binary": ["cli"]},
    entry_points={
        "console_scripts": [
            "cortex = cortex.binary:run",
        ],
    },
    cmdclass={
        "install": CustomInstallCommand,
    },
    # include_package_data=True,
)
