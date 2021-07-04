from typing import Any, List, Optional
import requests
import time
import logging

from . import paths, metadata_types

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

# IMDSv2 token related constants
tokenRefreshPath = "latest/api/token"
tokenTTLHeader = "X-aws-ec2-metadata-token-ttl-seconds"
tokenRequestHeader = "X-aws-ec2-metadata-token"
tokenTTL = "21600"
secondsBeforeTTLRefresh = 15
tokenRetryAttempts = 2
pathsAccepted = [
    paths.SpotInstanceActionPath,
    paths.ScheduledEventPath,
    paths.RebalanceRecommendationPath,
]

# access token
token: Optional[str] = None
refreshTime: Optional[int] = None

# inspired from https://github.com/aws/aws-node-termination-handler/


def _request_imds(context_path: str, endpoint: str = "169.254.169.254") -> requests.Response:
    global token, refreshTime

    if context_path not in pathsAccepted:
        raise ValueError(
            f"context_path {context_path} is not in one of the following values: {pathsAccepted}"
        )

    if (token is None and refreshTime is None) or (
        refreshTime is not None
        and time.time() - refreshTime + secondsBeforeTTLRefresh >= int(tokenTTL)
    ):
        for _ in range(tokenRetryAttempts):
            response = requests.put(
                f"http://{endpoint}/{tokenRefreshPath}", headers={tokenTTLHeader: tokenTTL}
            )
            if response.status_code == 200:
                token = response.text
                refreshTime = time.time()

        if response.status_code != 200:
            logging.debug("unable to retrieve an IMDSv2 token, continuing with IMDSv1")
            token = None
            refreshTime = None

    if token:
        return requests.get(
            f"http://{endpoint}/{context_path}",
            headers={
                tokenRequestHeader: token,
            },
        )

    return requests.get(f"http://{endpoint}/{context_path}")


def _request_metadata(context_path: str, endpoint: str = "169.254.169.254") -> requests.Response:
    return requests.get(f"http://{endpoint}/{context_path}")


def get_scheduled_maintainence_events(
    endpoint: str = "169.254.169.254",
) -> List[metadata_types.ScheduledEventDetail]:
    response = _request_imds(paths.ScheduledEventPath, endpoint)
    if response.status_code == 404:
        return []
    elif response.status_code == 200:
        eventDetails = []
        for event in response.json():
            eventDetails.append(
                metadata_types.ScheduledEventDetail(
                    event["NotBefore"],
                    event["Code"],
                    event["Description"],
                    event["EventId"],
                    event["NotAfter"],
                    event["State"],
                )
            )
        return eventDetails
    raise RuntimeError(
        f"received {response.status_code} status code on {paths.ScheduledEventPath} path"
    )


def get_spot_interruption_event(endpoint: str = "169.254.169.254") -> metadata_types.InstanceAction:
    response = _request_imds(paths.SpotInstanceActionPath, endpoint)
    if response.status_code == 404:
        return None
    elif response.status_code == 200:
        data = response.json()
        return metadata_types.InstanceAction(data["action"], data["time"])
    raise RuntimeError(
        f"received {response.status_code} status code on {paths.SpotInstanceActionPath} path"
    )


def get_rebalance_recommendation_event(
    endpoint: str = "169.254.169.254",
) -> metadata_types.RebalanceRecommendation:
    response = _request_imds(paths.RebalanceRecommendationPath, endpoint)
    if response.status_code == 404:
        return None
    elif response.status_code == 200:
        data = response.json()
        return metadata_types.RebalanceRecommendation(data["noticeTime"])
    raise RuntimeError(
        f"received {response.status_code} status code on {paths.RebalanceRecommendationPath} path"
    )


def get_metadata_info(endpoint: str = "169.254.169.254") -> metadata_types.NodeMetadata:
    labels = [
        ("instanceId", paths.InstanceIDPath),
        ("instanceLifeCycle", paths.InstanceLifeCycle),
        ("instanceType", paths.InstanceTypePath),
        ("publicHostname", paths.PublicHostnamePath),
        ("publicIp", paths.PublicIPPath),
        ("localHostname", paths.LocalHostnamePath),
        ("localIp", paths.LocalIPPath),
        ("availabilityZone", paths.AZPlacementPath),
    ]
    expandableDct = {}

    for label, path in labels:
        response = _request_metadata(path, endpoint)
        if response.status_code == 200:
            expandableDct[label] = response.text
            continue
        if response.status_code != 404:
            raise RuntimeError(f"received {response.status_code} status code on {path} path")
        expandableDct[label] = ""

    if len(expandableDct["availabilityZone"]) > 0:
        expandableDct["region"] = expandableDct["availabilityZone"][:-1]
    else:
        expandableDct["region"] = ""

    return metadata_types.NodeMetadata(**expandableDct)
