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

from collections import namedtuple

# [
#   {
#     "NotBefore" : "21 Jan 2019 09:00:43 GMT",
#     "Code" : "system-reboot",
#     "Description" : "scheduled reboot",
#     "EventId" : "instance-event-0d59937288b749b32",
#     "NotAfter" : "21 Jan 2019 09:17:23 GMT",
#     "State" : "active"
#   }
# ]

ScheduledEventDetail = namedtuple(
    "ScheduledEventDetail",
    [
        "NotBefore",
        "Code",  # system-reboot or instance-reboot
        "Description",
        "EventId",
        "NotAfter",
        "State",
    ],
)

InstanceAction = namedtuple(
    "InstanceAction",
    [
        "action",  # terminate or stop
        "time",  # time when this action was emitted
    ],
)

RebalanceRecommendation = namedtuple(
    "RebalanceRecommendation",
    [
        "noticeTime",  # time when this recommendation was issued
    ],
)

NodeMetadata = namedtuple(
    "NodeMetadata",
    [
        "instanceId",
        "instanceLifeCycle",
        "instanceType",
        "publicHostname",
        "publicIp",
        "localHostname",
        "localIp",
        "availabilityZone",
        "region",
    ],
)
