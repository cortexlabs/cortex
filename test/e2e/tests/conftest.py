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
        "--env",
        action="store",
        default=None,
        help="set cortex environment, to test on an existing cluster",
    )
    parser.addoption(
        "--config",
        action="store",
        default=None,
        help="set cortex cluster config, to test on a new cluster",
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
    parser.addoption(
        "--skip-autoscaling",
        action="store_true",
        help="skip autoscaling tests",
    )
    parser.addoption(
        "--skip-load",
        action="store_true",
        help="skip load tests",
    )
    parser.addoption(
        "--skip-long-running",
        action="store_true",
        help="skip long-running test",
    )


def pytest_configure(config):
    load_dotenv(".env")

    s3_path = os.environ.get("CORTEX_TEST_BATCH_S3_PATH")
    s3_path = config.getoption("--s3-path") if not s3_path else s3_path

    configuration = {
        "aws": {
            "env": config.getoption("--env"),
            "config": config.getoption("--config"),
            "s3_path": s3_path,
        },
        "global": {
            "realtime_deploy_timeout": int(
                os.environ.get("CORTEX_TEST_REALTIME_DEPLOY_TIMEOUT", 200)
            ),
            "batch_deploy_timeout": int(os.environ.get("CORTEX_TEST_BATCH_DEPLOY_TIMEOUT", 30)),
            "batch_job_timeout": int(os.environ.get("CORTEX_TEST_BATCH_JOB_TIMEOUT", 200)),
            "async_deploy_timeout": int(os.environ.get("CORTEX_TEST_ASYNC_DEPLOY_TIMEOUT", 90)),
            "async_workload_timeout": int(
                os.environ.get("CORTEX_TEST_ASYNC_WORKLOAD_TIMEOUT", 200)
            ),
            "task_deploy_timeout": int(os.environ.get("CORTEX_TEST_TASK_DEPLOY_TIMEOUT", 30)),
            "task_job_timeout": int(os.environ.get("CORTEX_TEST_TASK_JOB_TIMEOUT", 200)),
            "skip_gpus": config.getoption("--skip-gpus"),
            "skip_infs": config.getoption("--skip-infs"),
            "skip_autoscaling": config.getoption("--skip-autoscaling"),
            "skip_long_running": config.getoption("--skip-long-running"),
            "skip_load": config.getoption("--skip-load"),
            "autoscaling_test_config": {
                "max_replicas": 20,
            },
            "load_test_config": {
                "realtime": {
                    "total_requests": 10 ** 6,
                    "desired_replicas": 50,
                    "concurrency": 50,
                    "min_rtt": 0.005,  # measured in seconds
                    "max_rtt": 1.000,  # measured in seconds
                    "avg_rtt": 0.06,  # measured in seconds
                    "avg_rtt_tolerance": 0.05,  # measured in seconds
                    "status_code_timeout": 60,  # measured in seconds
                },
                "async": {
                    "total_requests": 10 ** 4,
                    "desired_replicas": 50,
                    "concurrency": 10,
                    "submit_timeout": 120,  # measured in seconds
                    "workload_timeout": 120,  # measured in seconds
                },
                "batch": {
                    "jobs": 10,
                    "workers_per_job": 10,
                    "items_per_job": 10 ** 5,
                    "batch_size": 10 * 2,
                    "workload_timeout": 180,  # measured in seconds
                },
                "task": {
                    "jobs": 10 ** 2,
                    "concurrency": 4,
                    "submit_timeout": 60,  # measured in seconds
                    "workload_timeout": 180,  # measured in seconds
                },
            },
            "long_running_test_config": {
                "time_to_run": 5 * 24 * 3600,  # measured in seconds
                "status_code_timeout": 60,  # measured in seconds
            },
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
        raise ValueError("--env and --config are mutually exclusive")
