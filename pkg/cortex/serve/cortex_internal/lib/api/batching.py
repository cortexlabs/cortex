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
    def __init__(self, predictor_impl: Callable, max_batch_size: int, batch_interval: int):
        self.predictor_impl = predictor_impl

        self.batch_max_size = max_batch_size
        self.batch_interval = batch_interval  # measured in seconds

        self.barrier = td.Barrier(self.batch_max_size + 1)

        self.samples = {}
        self.predictions = {}
        td.Thread(target=self._batch_engine).start()

        self.thread_id_generator = itertools.count()

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
            thread_ids = self._get_thread_ids(self.batch_max_size)
            try:
                if self.samples:
                    batch = self._make_batch(thread_ids)

                    predictions = self.predictor_impl.predict(**batch)
                    if not isinstance(predictions, list):
                        raise UserRuntimeException(
                            f"please return a list when using server side batching, got {type(predictions)}"
                        )

                    self.predictions = dict(zip(thread_ids, predictions))
            except Exception as e:
                self.predictions = {thread_id: e for thread_id in thread_ids}
                logger.error(traceback.format_exc())
            finally:
                for thread_id in thread_ids:
                    del self.samples[thread_id]
                self.barrier.reset()

    def _get_thread_ids(self, max_number: int) -> List[int]:
        if len(self.samples) <= max_number:
            return list(self.samples.keys())
        return sorted(self.samples)[:max_number]

    def _make_batch(self, thread_ids: List[int]) -> Dict[str, List[Any]]:
        batched_samples = defaultdict(list)
        for thread_id in thread_ids:
            for key, sample in self.samples[thread_id].items():
                batched_samples[key].append(sample)

        return dict(batched_samples)

    def _enqueue_request(self, thread_id: int, **kwargs):
        """
        Enqueue sample for batch inference. This is a blocking method.
        """

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
        thread_id = next(self.thread_id_generator)
        self._enqueue_request(thread_id, **kwargs)
        prediction = self._get_prediction(thread_id)
        return prediction

    def _get_prediction(self, thread_id: int) -> Any:
        """
        Return the prediction. This is a blocking method.
        """
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
