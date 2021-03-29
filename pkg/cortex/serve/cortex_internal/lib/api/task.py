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

import imp
import inspect
import os

import datadog
import dill

from cortex_internal.lib.api.validations import validate_class_impl
from cortex_internal.lib.exceptions import CortexException, UserException
from cortex_internal.lib.log import configure_logger
from cortex_internal.lib.metrics import MetricsClient

logger = configure_logger("cortex", os.environ["CORTEX_LOG_CONFIG_FILE"])

TASK_CLASS_VALIDATION = {
    "required": [
        {
            "name": "__init__",
            "required_args": ["self"],
            "optional_args": ["metrics_client"],
        },
        {
            "name": "__call__",
            "required_args": ["self", "config"],
            "optional_args": [],
        },
    ],
    "optional": [],
}


class TaskAPI:
    def __init__(self, api_spec: dict):
        """
        Args:
            api_spec: API configuration.
        """

        self.path = api_spec["definition"]["path"]
        self.config = api_spec["definition"].get("config", {})
        self.api_spec = api_spec

        host_ip = os.environ["HOST_IP"]
        datadog.initialize(statsd_host=host_ip, statsd_port=9125)
        self.__statsd = datadog.statsd

    @property
    def statsd(self):
        return self.__statsd

    def get_callable(self, project_dir: str):
        impl = self._get_impl(project_dir)

        constructor_args = inspect.getfullargspec(impl.__init__).args
        args = {}
        if "metrics_client" in constructor_args:
            args["metrics_client"] = MetricsClient(self.statsd)

        return impl(**args)

    def _get_impl(self, project_dir: str):
        try:
            task_callable = self._read_impl(
                "cortex_task", os.path.join(project_dir, self.path), "Task"
            )
        except CortexException as e:
            e.wrap("error in " + self.path)
            raise

        try:
            self._validate_impl(task_callable)
        except CortexException as e:
            e.wrap("error in " + self.path)
            raise
        return task_callable

    @staticmethod
    def _read_impl(module_name: str, impl_path: str, target_class_name: str):
        if impl_path.endswith(".pickle"):
            try:
                with open(impl_path, "rb") as pickle_file:
                    return dill.load(pickle_file)
            except Exception as e:
                raise UserException("unable to load pickle", str(e)) from e

        try:
            impl = imp.load_source(module_name, impl_path)
        except Exception as e:
            raise UserException(str(e)) from e

        classes = inspect.getmembers(impl, inspect.isclass)

        if len(classes) > 0:
            task_class = None
            for class_df in classes:
                if class_df[0] == target_class_name:
                    if task_class is not None:
                        raise UserException(
                            f"multiple definitions for {target_class_name} class found; please check "
                            f"your imports and class definitions and ensure that there is only one "
                            f"task class definition"
                        )
                    task_class = class_df[1]
            if task_class is None:
                raise UserException(f"{target_class_name} class is not defined")
            return task_class
        else:
            raise UserException("no callable class was provided")

    @staticmethod
    def _validate_impl(impl):
        validate_class_impl(impl, TASK_CLASS_VALIDATION)
