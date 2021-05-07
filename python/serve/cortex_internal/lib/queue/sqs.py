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

import threading
import time
from typing import Callable, Dict, Any, Optional, Tuple

import botocore.exceptions

from cortex_internal.lib.exceptions import UserRuntimeException
from cortex_internal.lib.log import logger as log
from cortex_internal.lib.signals import SignalHandler
from cortex_internal.lib.telemetry import capture_exception


class SQSHandler:
    def __init__(
        self,
        sqs_client,
        queue_url: str,
        dead_letter_queue_url: str = None,
        message_wait_time: int = 10,
        visibility_timeout: int = 30,
        not_found_sleep_time: int = 10,
        renewal_period: int = 15,
        stop_if_no_messages: bool = False,
    ):
        self.sqs_client = sqs_client
        self.queue_url = queue_url
        self.dead_letter_queue_url = dead_letter_queue_url
        self.message_wait_time = message_wait_time
        self.visibility_timeout = visibility_timeout
        self.not_found_sleep_time = not_found_sleep_time
        self.renewal_period = renewal_period
        self.stop_if_no_messages = stop_if_no_messages

        self.receipt_handle_mutex = threading.Lock()
        self.stop_renewal = set()

    def start(
        self,
        message_fn: Callable[[Dict[str, Any]], None],
        message_failure_fn: Callable[[Dict[str, Any]], None],
        on_job_complete_fn: Optional[Callable[[Dict[str, Any]], None]] = None,
    ):
        no_messages_found_in_previous_iteration = False
        signal_handler = SignalHandler()

        while not signal_handler.received_signal():
            response = self.sqs_client.receive_message(
                QueueUrl=self.queue_url,
                MaxNumberOfMessages=1,
                WaitTimeSeconds=self.message_wait_time,
                VisibilityTimeout=self.visibility_timeout,
                MessageAttributeNames=["All"],
            )

            if response.get("Messages") is None or len(response["Messages"]) == 0:
                visible_messages, invisible_messages = self._get_total_messages_in_queue()
                if visible_messages + invisible_messages == 0:
                    if no_messages_found_in_previous_iteration and self.stop_if_no_messages:
                        log.info("no messages left in queue, exiting...")
                        return
                    no_messages_found_in_previous_iteration = True

                time.sleep(self.not_found_sleep_time)
                continue

            no_messages_found_in_previous_iteration = False
            message = response["Messages"][0]
            receipt_handle = message["ReceiptHandle"]

            renewer = threading.Thread(
                target=self._renew_message_visibility,
                args=(receipt_handle,),
                daemon=True,
            )
            renewer.start()

            if is_on_job_complete(message):
                self._on_job_complete(message, on_job_complete_fn)
            else:
                self._handle_message(message, message_fn, message_failure_fn)

    def _renew_message_visibility(self, receipt_handle: str):
        interval = self.renewal_period
        new_timeout = self.visibility_timeout

        cur_time = time.time()
        while True:
            time.sleep((cur_time + interval) - time.time())
            cur_time += interval
            new_timeout += interval

            with self.receipt_handle_mutex:
                if receipt_handle in self.stop_renewal:
                    self.stop_renewal.remove(receipt_handle)
                    break

                try:
                    self.sqs_client.change_message_visibility(
                        QueueUrl=self.queue_url,
                        ReceiptHandle=receipt_handle,
                        VisibilityTimeout=new_timeout,
                    )
                except botocore.exceptions.ClientError as err:
                    if err.response["Error"]["Code"] == "InvalidParameterValue":
                        # unexpected; this error is thrown when attempting to renew a message that has been deleted
                        continue
                    elif err.response["Error"]["Code"] == "AWS.SimpleQueueService.NonExistentQueue":
                        # there may be a delay between the cron may deleting the queue and this worker stopping
                        log.info(
                            "failed to renew message visibility because the queue was not found"
                        )
                    else:
                        self.stop_renewal.remove(receipt_handle)
                        raise err

    def _get_total_messages_in_queue(self):
        return get_total_messages_in_queue(sqs_client=self.sqs_client, queue_url=self.queue_url)

    def _handle_message(self, message, callback_fn, failure_callback_fn):
        receipt_handle = message["ReceiptHandle"]

        try:
            callback_fn(message)
        except Exception as err:
            if not isinstance(err, UserRuntimeException):
                capture_exception(err)

            failure_callback_fn(message)

            with self.receipt_handle_mutex:
                self.stop_renewal.add(receipt_handle)
                if self.dead_letter_queue_url is not None:
                    self.sqs_client.change_message_visibility(  # return message
                        QueueUrl=self.queue_url, ReceiptHandle=receipt_handle, VisibilityTimeout=0
                    )
                else:
                    self.sqs_client.delete_message(
                        QueueUrl=self.queue_url, ReceiptHandle=receipt_handle
                    )
        else:
            with self.receipt_handle_mutex:
                self.stop_renewal.add(receipt_handle)
                self.sqs_client.delete_message(
                    QueueUrl=self.queue_url, ReceiptHandle=receipt_handle
                )

    def _on_job_complete(self, message, callback_fn):
        receipt_handle = message["ReceiptHandle"]
        try:
            callback_fn(message)
        except Exception as err:
            raise type(err)("failed to handle on_job_complete") from err
        finally:
            with self.receipt_handle_mutex:
                self.stop_renewal.add(receipt_handle)
                self.sqs_client.delete_message(
                    QueueUrl=self.queue_url, ReceiptHandle=receipt_handle
                )


def get_total_messages_in_queue(sqs_client, queue_url: str) -> Tuple[int, int]:
    attributes = sqs_client.get_queue_attributes(QueueUrl=queue_url, AttributeNames=["All"])[
        "Attributes"
    ]
    visible_count = int(attributes.get("ApproximateNumberOfMessages", 0))
    not_visible_count = int(attributes.get("ApproximateNumberOfMessagesNotVisible", 0))
    return visible_count, not_visible_count


def is_on_job_complete(message: Dict[str, Any]) -> bool:
    return "MessageAttributes" in message and "job_complete" in message["MessageAttributes"]
