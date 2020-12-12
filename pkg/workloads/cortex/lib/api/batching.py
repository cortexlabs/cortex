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

import threading as td
import time
import traceback
from http import HTTPStatus
from typing import Any, Callable, Dict, List

from starlette.responses import Response

from ..exceptions import UserRuntimeException
from ..log import cx_logger as logger


class DynamicBatcher:
    def __init__(self, predictor_impl: Callable, max_batch_size: int, batch_interval: int):
        self.predictor_impl = predictor_impl

        self.batch_max_size = max_batch_size
        self.batch_interval = batch_interval  # measured in seconds

        self.waiter = td.Event()
        self.waiter.set()

        self.barrier = td.Barrier(self.batch_max_size + 1)

        self.samples = {}
        self.predictions = {}
        td.Thread(target=self._batch_engine).start()

    def _batch_engine(self):
        while True:
            if len(self.predictions) > 0:
                time.sleep(0.001)
                continue

            try:
                self.barrier.wait(self.batch_interval)
            except td.BrokenBarrierError:
                pass

            self.waiter.clear()
            self.predictions = {}

            try:
                if self.samples:
                    batch = self._make_batch(self.samples)

                    predictions = self.predictor_impl.predict(**batch)
                    if not isinstance(predictions, list):
                        raise UserRuntimeException(
                            f"please return a list when using server side batching, got {type(predictions)}"
                        )

                    self.predictions = dict(zip(self.samples.keys(), predictions))
            except Exception as e:
                self.predictions = {thread_id: e for thread_id in self.samples}
                logger().error(traceback.format_exc())
                continue
            finally:
                self.samples = {}
                self.barrier.reset()
                self.waiter.set()

    @staticmethod
    def _make_batch(samples: Dict[int, Dict[str, Any]]) -> Dict[str, List[Any]]:
        # TODO: seems a bit inefficient to me, but there is a possibility each request contains different keys
        batched_samples = {key: [] for sample in samples.values() for key in sample}
        for sample in samples.values():
            for key in batched_samples:
                batched_samples[key].append(sample.get(key))

        return batched_samples

    def _enqueue_request(self, **kwargs):
        """
        Enqueue sample for batch inference. This is a blocking method.
        """
        thread_id = td.get_ident()

        self.waiter.wait()
        self.samples[thread_id] = kwargs
        try:
            self.barrier.wait()
        except td.BrokenBarrierError:
            pass

    def predict(self, **kwargs):
        """
        Queues a request to be batched with other incoming request, waits for the response
        and returns the prediction result. This is a blocking method.
        """
        self._enqueue_request(**kwargs)
        prediction = self._get_prediction()
        return prediction

    def _get_prediction(self) -> Any:
        """
        Return the prediction. This is a blocking method.
        """
        thread_id = td.get_ident()
        while thread_id not in self.predictions:
            time.sleep(0.001)

        prediction = self.predictions[thread_id]
        del self.predictions[thread_id]

        if isinstance(prediction, Exception):
            return Response(
                content=str(prediction),
                status_code=HTTPStatus.INTERNAL_SERVER_ERROR,
                media_type="text/plain",
            )

        return prediction
