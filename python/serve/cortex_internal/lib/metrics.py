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

from typing import Dict

from datadog import DogStatsd

from cortex_internal.lib.exceptions import UserException


def validate_metric(fn):
    def _metric_validation(self, metric: str, value: float, tags: Dict[str, str] = None):
        internal_prefixes = ("cortex_", "istio_")
        if metric.startswith(internal_prefixes):
            raise UserException(
                f"Metric name ({metric}) is invalid because it starts with a cortex exclusive prefix.\n"
                f"The following are prefixes are exclusive to cortex: {internal_prefixes}."
            )
        return fn(self, metric=metric, value=value, tags=tags)

    _metric_validation.__name__ = fn.__name__
    return _metric_validation


class MetricsClient:
    def __init__(self, statsd_client: DogStatsd):
        self.__statsd = statsd_client

    @validate_metric
    def gauge(self, metric: str, value: float, tags: Dict[str, str] = None):
        """
        Record the value of a gauge.

        Example:
        >>> metrics.gauge('active_connections', 1001, tags={"protocol": "http"})
        """
        return self.__statsd.gauge(metric, value=value, tags=transform_tags(tags))

    @validate_metric
    def increment(self, metric: str, value: float = 1, tags: Dict[str, str] = None):
        """
        Increment the value of a counter.

        Example:
        >>> metrics.increment('model_calls', 1, tags={"model_version": "v1"})
        """
        return self.__statsd.increment(metric, value=value, tags=transform_tags(tags))

    @validate_metric
    def histogram(self, metric: str, value: float, tags: Dict[str, str] = None):
        """
        Set the value in a histogram metric

        Example:
        >>> metrics.histogram('inference_time_milliseconds', 120, tags={"model_version": "v1"})
        """
        return self.__statsd.histogram(metric, value=value, tags=transform_tags(tags))


def transform_tags(tags: Dict[str, str] = None):
    return [f"{key}:{value}" for key, value in tags.items()] if tags else None
