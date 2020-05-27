import grpc
import time

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
        self.timeout = 600

    def add_models_config(self, names, base_paths, replace_models=False):
        request = model_management_pb2.ReloadConfigRequest()
        model_server_config = model_server_config_pb2.ModelServerConfig()

        # create model(s) configuration
        config_list = model_server_config_pb2.ModelConfigList()
        for i in range(len(names)):
            model_config = config_list.config.add()
            model_config.name = names[i]
            model_config.base_path = base_paths[i]
            model_config.model_platform = self.model_platform

        if replace_models:
            model_server_config.model_config_list.CopyFrom(config_list)
            request.config.CopyFrom(model_server_config)
        else:
            model_server_config.model_config_list.MergeFrom(config_list)
            request.config.MergeFrom(model_server_config)

        # request TFS to load models
        limit = 60
        response = None
        for i in range(limit):
            try:
                stub = model_service_pb2_grpc.ModelServiceStub(self.channel)
                response = stub.HandleReloadConfigRequest(request, self.timeout)
                break
            except Exception as e:
                if isinstance(e, grpc.RpcError) and e.code() == grpc.StatusCode.UNAVAILABLE:
                    if i > 6:  # only start logging this after 30 seconds
                        cx_logger().warn("unable to trigger the loading of model(s) - retrying ...")
                else:
                    print(e)  # unexpected error
                    cx_logger().warn("unable to trigger the loading of model(s) - retrying ...")

            time.sleep(5)

        # report error or success
        if not response is None and response.status.error_code == 0:
            cx_logger().info("successfully loaded {} models into TF-Serving".format(names))
        else:
            raise CortexException(
                "couldn't load user-requested models - failed with error code {}: {}".format(
                    response.status.error_code, response.status.error_message
                )
            )

    def add_model_config(self, name, base_path, replace_model=False):
        self.add_models_config([name], [base_path], replace_model)
