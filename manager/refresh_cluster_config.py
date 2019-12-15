# Copyright 2019 Cortex Labs, Inc.
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

import boto3
import sys
import yaml
import os


def get_autoscaling_group():
    client = boto3.client("autoscaling", region_name=os.environ["CORTEX_REGION"])
    paginator = client.get_paginator("describe_auto_scaling_groups")
    page_iterator = paginator.paginate(PaginationConfig={"PageSize": 100})

    filtered_asgs = page_iterator.search(
        "AutoScalingGroups[?contains(Tags[?Key==`{}`].Value, `{}`)]|[?contains(Tags[?Key==`{}`].Value, `{}`)]".format(
            "alpha.eksctl.io/cluster-name",
            os.environ["CORTEX_CLUSTER_NAME"],
            "alpha.eksctl.io/nodegroup-name",
            "ng-cortex-worker",
        )
    )
    asgs = list(filtered_asgs)
    if len(asgs) != 1:
        raise Exception("found {} matching autoscaling groups, expected only one".format(len(asgs)))
    return asgs[0]


def get_launch_template(launch_template_id):
    client = boto3.client("ec2", region_name=os.environ["CORTEX_REGION"])
    resp = client.describe_launch_template_versions(LaunchTemplateId=launch_template_id)
    return resp["LaunchTemplateVersions"][0]["LaunchTemplateData"]


def refresh_yaml(input_yaml_path, output_yaml_path):
    with open(input_yaml_path, "r") as f:
        config = yaml.safe_load(f)
    asg = get_autoscaling_group()
    cluster_config_str = config["data"]["cluster.yaml"]
    cluster_config = yaml.safe_load(cluster_config_str)
    cluster_config["min_instances"] = asg["MinSize"]
    cluster_config["max_instances"] = asg["MaxSize"]

    if asg.get("MixedInstancesPolicy") is not None:
        launch_template = get_launch_template(
            asg["MixedInstancesPolicy"]["LaunchTemplate"]["LaunchTemplateSpecification"][
                "LaunchTemplateId"
            ]
        )
    else:
        launch_template = get_launch_template(asg["LaunchTemplate"]["LaunchTemplateId"])

    cluster_config["instance_type"] = launch_template["InstanceType"]

    if launch_template.get("BlockDeviceMappings"):
        cluster_config["instance_volume_size"] = launch_template["BlockDeviceMappings"][0]["Ebs"][
            "VolumeSize"
        ]
    else:
        cluster_config["instance_volume_size"] = 20  # AWS volume default

    if asg.get("LaunchTemplate") is not None:
        cluster_config["spot"] = False
        cluster_config["spot_config"] = None

    if asg.get("MixedInstancesPolicy") is not None:
        mixed_instance_policy = asg["MixedInstancesPolicy"]
        cluster_config["spot"] = True
        spot_config = {}
        instances_distribution_metadata = mixed_instance_policy["InstancesDistribution"]
        spot_config["on_demand_base_capacity"] = instances_distribution_metadata[
            "OnDemandBaseCapacity"
        ]
        spot_config["on_demand_percentage_above_base_capacity"] = instances_distribution_metadata[
            "OnDemandPercentageAboveBaseCapacity"
        ]
        spot_config["max_price"] = float(instances_distribution_metadata["SpotMaxPrice"])
        spot_config["instance_pools"] = instances_distribution_metadata["SpotInstancePools"]

        instance_distribution = [
            node["InstanceType"] for node in mixed_instance_policy["LaunchTemplate"]["Overrides"]
        ]
        spot_config["instance_distribution"] = instance_distribution

        cluster_config["spot_config"] = spot_config

    with open(output_yaml_path, "w") as f:
        yaml.dump(cluster_config, f)


if __name__ == "__main__":
    refresh_yaml(sys.argv[1], sys.argv[2])
