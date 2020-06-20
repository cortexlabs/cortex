# Copyright 2020 Cortex Labs, Inc.
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

import sys
import yaml
import os
import collections


# kubelet config schema: https://github.com/kubernetes/kubernetes/blob/master/staging/src/k8s.io/kubelet/config/v1beta1/types.go
def default_nodegroup(cluster_config):
    return {
        "ami": "auto",
        "iam": {"withAddonPolicies": {"autoScaler": True}},
        "privateNetworking": cluster_config.get("subnet_visibility", "public") != "public",
        "kubeletExtraConfig": {
            "kubeReserved": {"cpu": "150m", "memory": "300Mi", "ephemeral-storage": "1Gi"},
            "kubeReservedCgroup": "/kube-reserved",
            "systemReserved": {"cpu": "150m", "memory": "300Mi", "ephemeral-storage": "1Gi"},
            "evictionHard": {"memory.available": "200Mi", "nodefs.available": "5%"},
        },
    }


def merge_override(a, b):
    "merges b into a"
    for key in b:
        if key in a:
            if isinstance(a[key], dict) and isinstance(b[key], dict):
                merge_override(a[key], b[key])
            else:
                a[key] = b[key]
        else:
            a[key] = b[key]
    return a


def apply_worker_settings(nodegroup):
    worker_settings = {
        "name": "ng-cortex-worker-on-demand",
        "labels": {"workload": "true"},
        "taints": {"workload": "true:NoSchedule"},
        "tags": {
            "k8s.io/cluster-autoscaler/enabled": "true",
            "k8s.io/cluster-autoscaler/node-template/label/workload": "true",
        },
    }

    return merge_override(nodegroup, worker_settings)


def apply_clusterconfig(nodegroup, config):
    clusterconfig_settings = {
        "instanceType": config["instance_type"],
        "availabilityZones": config["availability_zones"],
        "volumeSize": config["instance_volume_size"],
        "minSize": config["min_instances"],
        "maxSize": config["max_instances"],
        "volumeType": config["instance_volume_type"],
        "desiredCapacity": 1 if config["min_instances"] == 0 else config["min_instances"],
    }
    # add iops to settings if volume_type is io1
    if config["instance_volume_type"] == "io1":
        clusterconfig_settings["volumeIOPS"] = config["instance_volume_iops"]

    return merge_override(nodegroup, clusterconfig_settings)


def apply_spot_settings(nodegroup, config):
    spot_settings = {
        "name": "ng-cortex-worker-spot",
        "instanceType": "mixed",
        "instancesDistribution": {
            "instanceTypes": config["spot_config"]["instance_distribution"],
            "onDemandBaseCapacity": config["spot_config"]["on_demand_base_capacity"],
            "onDemandPercentageAboveBaseCapacity": config["spot_config"][
                "on_demand_percentage_above_base_capacity"
            ],
            "maxPrice": config["spot_config"]["max_price"],
            "spotInstancePools": config["spot_config"]["instance_pools"],
        },
        "labels": {"lifecycle": "Ec2Spot"},
    }

    return merge_override(nodegroup, spot_settings)


def apply_gpu_settings(nodegroup):
    gpu_settings = {
        "tags": {
            "k8s.io/cluster-autoscaler/node-template/label/nvidia.com/gpu": "true",
            "k8s.io/cluster-autoscaler/node-template/taint/dedicated": "nvidia.com/gpu=true",
            "k8s.io/cluster-autoscaler/node-template/label/k8s.amazonaws.com/accelerator": "true",  # accepted values are GPU type such as nvidia-tesla-k80 but using "true" as a placeholder for now because the value doesn't matter for AWS cluster autoscaler
        },
        "labels": {
            "nvidia.com/gpu": "true",
            "k8s.amazonaws.com/accelerator": "true",  # accepted values are GPU type such as nvidia-tesla-k80 but using "true" as a placeholder for now because the value doesn't matter for AWS cluster autoscaler
        },
        "taints": {"nvidia.com/gpu": "true:NoSchedule"},
    }

    return merge_override(nodegroup, gpu_settings)


def is_gpu(instance_type):
    return instance_type.startswith("g") or instance_type.startswith("p")


def apply_inf_settings(nodegroup, cluster_config):
    instance_type = cluster_config["instance_type"]
    instance_region = cluster_config["region"]

    num_chips, hugepages_mem = get_inf_resources(instance_type)
    inf_settings = {
        "ami": get_ami_image(instance_region),
        "tags": {
            "k8s.io/cluster-autoscaler/node-template/label/aws.amazon.com/infa": "true",
            "k8s.io/cluster-autoscaler/node-template/taint/dedicated": "aws.amazon.com/infa=true",
            "k8s.io/cluster-autoscaler/node-template/resources/aws.amazon.com/infa": str(num_chips),
            "k8s.io/cluster-autoscaler/node-template/resources/hugepages-2Mi": hugepages_mem,
        },
        "labels": {"aws.amazon.com/infa": "true"},
        "taints": {"aws.amazon.com/infa": "true:NoSchedule"},
    }
    return merge_override(nodegroup, inf_settings)


def is_inf(instance_type):
    return instance_type.startswith("inf")


def get_inf_resources(instance_type):
    num_chips = 0
    if instance_type in ["inf1.xlarge", "inf1.2xlarge"]:
        num_chips = 1
    elif instance_type == "inf1.6xlarge":
        num_chips = 4
    elif instance_type == "inf1.24xlarge":
        num_chips = 16

    return num_chips, f"{128 * num_chips}Mi"


def get_ami_image(region):
    if region.startswith("us-east-1"):
        return "ami-07a7b48058cfe1a73"
    if region.startswith("us-west-2"):
        return "ami-00c8c8387d112425c"
    raise RuntimeError(f"ami image is in region {region} instead of 'us-east-1' or 'us-west-2'")


def generate_eks(cluster_config_path):
    with open(cluster_config_path, "r") as f:
        cluster_config = yaml.safe_load(f)

    operator_nodegroup = default_nodegroup(cluster_config)
    operator_settings = {
        "name": "ng-cortex-operator",
        "instanceType": "t3.medium",
        "availabilityZones": cluster_config["availability_zones"],
        "minSize": 1,
        "maxSize": 1,
        "desiredCapacity": 1,
    }
    operator_nodegroup = merge_override(operator_nodegroup, operator_settings)

    worker_nodegroup = default_nodegroup(cluster_config)
    apply_worker_settings(worker_nodegroup)

    apply_clusterconfig(worker_nodegroup, cluster_config)

    if cluster_config["spot"]:
        apply_spot_settings(worker_nodegroup, cluster_config)

    if is_gpu(cluster_config["instance_type"]):
        apply_gpu_settings(worker_nodegroup)

    if is_inf(cluster_config["instance_type"]):
        apply_inf_settings(worker_nodegroup, cluster_config)

    nat_gateway = "Disable"
    if cluster_config["nat_gateway"] == "single":
        nat_gateway = "Single"
    elif cluster_config["nat_gateway"] == "highly_available":
        nat_gateway = "HighlyAvailable"

    eks = {
        "apiVersion": "eksctl.io/v1alpha5",
        "kind": "ClusterConfig",
        "metadata": {
            "name": cluster_config["cluster_name"],
            "region": cluster_config["region"],
            "version": "1.16",
            "tags": cluster_config["tags"],
        },
        "vpc": {"nat": {"gateway": nat_gateway}},
        "availabilityZones": cluster_config["availability_zones"],
        "nodeGroups": [operator_nodegroup, worker_nodegroup],
    }

    if cluster_config.get("spot_config") is not None and cluster_config["spot_config"].get(
        "on_demand_backup", False
    ):
        backup_nodegroup = default_nodegroup(cluster_config)
        apply_worker_settings(backup_nodegroup)
        apply_clusterconfig(backup_nodegroup, cluster_config)
        if is_gpu(cluster_config["instance_type"]):
            apply_gpu_settings(backup_nodegroup)
        if is_inf(cluster_config["instance_type"]):
            apply_inf_settings(backup_nodegroup, cluster_config)

        backup_nodegroup["minSize"] = 0
        backup_nodegroup["desiredCapacity"] = 0

        eks["nodeGroups"].append(backup_nodegroup)

    print(yaml.dump(eks, Dumper=IgnoreAliases, default_flow_style=False, default_style=""))


class IgnoreAliases(yaml.Dumper):
    """By default, yaml dumper tries to compress yaml by annotating collections (lists and maps)
    and replacing subsequent identical collections with aliases. This class overrides the default
    behaviour to preserve the duplication of arrays.
    """

    def ignore_aliases(self, data):
        return True


if __name__ == "__main__":
    generate_eks(cluster_config_path=sys.argv[1])
