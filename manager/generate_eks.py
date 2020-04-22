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


# kublet config schema: https://github.com/kubernetes/kubernetes/blob/master/staging/src/k8s.io/kubelet/config/v1beta1/types.go
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
    if config["min_instances"] == 0:
        desired_capacity = 1
    else:
        desired_capacity = config["min_instances"]

    clusterconfig_settings = {
        "instanceType": config["instance_type"],
        "availabilityZones": config["availability_zones"],
        "volumeSize": config["instance_volume_size"],
        "minSize": config["min_instances"],
        "maxSize": config["max_instances"],
        "desiredCapacity": desired_capacity,
    }

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
        },
        "labels": {"nvidia.com/gpu": "true"},
        "taints": {"nvidia.com/gpu": "true:NoSchedule"},
    }

    return merge_override(nodegroup, gpu_settings)


def is_gpu(instance_type):
    return instance_type.startswith("g") or instance_type.startswith("p")


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

    eks = {
        "apiVersion": "eksctl.io/v1alpha5",
        "kind": "ClusterConfig",
        "metadata": {
            "name": cluster_config["cluster_name"],
            "region": cluster_config["region"],
            "version": "1.15",
        },
        "vpc": {"nat": {"gateway": "Disable"}},
        "availabilityZones": cluster_config["availability_zones"],
        "cloudWatch": {"clusterLogging": {"enableTypes": ["*"]}},
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
