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

#  SpotInstanceActionPath is the context path to spot/instance-action within IMDS
SpotInstanceActionPath = "latest/meta-data/spot/instance-action"

#  ScheduledEventPath is the context path to events/maintenance/scheduled within IMDS
ScheduledEventPath = "latest/meta-data/events/maintenance/scheduled"

#  RebalanceRecommendationPath is the context path to events/recommendations/rebalance within IMDS
RebalanceRecommendationPath = "latest/meta-data/events/recommendations/rebalance"

#  InstanceIDPath path to account id
AccountIDPath = "latest/meta-data/account-id"

#  InstanceIDPath path to instance id
InstanceIDPath = "latest/meta-data/instance-id"

#  InstanceLifeCycle path to instance life cycle
InstanceLifeCycle = "latest/meta-data/instance-life-cycle"

#  InstanceTypePath path to instance type
InstanceTypePath = "latest/meta-data/instance-type"

#  PublicHostnamePath path to public hostname
PublicHostnamePath = "latest/meta-data/public-hostname"

#  PublicIPPath path to public ip
PublicIPPath = "latest/meta-data/public-ipv4"

#  LocalHostnamePath path to local hostname
LocalHostnamePath = "latest/meta-data/local-hostname"

#  LocalIPPath path to local ip
LocalIPPath = "latest/meta-data/local-ipv4"

#  AZPlacementPath path to availability zone placement
AZPlacementPath = "latest/meta-data/placement/availability-zone"

#  IdentityDocPath is the path to the instance identity document
IdentityDocPath = "latest/dynamic/instance-identity/document"
