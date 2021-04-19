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

import itertools
import threading as td
import time
import traceback
from collections import defaultdict
from http import HTTPStatus
from typing import Any, Callable, Dict, List

from starlette.responses import Response

from cortex_internal.lib.exceptions import UserRuntimeException
from cortex_internal.lib.log import logger


class DynamicBatcher:
    def __init__(
        self,
        predictor_impl: Callable,
        max_batch_size: int,
        batch_interval: int,
        test_mode: bool = False,
    ):
        self.predictor_impl = predictor_impl

        self.batch_max_size = max_batch_size
        self.batch_interval = batch_interval  # measured in seconds
        self.test_mode = test_mode  # only for unit testing
        self._test_batch_lengths = []  # only when unit testing

        self.barrier = td.Barrier(self.batch_max_size + 1)

        self.samples = {}
        self.predictions = {}
        td.Thread(target=self._batch_engine, daemon=True).start()

        self.sample_id_generator = itertools.count()

    def _batch_engine(self):
        while True:
            if len(self.predictions) > 0:
                time.sleep(0.001)
                continue

            try:
                self.barrier.wait(self.batch_interval)
            except td.BrokenBarrierError:
                pass

            self.predictions = {}
            sample_ids = self._get_sample_ids(self.batch_max_size)
            try:
                if self.samples:
                    batch = self._make_batch(sample_ids)

                    predictions = self.predictor_impl.predict(**batch)
                    if not isinstance(predictions, list):
                        raise UserRuntimeException(
                            f"please return a list when using server side batching, got {type(predictions)}"
                        )

                    if self.test_mode:
                        self._test_batch_lengths.append(len(predictions))

                    self.predictions = dict(zip(sample_ids, predictions))
            except Exception as e:
                self.predictions = {sample_id: e for sample_id in sample_ids}
                logger.error(traceback.format_exc())
            finally:
                for sample_id in sample_ids:
                    del self.samples[sample_id]
                self.barrier.reset()

    def _get_sample_ids(self, max_number: int) -> List[int]:
        if len(self.samples) <= max_number:
            return list(self.samples.keys())
        return sorted(self.samples)[:max_number]

    def _make_batch(self, sample_ids: List[int]) -> Dict[str, List[Any]]:
        batched_samples = defaultdict(list)
        for sample_id in sample_ids:
            for key, sample in self.samples[sample_id].items():
                batched_samples[key].append(sample)

        return dict(batched_samples)

    def _enqueue_request(self, sample_id: int, **kwargs):
        """
        Enqueue sample for batch inference. This is a blocking method.
        """

        self.samples[sample_id] = kwargs
        try:
            self.barrier.wait()
        except td.BrokenBarrierError:
            pass

    def predict(self, **kwargs):
        """
        Queues a request to be batched with other incoming request, waits for the response
        and returns the prediction result. This is a blocking method.
        """
        sample_id = next(self.sample_id_generator)
        self._enqueue_request(sample_id, **kwargs)
        prediction = self._get_prediction(sample_id)
        return prediction

    def _get_prediction(self, sample_id: int) -> Any:
        """
        Return the prediction. This is a blocking method.
        """
        while sample_id not in self.predictions:
            time.sleep(0.001)

        prediction = self.predictions[sample_id]
        del self.predictions[sample_id]

        if isinstance(prediction, Exception):
            return Response(
                content=str(prediction),
                status_code=HTTPStatus.INTERNAL_SERVER_ERROR,
                media_type="text/plain",
            )

        return prediction

PYTHON_CLASS_VALIDATION = {
    "required": [
        {
            "name": "__init__",
            "required_args": ["self", "config", "job_spec"],
            "optional_args": ["model_client", "metrics_client"],
        },
        {
            "name": "handle_batch",
            "required_args": ["self"],
            "optional_args": ["payload", "batch_id"]
        },
    ],
    "optional": [
        {
            "name": "load_model",
            "required_args": ["self", "model_path"],
        },
        {
            "name": "on_job_complete",
            "required_args": ["self"]
        },
    ],
}

TENSORFLOW_CLASS_VALIDATION = {
    "required": [
        {
            "name": "__init__",
            "required_args": ["self", "config", "tensorflow_client", "job_spec"],
            "optional_args": ["metrics_client"],
        },
        {
            "name": "handle_batch",
            "required_args": ["self"],
            "optional_args": ["payload", "batch_ud"]
        },
    ],
    "optional": [
        {
            "name": "on_job_complete",
            "required_args": ["self"]
        },
    ],
}


class BatchingAPI:
    """
    Class to validate/load the predictor class (PythonPredictor, TensorFlowPredictor).
    Also makes the specified models in cortex.yaml available to the predictor's implementation.
    """

    def __init__(self, api_spec: dict, model_dir: str):
        """
        Args:
            api_spec: API configuration.
            model_dir: Where the models are stored on disk.
        """

        self.type = predictor_type_from_api_spec(api_spec)
        self.path = api_spec["predictor"]["path"]
        self.config = api_spec["predictor"].get("config", {})
        self.protobuf_path = api_spec["predictor"].get("protobuf_path")

        self.api_spec = api_spec

        self.crons = []
        if not are_models_specified(self.api_spec):
            return

        self.model_dir = model_dir

        self.caching_enabled = self._is_model_caching_enabled()
        self.multiple_processes = self.api_spec["predictor"]["processes_per_replica"] > 1

        # model caching can only be enabled when processes_per_replica is 1
        # model side-reloading is supported for any number of processes_per_replica

        if self.caching_enabled:
            if self.type == PythonPredictorType:
                mem_cache_size = self.api_spec["predictor"]["multi_model_reloading"]["cache_size"]
                disk_cache_size = self.api_spec["predictor"]["multi_model_reloading"][
                    "disk_cache_size"
                ]
            else:
                mem_cache_size = self.api_spec["predictor"]["models"]["cache_size"]
                disk_cache_size = self.api_spec["predictor"]["models"]["disk_cache_size"]
            self.models = ModelsHolder(
                self.type,
                self.model_dir,
                mem_cache_size=mem_cache_size,
                disk_cache_size=disk_cache_size,
                on_download_callback=model_downloader,
            )
        elif not self.caching_enabled and self.type not in [
            TensorFlowPredictorType,
            TensorFlowNeuronPredictorType,
        ]:
            self.models = ModelsHolder(self.type, self.model_dir)
        else:
            self.models = None

        if self.multiple_processes:
            self.models_tree = None
        else:
            self.models_tree = ModelsTree()

    def initialize_client(
        self, tf_serving_host: Optional[str] = None, tf_serving_port: Optional[str] = None
    ) -> Union[PythonClient, TensorFlowClient]:
        """
        Initialize client that gives access to models specified in the API spec (cortex.yaml).
        Only applies when models are provided in the API spec.

        Args:
            tf_serving_host: Host of TF serving server. To be only used when the TensorFlow predictor is used.
            tf_serving_port: Port of TF serving server. To be only used when the TensorFlow predictor is used.

        Return:
            The client for the respective predictor type.
        """

        client = None

        if are_models_specified(self.api_spec):
            if self.type == PythonPredictorType:
                client = PythonClient(self.api_spec, self.models, self.model_dir, self.models_tree)

            if self.type in [TensorFlowPredictorType, TensorFlowNeuronPredictorType]:
                tf_serving_address = tf_serving_host + ":" + tf_serving_port
                client = TensorFlowClient(
                    tf_serving_address,
                    self.api_spec,
                    self.models,
                    self.model_dir,
                    self.models_tree,
                )
                if not self.caching_enabled:
                    cron = TFSAPIServingThreadUpdater(interval=5.0, client=client)
                    cron.start()

        return client

    def initialize_impl(
        self,
        project_dir: str,
        client: Union[PythonClient, TensorFlowClient],
        metrics_client: DogStatsd,
        job_spec: Optional[Dict[str, Any]] = None,
        proto_module_pb2: Optional[Any] = None,
    ):
        """
        Initialize predictor class as provided by the user.

        job_spec is a dictionary when the "kind" of the API is set to "BatchAPI". Otherwise, it's None.
        proto_module_pb2 is a module of the compiled proto when grpc is enabled for the "RealtimeAPI" kind. Otherwise, it's None.

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
        if "proto_module_pb2" in constructor_args:
            args["proto_module_pb2"] = proto_module_pb2

        # initialize predictor class
        try:
            if self.type == PythonPredictorType:
                if are_models_specified(self.api_spec):
                    args["python_client"] = client
                    # set load method to enable the use of the client in the constructor
                    # setting/getting from self in load_model won't work because self will be set to None
                    client.set_load_method(
                        lambda model_path: class_impl.load_model(None, model_path)
                    )
                    initialized_impl = class_impl(**args)
                    client.set_load_method(initialized_impl.load_model)
                else:
                    initialized_impl = class_impl(**args)
            if self.type in [TensorFlowPredictorType, TensorFlowNeuronPredictorType]:
                args["tensorflow_client"] = client
                initialized_impl = class_impl(**args)
        except Exception as e:
            raise UserRuntimeException(self.path, "__init__", str(e)) from e

        # initialize the crons if models have been specified and if the API kind is RealtimeAPI
        if are_models_specified(self.api_spec) and self.api_spec["kind"] == "RealtimeAPI":
            if not self.multiple_processes and self.caching_enabled:
                self.crons += [
                    ModelTreeUpdater(
                        interval=10,
                        api_spec=self.api_spec,
                        tree=self.models_tree,
                        ondisk_models_dir=self.model_dir,
                    ),
                    ModelsGC(
                        interval=10,
                        api_spec=self.api_spec,
                        models=self.models,
                        tree=self.models_tree,
                    ),
                ]

            if not self.caching_enabled and self.type == PythonPredictorType:
                self.crons += [
                    FileBasedModelsGC(interval=10, models=self.models, download_dir=self.model_dir)
                ]

        for cron in self.crons:
            cron.start()

        return initialized_impl

    def class_impl(self, project_dir):
        """Can only raise UserException/CortexException exceptions"""
        if self.type in [TensorFlowPredictorType, TensorFlowNeuronPredictorType]:
            target_class_name = "TensorFlowPredictor"
            validations = TENSORFLOW_CLASS_VALIDATION
        elif self.type == PythonPredictorType:
            target_class_name = "PythonPredictor"
            validations = PYTHON_CLASS_VALIDATION
        else:
            raise CortexException(f"invalid predictor type: {self.type}")

        try:
            predictor_class = self._get_class_impl(
                "cortex_predictor", os.path.join(project_dir, self.path), target_class_name
            )
        except Exception as e:
            e.wrap("error in " + self.path)
            raise

        try:
            validate_class_impl(predictor_class, validations)
            validate_predictor_with_grpc(predictor_class, self.api_spec)
            if self.type == PythonPredictorType:
                validate_python_predictor_with_models(predictor_class, self.api_spec)
        except Exception as e:
            e.wrap("error in " + self.path)
            raise
        return predictor_class

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
        predictor_class = None
        for class_df in classes:
            if class_df[0] == target_class_name:
                if predictor_class is not None:
                    raise UserException(
                        f"multiple definitions for {target_class_name} class found; please check your imports and class definitions and ensure that there is only one Predictor class definition"
                    )
                predictor_class = class_df[1]
        if predictor_class is None:
            raise UserException(f"{target_class_name} class is not defined")

        return predictor_class

    def _is_model_caching_enabled(self) -> bool:
        """
        Checks if model caching is enabled.
        """
        models = None
        if self.type != PythonPredictorType and self.api_spec["predictor"]["models"]:
            models = self.api_spec["predictor"]["models"]
        if self.type == PythonPredictorType and self.api_spec["predictor"]["multi_model_reloading"]:
            models = self.api_spec["predictor"]["multi_model_reloading"]

        return models and models["cache_size"] and models["disk_cache_size"]

    def __del__(self) -> None:
        for cron in self.crons:
            cron.stop()
        for cron in self.crons:
            cron.join()


class RealtimeAPI:
    def __init__(
        self,
        storage: S3,
        api_spec: Dict[str, Any],
        model_dir: str,
        cache_dir: str = ".",
    ):
        self.storage = storage
        self.api_spec = api_spec
        self.cache_dir = cache_dir

        self.id = api_spec["id"]
        self.predictor_id = api_spec["predictor_id"]
        self.deployment_id = api_spec["deployment_id"]

        self.key = api_spec["key"]
        self.metadata_root = api_spec["metadata_root"]
        self.name = api_spec["name"]
        self.predictor = Predictor(api_spec, model_dir)

        host_ip = os.environ["HOST_IP"]
        datadog.initialize(statsd_host=host_ip, statsd_port=9125)
        self.__statsd = datadog.statsd

    @property
    def statsd(self):
        return self.__statsd

    @property
    def python_server_side_batching_enabled(self):
        return (
            self.api_spec["predictor"].get("server_side_batching") is not None
            and self.api_spec["predictor"]["type"] == "python"
        )

    def metric_dimensions_with_id(self):
        return [
            {"Name": "api_name", "Value": self.name},
            {"Name": "api_id", "Value": self.id},
            {"Name": "predictor_id", "Value": self.predictor_id},
            {"Name": "deployment_id", "Value": self.deployment_id},
        ]

    def metric_dimensions(self):
        return [{"Name": "api_name", "Value": self.name}]

    def post_request_metrics(self, status_code, total_time):
        total_time_ms = total_time * 1000
        metrics = [
            self.status_code_metric(self.metric_dimensions(), status_code),
            self.status_code_metric(self.metric_dimensions_with_id(), status_code),
            self.latency_metric(self.metric_dimensions(), total_time_ms),
            self.latency_metric(self.metric_dimensions_with_id(), total_time_ms),
        ]
        self.post_metrics(metrics)

    def post_status_code_request_metrics(self, status_code):
        metrics = [
            self.status_code_metric(self.metric_dimensions(), status_code),
            self.status_code_metric(self.metric_dimensions_with_id(), status_code),
        ]
        self.post_metrics(metrics)

    def post_latency_request_metrics(self, total_time):
        total_time_ms = total_time * 1000
        metrics = [
            self.latency_metric(self.metric_dimensions(), total_time_ms),
            self.latency_metric(self.metric_dimensions_with_id(), total_time_ms),
        ]
        self.post_metrics(metrics)

    def post_metrics(self, metrics):
        try:
            if self.statsd is None:
                raise CortexException("statsd client not initialized")  # unexpected

            for metric in metrics:
                tags = ["{}:{}".format(dim["Name"], dim["Value"]) for dim in metric["Dimensions"]]
                if metric.get("Unit") == "Count":
                    self.statsd.increment(metric["MetricName"], value=metric["Value"], tags=tags)
                else:
                    self.statsd.histogram(metric["MetricName"], value=metric["Value"], tags=tags)
        except:
            logger.warn("failure encountered while publishing metrics", exc_info=True)

    def status_code_metric(self, dimensions, status_code):
        status_code_series = int(status_code / 100)
        status_code_dimensions = dimensions + [
            {"Name": "response_code", "Value": "{}XX".format(status_code_series)}
        ]
        return {
            "MetricName": "cortex_status_code",
            "Dimensions": status_code_dimensions,
            "Value": 1,
            "Unit": "Count",
        }

    def latency_metric(self, dimensions, total_time):
        return {
            "MetricName": "cortex_latency",
            "Dimensions": dimensions,
            "Value": total_time,  # milliseconds
        }