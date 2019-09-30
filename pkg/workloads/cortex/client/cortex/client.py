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
from pathlib import Path
import os
import types
import subprocess
import sys
import shutil
import yaml
import urllib.parse
import base64

import dill
import requests
from requests.exceptions import HTTPError
import msgpack


class Client(object):
    def __init__(self, aws_access_key_id, aws_secret_access_key, operator_url):
        """Initialize a Client to a Cortex Operator

        Args:
            aws_access_key_id (string): AWS access key associated with the account that the cluster is running on
            aws_secret_access_key (string): AWS secret key associated with the AWS access key
            operator_url (string): operator URL of your cluster
        """

        self.operator_url = operator_url
        self.workspace = str(Path.home() / ".cortex" / "workspace")
        self.aws_access_key_id = aws_access_key_id
        self.aws_secret_access_key = aws_secret_access_key
        self.headers = {
            "CortexAPIVersion": "master",
            "Authorization": "CortexAWS {}|{}".format(
                self.aws_access_key_id, self.aws_secret_access_key
            ),
        }

        pathlib.Path(self.workspace).mkdir(parents=True, exist_ok=True)

    def deploy(
        self,
        deployment_name,
        api_name,
        model_path,
        pre_inference=None,
        post_inference=None,
        model_format=None,
        tf_serving_key=None,
    ):
        """Deploy an API

        Args:
            deployment_name (string): deployment name
            api_name (string): API name
            model_path (string): S3 path to an exported model
            pre_inference (function, optional): function used to prepare requests for model input
            post_inference (function, optional): function used to prepare model output for response
            model_format (string, optional): model format, must be "tensorflow" or "onnx" (default: "onnx" if model path ends with .onnx, "tensorflow" if model path ends with .zip or is a directory)
            tf_serving_key (string, optional): name of the signature def to use for prediction (required if your model has more than one signature def)

        Returns:
            string: url to the deployed API
        """

        working_dir = os.path.join(self.workspace, deployment_name)
        api_working_dir = os.path.join(working_dir, api_name)
        pathlib.Path(api_working_dir).mkdir(parents=True, exist_ok=True)

        api_config = {"kind": "api", "model": model_path, "name": api_name}

        if tf_serving_key is not None:
            api_config["model_format"] = tf_serving_key

        if model_format is not None:
            api_config["model_format"] = model_format

        if pre_inference is not None or post_inference is not None:
            reqs = subprocess.check_output([sys.executable, "-m", "pip", "freeze"])

            with open(os.path.join(api_working_dir, "requirements.txt"), "w") as f:
                f.writelines(reqs.decode())

            handlers = {}

            if pre_inference is not None:
                handlers["pre_inference"] = pre_inference

            if post_inference is not None:
                handlers["post_inference"] = post_inference

            with open(os.path.join(api_working_dir, "request_handler.pickle"), "wb") as f:
                dill.dump(handlers, f, recurse=True)

            api_config["request_handler"] = "request_handler.pickle"

        cortex_config = [{"kind": "deployment", "name": deployment_name}, api_config]

        cortex_yaml_path = os.path.join(working_dir, "cortex.yaml")
        with open(cortex_yaml_path, "w") as f:
            f.write(yaml.dump(cortex_config))

        project_zip_path = os.path.join(working_dir, "project")
        shutil.make_archive(project_zip_path, "zip", api_working_dir)
        project_zip_path += ".zip"

        queries = {"force": "false", "ignoreCache": "false"}

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
                resources = resp.json()
            except HTTPError as err:
                resp = err.response
                if "error" in resp.json():
                    raise Exception(resp.json()["error"]) from err
                raise

            b64_encoded_context = resources["context"]
            context_msgpack_bytestring = base64.b64decode(b64_encoded_context)
            ctx = msgpack.loads(context_msgpack_bytestring, raw=False)
            return urllib.parse.urljoin(resources["apis_base_url"], ctx["apis"][api_name]["path"])
