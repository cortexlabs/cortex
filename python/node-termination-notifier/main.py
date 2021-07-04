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

import time
from typing import Union
import logging

import click
import zmq
from prometheus_client import Gauge, start_http_server

from termination.imds import (
    get_scheduled_maintainence_events,
    get_spot_interruption_event,
    get_rebalance_recommendation_event,
    get_metadata_info,
    ScheduledEventDetail,
    RebalanceRecommendation,
    InstanceAction,
    NodeMetadata,
)


def record_prom_event(
    gauge: Gauge,
    event: Union[ScheduledEventDetail, RebalanceRecommendation, InstanceAction],
    metadata_info: NodeMetadata,
):
    if isinstance(event, ScheduledEventDetail):
        action = "schedule"
    elif isinstance(event, RebalanceRecommendation):
        action = "rebalance"
    else:
        action = "termination"

    gauge.labels(
        metadata_info.instanceId,
        metadata_info.instanceLifeCycle,
        metadata_info.instanceType,
        action,
    )
    gauge.set(1)


def convert_metadata_info_to_json(metadata_info: NodeMetadata) -> dict:
    return {
        "instanceId": metadata_info.instanceId,
        "instanceLifeCycle": metadata_info.instanceLifeCycle,
        "instanceType": metadata_info.instanceType,
        "availabilityZone": metadata_info.availabilityZone,
        "region": metadata_info.region,
    }


def convert_scheduled_event_to_json(
    event: ScheduledEventDetail, metadata_info: NodeMetadata
) -> dict:
    return {
        "action": "schedule",
        "details": {
            "NotBefore": event.NotBefore,
            "Code": event.Code,
            "Description": event.Description,
            "EventId": event.EventId,
            "NotAfter": event.NotAfter,
            "State": event.State,
        },
        "metadata": convert_metadata_info_to_json(metadata_info),
    }


def convert_instance_action_event_to_json(
    event: InstanceAction, metadata_info: NodeMetadata
) -> dict:
    return {
        "action": "schedule",
        "details": {
            "action": event.action,
            "time": event.time,
        },
        "metadata": convert_metadata_info_to_json(metadata_info),
    }


def convert_rebalance_event_to_json(
    event: RebalanceRecommendation, metadata_info: NodeMetadata
) -> dict:
    return {
        "action": "schedule",
        "details": {
            "noticeTime": event.noticeTime,
        },
        "metadata": convert_metadata_info_to_json(metadata_info),
    }


@click.command(help="AWS Node Termination notifier script")
@click.argument("pub-port", type=int, envvar="PUB_PORT")
@click.argument("metrics-port", type=int, envvar="METRICS_PORT")
@click.option(
    "--query-endpoint",
    "-q",
    type=str,
    default="169.254.169.254",
    show_default=True,
    help="Endpoint for getting the AWS termination notifications",
)
def main(pub_port: int, metrics_port: int, query_endpoint: str):
    logging.basicConfig(level=logging.INFO)

    past_scheduled_maintainance_events = []
    past_spot_interruption_event = None
    past_rebalance_recommendation_event = None

    aws_node_termination = Gauge(
        name="aws_node_termination",
        documentation="AWS Node Termination notifier",
        labelnames=("instanceId", "instanceLifeCycle", "instanceType", "action"),
    )
    start_http_server(metrics_port)

    context = zmq.Context()
    socket = context.socket(zmq.PUB)
    socket.bind(f"tcp://127.0.0.1:{pub_port}")

    metadata_info: NodeMetadata = get_metadata_info(query_endpoint)
    logging.info(metadata_info)

    iterationPeriod = 5
    while True:
        scheduled_maintainance_events = get_scheduled_maintainence_events(query_endpoint)
        for event in scheduled_maintainance_events:
            if event not in past_scheduled_maintainance_events:
                logging.info(f"got event: {event} with {metadata_info}")
                record_prom_event(aws_node_termination, event, metadata_info)
                socket.send_json(convert_scheduled_event_to_json(event, metadata_info))
        past_scheduled_maintainance_events = scheduled_maintainance_events

        spot_interruption_event = get_spot_interruption_event(query_endpoint)
        if spot_interruption_event != past_spot_interruption_event:
            logging.info(f"got event: {spot_interruption_event} with {metadata_info}")
            record_prom_event(aws_node_termination, spot_interruption_event, metadata_info)
            socket.send_json(convert_instance_action_event_to_json(event, metadata_info))
        past_spot_interruption_event = spot_interruption_event

        rebalance_recommendation_event = get_rebalance_recommendation_event(query_endpoint)
        if rebalance_recommendation_event != past_rebalance_recommendation_event:
            logging.info(f"got event: {rebalance_recommendation_event} with {metadata_info}")
            record_prom_event(aws_node_termination, rebalance_recommendation_event, metadata_info)
            socket.send_json(convert_rebalance_event_to_json(event, metadata_info))
        past_rebalance_recommendation_event = rebalance_recommendation_event

        time.sleep(iterationPeriod)


if __name__ == "__main__":
    main()
