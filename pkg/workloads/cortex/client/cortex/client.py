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

import pathlib
import os
import types
import subprocess
import sys
import shutil
import yaml
import urllib.parse

import dill
import requests
from requests.exceptions import HTTPError


class Client(object):
    def __init__(
        self, aws_access_key_id, aws_secret_access_key, operator_url, workspace="./workspace"
    ):
        self.operator_url = operator_url
        self.workspace = workspace
        self.aws_access_key_id = aws_access_key_id
        self.aws_secret_access_key = aws_secret_access_key
        self.headers = {
            "CortexAPIVersion": "master",
            "Authorization": "CortexAWS {}|{}".format(
                self.aws_access_key_id, self.aws_secret_access_key
            ),
        }

        pathlib.Path(self.workspace).mkdir(parents=True, exist_ok=True)

    def deploy(self, deployment_name, api_name, model_path, pre_inference, post_inference):
        working_dir = os.path.join(self.workspace, deployment_name)
        api_working_dir = os.path.join(working_dir, api_name)
        pathlib.Path(api_working_dir).mkdir(parents=True, exist_ok=True)

        reqs = subprocess.check_output([sys.executable, "-m", "pip", "freeze"])

        with open(os.path.join(api_working_dir, "requirements.txt"), "w") as f:
            f.writelines(reqs.decode())

        with open(os.path.join(api_working_dir, "request_handler.pickle"), "wb") as f:
            dill.dump(
                {"pre_inference": pre_inference, "post_inference": post_inference}, f, recurse=True
            )

        cortex_config = [
            {"kind": "deployment", "name": deployment_name},
            {
                "kind": "api",
                "model": model_path,
                "name": api_name,
                "request_handler": "request_handler.pickle",
            },
        ]

        cortex_yaml_path = os.path.join(working_dir, "cortex.yaml")
        with open(cortex_yaml_path, "w") as f:
            f.write(yaml.dump(cortex_config))

        project_zip_path = os.path.join(working_dir, "project")
        shutil.make_archive(project_zip_path, "zip", api_working_dir)
        project_zip_path += ".zip"

        queries = {"force": "false", "ignoreCache": "false"}

        deployment_status = {}

        with open(cortex_yaml_path, "rb") as config, open(project_zip_path, "rb") as project:
            files = {"cortex.yaml": config, "project.zip": project}
            try:
                resp = requests.post(
                    urllib.parse.urljoin(self.operator_url, "deploy"),
                    params=queries,
                    files=files,
                    headers=self.headers,
                    verify=False,
                )
                resp.raise_for_status()
                deployment_status = resp.json()
            except HTTPError as err:
                resp = err.response
                if "error" in resp.json():
                    raise Exception(resp.json()["error"]) from err
                raise

            if resp.status_code / 100 == 2:
                return self.get_endpoint(deployment_name, api_name)

            return None

    def get_endpoint(self, deployment_name, api_name):
        queries = {"appName": deployment_name}

        try:
            resp = requests.get(
                urllib.parse.urljoin(self.operator_url, "resources"),
                params=queries,
                headers=self.headers,
                verify=False,
            )
            resp.raise_for_status()

            resources = resp.json()
            return urllib.parse.urljoin(
                resources["apis_base_url"],
                resources["api_name_statuses"][api_name]["active_status"]["path"],
            )
        except HTTPError as err:
            resp = err.response
            if "error" in resp.json():
                raise Exception(resp.json()["error"]) from err
            raise
