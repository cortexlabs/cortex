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
from copy import deepcopy
from typing import Any, Dict, Optional

import dill
from datadog.dogstatsd.base import DogStatsd

from cortex_internal.lib import util
from cortex_internal.lib.api.utils import CortexMetrics
from cortex_internal.lib.api.validations import (
    are_models_specified,
    validate_class_impl,
)
from cortex_internal.lib.client.tensorflow import TensorFlowClient
from cortex_internal.lib.exceptions import CortexException, UserException, UserRuntimeException
from cortex_internal.lib.metrics import MetricsClient
from cortex_internal.lib.model import TFSAPIServingThreadUpdater
from cortex_internal.lib.type import (
    PythonHandlerType,
    TensorFlowNeuronHandlerType,
    TensorFlowHandlerType,
    handler_type_from_api_spec,
)

PYTHON_CLASS_VALIDATION = {
    "required": [
        {
            "name": "__init__",
            "required_args": ["self", "config"],
            "optional_args": ["job_spec", "metrics_client"],
        },
        {
            "name": "handle_batch",
            "required_args": ["self"],
            "optional_args": ["payload", "batch_id"],
        },
    ],
    "optional": [
        {"name": "on_job_complete", "required_args": ["self"]},
    ],
}

TENSORFLOW_CLASS_VALIDATION = {
    "required": [
        {
            "name": "__init__",
            "required_args": ["self", "config", "tensorflow_client"],
            "optional_args": ["job_spec", "metrics_client"],
        },
        {
            "name": "handle_batch",
            "required_args": ["self"],
            "optional_args": ["payload", "batch_id"],
        },
    ],
    "optional": [
        {"name": "on_job_complete", "required_args": ["self"]},
    ],
}


class BatchAPI:
    """
    Class to validate/load the handler class (Handler).
    Also makes the specified models in cortex.yaml available to the handler's implementation.
    """

    def __init__(self, api_spec: dict, statsd_client: DogStatsd, model_dir: str):
        """
        Args:
            api_spec: API configuration.
            model_dir: Where the models are stored on disk.
        """

        self.metrics = CortexMetrics(statsd_client, api_spec)

        self.name = api_spec["name"]
        self.type = handler_type_from_api_spec(api_spec)
        self.path = api_spec["handler"]["path"]
        self.config = api_spec["handler"].get("config", {})
        self.protobuf_path = api_spec["handler"].get("protobuf_path")
        self.api_spec = api_spec

    def initialize_client(
        self, tf_serving_host: Optional[str] = None, tf_serving_port: Optional[str] = None
    ) -> TensorFlowClient:
        """
        Initialize client that gives access to models specified in the API spec (cortex.yaml).
        Only applies when models are provided in the API spec.

        Args:
            tf_serving_host: Host of TF serving server. To be only used when the TensorFlow type is used.
            tf_serving_port: Port of TF serving server. To be only used when the TensorFlow type is used.

        Return:
            The client for the respective handler type.
        """

        client = None

        if are_models_specified(self.api_spec) and self.type in [
            TensorFlowHandlerType,
            TensorFlowNeuronHandlerType,
        ]:
            tf_serving_address = tf_serving_host + ":" + tf_serving_port
            client = TensorFlowClient(
                tf_serving_address,
                self.api_spec,
            )
            TFSAPIServingThreadUpdater(interval=5.0, client=client).start()

        return client

    def initialize_impl(
        self,
        project_dir: str,
        client: TensorFlowClient,
        metrics_client: MetricsClient,
        job_spec: Optional[Dict[str, Any]] = None,
    ):
        """
        Initialize handler class as provided by the user.

        job_spec is a dictionary when the "kind" of the API is set to "BatchAPI". Otherwise, it's None.

        Can raise UserRuntimeException/UserException/CortexException.
        """

        # build args
        class_impl = self.class_impl(project_dir)
        constructor_args = inspect.getfullargspec(class_impl.__init__).args
        config = deepcopy(self.config)
        args = {}
        if job_spec is not None and job_spec.get("config") is not None:
            util.merge_dicts_in_place_overwrite(config, job_spec["config"])
        if "config" in constructor_args:
            args["config"] = config
        if "job_spec" in constructor_args:
            args["job_spec"] = job_spec
        if "metrics_client" in constructor_args:
            args["metrics_client"] = metrics_client

        # initialize handler class
        try:
            if self.type == PythonHandlerType:
                initialized_impl = class_impl(**args)
            if self.type in [TensorFlowHandlerType, TensorFlowNeuronHandlerType]:
                args["tensorflow_client"] = client
                initialized_impl = class_impl(**args)
        except Exception as e:
            raise UserRuntimeException(self.path, "__init__", str(e)) from e

        return initialized_impl

    def class_impl(self, project_dir):
        """Can only raise UserException/CortexException exceptions"""
        target_class_name = "Handler"
        if self.type in [TensorFlowHandlerType, TensorFlowNeuronHandlerType]:
            validations = TENSORFLOW_CLASS_VALIDATION
        elif self.type == PythonHandlerType:
            validations = PYTHON_CLASS_VALIDATION
        else:
            raise CortexException(f"invalid handler type: {self.type}")

        try:
            handler_class = self._get_class_impl(
                "cortex_handler", os.path.join(project_dir, self.path), target_class_name
            )
        except Exception as e:
            e.wrap("error in " + self.path)
            raise

        try:
            validate_class_impl(handler_class, validations)
        except Exception as e:
            e.wrap("error in " + self.path)
            raise
        return handler_class

    def _get_class_impl(self, module_name, impl_path, target_class_name):
        """Can only raise UserException exception"""
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
        handler_class = None
        for class_df in classes:
            if class_df[0] == target_class_name:
                if handler_class is not None:
                    raise UserException(
                        f"multiple definitions for {target_class_name} class found; please check your imports and class definitions and ensure that there is only one handler class definition"
                    )
                handler_class = class_df[1]
        if handler_class is None:
            raise UserException(f"{target_class_name} class is not defined")

        return handler_class
