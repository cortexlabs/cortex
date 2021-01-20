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
import imp
import inspect
import dill

from cortex_internal.lib.exceptions import CortexException, UserException, UserRuntimeException
from cortex_internal.lib.log import configure_logger

logger = configure_logger("cortex", os.environ["CORTEX_LOG_CONFIG_FILE"])


class TaskAPI:
    def __init__(self, provider: str, api_spec: dict):
        """
        Args:
            provider: "aws" or "gcp".
            api_spec: API configuration.
        """

        self.provider = provider

        self.path = api_spec["definition"]["path"]
        self.config = api_spec["definition"].get("config", {})
        self.api_spec = api_spec

    def get_callable(self, project_dir: str):
        impl = self._get_impl(project_dir)
        if inspect.isclass(impl):
            return impl()
        return impl

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

    def _read_impl(self, module_name: str, impl_path: str, target_class_name: str):
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
        callables = inspect.getmembers(impl, callable)

        if len(classes) > 0:
            task_class = None
            for class_df in classes:
                if class_df[0] == target_class_name:
                    if task_class is not None:
                        raise UserException(
                            f"multiple definitions for {target_class_name} class found; please check your imports and class definitions and ensure that there is only one task class definition"
                        )
                    task_class = class_df[1]
            if task_class is None:
                raise UserException(f"{target_class_name} class is not defined")
            return task_class
        elif len(callables) == 0:
            raise UserException("no callable class or function were provided")
        else:
            return callables[0]

    def _validate_impl(self, impl):
        if inspect.isclass(impl):
            target_class_name = impl.__name__

            constructor_fn = getattr(impl, "__init__", None)
            if constructor_fn:
                argspec = inspect.getfullargspec(constructor_fn)
                if not (len(argspec.args) == 1 and argspec.args[0] == "self"):
                    raise UserException(
                        f"class {target_class_name}",
                        f'invalid signature for method "__init__"',
                        f'only "self" parameter must be present in method\'s signature',
                    )

            callable_fn = getattr(impl, "__call__", None)
            if callable_fn:
                argspec = inspect.getfullargspec(callable_fn)
                if not (
                    len(argspec.args) == 2
                    and argspec.args[0] == "self"
                    and argspec.args[1] == "config"
                ):
                    raise UserException(
                        f"class {target_class_name}",
                        f'invalid signature for method "__call__"',
                        f'the following parameters must be present in method\'s signature: "self", "config"',
                    )
            else:
                raise UserException(
                    f"class {target_class_name}",
                    f'"__call__" method not defined',
                )
        else:
            callable_fn = impl
            argspec = inspect.getfullargspec(callable_fn)
            if not (len(argspec.args) == 1 and argspec.args[0] == "config"):
                raise UserException(
                    f'callable function must have the "config" parameter in its signature',
                )
