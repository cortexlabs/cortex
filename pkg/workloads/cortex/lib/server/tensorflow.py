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

import grpc
import time
import threading

from tensorflow_serving.apis import model_service_pb2_grpc
from tensorflow_serving.apis import model_management_pb2
from tensorflow_serving.config import model_server_config_pb2

from cortex.lib.exceptions import CortexException
from cortex.lib.log import cx_logger


class TensorFlowServing:
    def __init__(self, address):
        self.address = address
        self.model_platform = "tensorflow"
        self.channel = grpc.insecure_channel(self.address)
        self.stub = model_service_pb2_grpc.ModelServiceStub(self.channel)
        self.timeout = 600  # gRPC timeout in seconds

    def add_models_config(self, names, base_paths, replace_models=False):
        request = model_management_pb2.ReloadConfigRequest()
        model_server_config = model_server_config_pb2.ModelServerConfig()

        # create model(s) configuration
        config_list = model_server_config_pb2.ModelConfigList()
        for i, name in enumerate(names):
            model_config = config_list.config.add()
            model_config.name = name
            model_config.base_path = base_paths[i]
            model_config.model_platform = self.model_platform

        if replace_models:
            model_server_config.model_config_list.CopyFrom(config_list)
            request.config.CopyFrom(model_server_config)
        else:
            model_server_config.model_config_list.MergeFrom(config_list)
            request.config.MergeFrom(model_server_config)

        loaded_models = threading.Event()

        def log_loading_models():
            while not loaded_models.is_set():
                time.sleep(2)
                cx_logger().info("model(s) still loading ...")

        log_thread = threading.Thread(target=log_loading_models, daemon=True)
        log_thread.start()

        timeout_error_limit = 3
        timeout_error_counter = 0
        generic_error_limit = 200
        generic_error_counter = 0

        # request TFS to load models
        response = None
        while True:
            try:
                # this request doesn't return until all models have been successfully loaded
                response = self.stub.HandleReloadConfigRequest(request, self.timeout)
                break
            except Exception as e:
                if not (
                    isinstance(e, grpc.RpcError)
                    and e.code() in [grpc.StatusCode.UNAVAILABLE, grpc.StatusCode.DEADLINE_EXCEEDED]
                ):
                    print(e)  # unexpected error

                if isinstance(e, grpc.RpcError) and e.code() == grpc.StatusCode.DEADLINE_EXCEEDED:
                    timeout_error_counter += 1
                else:
                    generic_error_counter += 1

            if timeout_error_counter >= timeout_error_limit:
                break
            if generic_error_counter >= generic_error_limit:
                break

            time.sleep(1.0)

        loaded_models.set()
        log_thread.join()

        # report error or success
        if response and response.status.error_code == 0:
            cx_logger().info("successfully loaded {} models into TF-Serving".format(names))
        else:
            if response:
                raise CortexException(
                    "couldn't load user-requested models - failed with error code {}: {}".format(
                        response.status.error_code, response.status.error_message
                    )
                )
            else:
                raise CortexException("couldn't load user-requested models")

    def add_model_config(self, name, base_path, replace_model=False):
        self.add_models_config([name], [base_path], replace_model)
