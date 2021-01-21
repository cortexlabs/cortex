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

import boto3
import sys
import yaml
import os


def get_autoscaling_group():
    client = boto3.client("autoscaling", region_name=os.environ["CORTEX_REGION"])
    paginator = client.get_paginator("describe_auto_scaling_groups")
    page_iterator = paginator.paginate(PaginationConfig={"PageSize": 100})

    filtered_asgs = page_iterator.search(
        "AutoScalingGroups[?contains(Tags[?Key==`{}`].Value, `{}`) && Tags[?Key==`{}`].Value]".format(
            "alpha.eksctl.io/cluster-name",
            os.environ["CORTEX_CLUSTER_NAME"],
            "k8s.io/cluster-autoscaler/node-template/label/workload",
        )
    )
    asgs = list(filtered_asgs)
    if len(asgs) == 0:
        raise Exception(
            "unable to find autoscaling groups belong to cluster "
            + os.environ["CORTEX_CLUSTER_NAME"]
        )
    return asgs


def get_launch_template(launch_template_id):
    client = boto3.client("ec2", region_name=os.environ["CORTEX_REGION"])
    resp = client.describe_launch_template_versions(LaunchTemplateId=launch_template_id)
    return resp["LaunchTemplateVersions"][0]["LaunchTemplateData"]


def extract_nodegroup_name(asg):
    for tag in asg["Tags"]:
        if tag["Key"] == "eksctl.io/v1alpha2/nodegroup-name":
            return tag["Value"]
    raise Exception(
        "tag {} not set for autoscaling group {}".format(
            "eksctl.io/v1alpha2/nodegroup-name", asg["AutoScalingGroupName"]
        )
    )


def refresh_yaml(configmap_yaml_path, output_yaml_path):
    with open(configmap_yaml_path, "r") as f:
        cluster_configmap = yaml.safe_load(f)

    cluster_configmap_str = cluster_configmap["data"]["cluster.yaml"]
    cluster_config = yaml.safe_load(cluster_configmap_str)

    asgs = get_autoscaling_group()
    # only possible when backup is enabled

    if cluster_config["spot"] and cluster_config.get("spot_config", {}).get(
        "on_demand_backup", False
    ):
        if len(asgs) != 2:
            raise Exception(
                "expected 2 autoscaling groups but found {} autoscaling groups".format(len(asgs))
            )
        asg_names = set()
        for group in asgs:
            nodegroup_name = extract_nodegroup_name(group)
            if nodegroup_name == "ng-cortex-worker-spot":
                asg = group
            asg_names.add(nodegroup_name)
        if "ng-cortex-worker-on-demand" not in asg_names:
            raise Exception(
                "expected autoscaling group with tag eksctl.io/v1alpha2/nodegroup-name={}".format(
                    "ng-cortex-worker-on-demand"
                )
            )
        if "ng-cortex-worker-spot" not in asg_names:
            raise Exception(
                "expected autoscaling group with tag eksctl.io/v1alpha2/nodegroup-name={}".format(
                    "ng-cortex-worker-spot"
                )
            )
    elif cluster_config["spot"]:
        if len(asgs) != 1:
            raise Exception(
                "expected 1 autoscaling groups but found {} autoscaling groups".format(len(asgs))
            )
        if "ng-cortex-worker-spot" not in extract_nodegroup_name(asgs[0]):
            raise Exception(
                "unable to find autoscaling group with tag eksctl.io/v1alpha2/nodegroup-name={}".format(
                    "ng-cortex-worker-spot"
                )
            )
        asg = asgs[0]
    else:
        if len(asgs) != 1:
            raise Exception(
                "expected 1 autoscaling groups but found {} autoscaling groups".format(len(asgs))
            )
        if "ng-cortex-worker-on-demand" not in extract_nodegroup_name(asgs[0]):
            raise Exception(
                "unable to find autoscaling group with tag eksctl.io/v1alpha2/nodegroup-name={}".format(
                    "ng-cortex-worker-on-demand"
                )
            )
        asg = asgs[0]

    cluster_config["min_instances"] = asg["MinSize"]
    cluster_config["max_instances"] = asg["MaxSize"]

    if len(cluster_config.get("subnets", [])) == 0:
        cluster_config["availability_zones"] = asg["AvailabilityZones"]

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
        spot_config = {"on_demand_backup": len(asgs) == 2}
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
    refresh_yaml(configmap_yaml_path=sys.argv[1], output_yaml_path=sys.argv[2])
