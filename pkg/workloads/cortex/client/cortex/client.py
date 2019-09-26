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


class Client(object):
    def __init__(
        self,
        aws_access_key_id,
        aws_secret_access_key,
        operator_url,
        deployment_name,
        workspace="./workspace",
    ):
        self.operator_url = operator_url
        self.deployment_name = deployment_name
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

    def deploy(self, api_name, model_path, pre_inference, post_inference):
        working_dir = os.path.join(self.workspace, self.deployment_name)
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
            {"kind": "deployment", "name": self.deployment_name},
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

        print(api_working_dir)
        project_zip_path = os.path.join(working_dir, "project")
        shutil.make_archive(project_zip_path, "zip", api_working_dir)
        project_zip_path += ".zip"

        queries = {"force": "false", "ignoreCache": "false"}

        with open(cortex_yaml_path, "rb") as config, open(project_zip_path, "rb") as project:
            files = {"cortex.yaml": config, "project.zip": project}
            print(urllib.parse.urljoin(self.operator_url, "deploy"))
            resp = requests.post(
                urllib.parse.urljoin(self.operator_url, "deploy"),
                params=queries,
                files=files,
                headers=self.headers,
                verify=False,
            )
            print(resp.json())

    def get_endpoint(self, api_name):
        queries = {"appName": self.deployment_name}

        resp = requests.get(
            urllib.parse.urljoin(self.operator_url, "resources"),
            params=queries,
            headers=self.headers,
            verify=False,
        )
        print(resp)
        resources = resp.json()
        print(
            urllib.parse.urljoin(
                resources["apis_base_url"],
                resources["api_name_statuses"][api_name]["active_status"]["path"],
            )
        )
