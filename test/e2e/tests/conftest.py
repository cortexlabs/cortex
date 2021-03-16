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
import os

import pytest
import yaml
from dotenv import load_dotenv


def pytest_addoption(parser):
    parser.addoption(
        "--aws-env",
        action="store",
        default=None,
        help="set cortex AWS environment, to test on an existing AWS cluster",
    )
    parser.addoption(
        "--aws-config",
        action="store",
        default=None,
        help="set cortex AWS cluster config, to test on a new AWS cluster",
    )
    parser.addoption(
        "--gcp-env",
        action="store",
        default=None,
        help="set cortex GCP environment, to test on an existing GCP cluster",
    )
    parser.addoption(
        "--gcp-config",
        action="store",
        default=None,
        help="set cortex GCP cluster config, to test on a new GCP cluster",
    )
    parser.addoption(
        "--s3-path",
        action="store",
        default=None,
        help="set s3 path where batch jobs results will be stored",
    )
    parser.addoption(
        "--skip-gpus",
        action="store_true",
        help="skip GPU tests",
    )
    parser.addoption(
        "--skip-infs",
        action="store_true",
        help="skip Inferentia tests",
    )


def pytest_configure(config):
    load_dotenv(".env")

    s3_path = os.environ.get("CORTEX_TEST_BATCH_S3_PATH")
    s3_path = config.getoption("--s3-path") if not s3_path else s3_path

    configuration = {
        "aws": {
            "env": config.getoption("--aws-env"),
            "config": config.getoption("--aws-config"),
            "s3_path": s3_path,
        },
        "gcp": {
            "env": config.getoption("--gcp-env"),
            "config": config.getoption("--gcp-config"),
        },
        "global": {
            "realtime_deploy_timeout": int(
                os.environ.get("CORTEX_TEST_REALTIME_DEPLOY_TIMEOUT", 200)
            ),
            "batch_deploy_timeout": int(os.environ.get("CORTEX_TEST_BATCH_DEPLOY_TIMEOUT", 30)),
            "batch_job_timeout": int(os.environ.get("CORTEX_TEST_BATCH_JOB_TIMEOUT", 200)),
            "skip_gpus": config.getoption("--skip-gpus"),
            "skip_infs": config.getoption("--skip-infs"),
        },
    }

    class Config:
        @pytest.fixture(autouse=True)
        def config(self):
            return configuration

    config.pluginmanager.register(Config())

    print("\n----- Test Configuration -----\n")
    print(yaml.dump(configuration, indent=2))

    if configuration["aws"]["env"] and configuration["aws"]["config"]:
        raise ValueError("--aws-env and --aws-config are mutually exclusive")

    if configuration["gcp"]["env"] and configuration["gcp"]["config"]:
        raise ValueError("--gcp-env and --gcp-config are mutually exclusive")
