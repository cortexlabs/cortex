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

import threading as td
import time
import traceback
from collections import defaultdict
from http import HTTPStatus
from typing import Any, Callable, Dict, List

from starlette.responses import Response

from ..exceptions import UserRuntimeException
from ..log import logger


class DynamicBatcher:
    def __init__(self, predictor_impl: Callable, max_batch_size: int, batch_interval: int):
        self.predictor_impl = predictor_impl

        self.batch_max_size = max_batch_size
        self.batch_interval = batch_interval  # measured in seconds

        # waiter prevents new threads from modifying the input batch while a batch prediction is in progress
        self.waiter = td.Event()
        self.waiter.set()

        self.barrier = td.Barrier(self.batch_max_size + 1, action=self.waiter.clear)

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
                logger.error(traceback.format_exc())
            finally:
                self.samples = {}
                self.barrier.reset()
                self.waiter.set()

    @staticmethod
    def _make_batch(samples: Dict[int, Dict[str, Any]]) -> Dict[str, List[Any]]:
        batched_samples = defaultdict(list)
        for thread_id in samples:
            for key, sample in samples[thread_id].items():
                batched_samples[key].append(sample)

        return dict(batched_samples)

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
