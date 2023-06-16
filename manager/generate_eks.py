# Copyright 2022 Cortex Labs, Inc.
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

import json
import click

from collections import namedtuple
import re
import yaml

K8S_VERSION = "1.26"
AMI_FAMILY = "AmazonLinux2"

ParsedInstanceType = namedtuple(
    "ParsedInstanceType", ["family", "generation", "capabilities", "size"]
)


def parse_instance_type(instance_type: str) -> ParsedInstanceType:
    parts = instance_type.split(".")
    if len(parts) != 2:
        raise ValueError(f"unexpected invalid instance type: {instance_type}")

    prefix = parts[0]
    size = parts[1]

    family = re.search("[a-z]*", prefix.lower()).group()
    generation = re.sub("\D", "", prefix.lower())
    capabilities = prefix[len(family) + len(generation) :]

    return ParsedInstanceType(family, generation, capabilities, size)


# kubelet config schema:
# https://github.com/kubernetes/kubernetes/blob/master/staging/src/k8s.io/kubelet/config/v1beta1/types.go
def default_nodegroup(cluster_config):
    partition = "aws"
    if "us-gov" in cluster_config["region"]:
        partition = "aws-us-gov"
    return {
        "iam": {
            "withAddonPolicies": {"autoScaler": True},
            "attachPolicyARNs": [
                f"arn:{partition}:iam::aws:policy/AmazonEKSWorkerNodePolicy",
                f"arn:{partition}:iam::aws:policy/AmazonEKS_CNI_Policy",
                f"arn:{partition}:iam::aws:policy/AmazonEC2ContainerRegistryReadOnly",
                f"arn:{partition}:iam::aws:policy/ElasticLoadBalancingFullAccess",
                cluster_config["cortex_policy_arn"],
            ]
            + cluster_config.get("iam_policy_arns", []),
        },
        "privateNetworking": cluster_config.get("subnet_visibility", "public") != "public",
        "kubeletExtraConfig": {
            "kubeReserved": {"cpu": "150m", "memory": "300Mi", "ephemeral-storage": "1Gi"},
            "kubeReservedCgroup": "/kube-reserved",
            "systemReserved": {"cpu": "150m", "memory": "300Mi", "ephemeral-storage": "1Gi"},
            "evictionHard": {"memory.available": "200Mi", "nodefs.available": "5%"},
            "registryPullQPS": 10,
        },
        "preBootstrapCommands": [
            "sudo yum install -y ipvsadm",
            "sudo modprobe ip_vs",  # IP virtual server
            "sudo modprobe ip_vs_rr",  # round robing load balancer
            "sudo modprobe ip_vs_lc",  # least connected load balancer
            "sudo modprobe ip_vs_wrr",  # weighted round robin load balancer
            "sudo modprobe ip_vs_sh",  # source-hashing load balancer
            "sudo modprobe nf_conntrack_ipv4",
        ],
        "overrideBootstrapCommand": "\n".join(
            [
                "#!/bin/bash",
                "source /var/lib/cloud/scripts/eksctl/bootstrap.helper.sh",
                f"/etc/eks/bootstrap.sh {cluster_config['cluster_name']} --container-runtime containerd --kubelet-extra-args \"--node-labels=${{NODE_LABELS}} --register-with-taints=${{NODE_TAINTS}}\"",
            ]
        ),
    }


def merge_override(a, b):
    "merges b into a"
    for key in b:
        if key in a:
            if isinstance(a[key], dict) and isinstance(b[key], dict):
                merge_override(a[key], b[key])
            elif isinstance(a[key], list) and isinstance(b[key], list):
                a[key] += b[key]
            else:
                a[key] = b[key]
        else:
            a[key] = b[key]
    return a


def apply_worker_settings(nodegroup, config):
    worker_settings = {
        "name": "cx-wd-" + config["name"],
        "asgSuspendProcesses": ["AZRebalance"],
        "labels": {"workload": "true"},
        "taints": [
            {
                "key": "workload",
                "value": "true",
                "effect": "NoSchedule",
            },
        ],
        "tags": {
            "k8s.io/cluster-autoscaler/enabled": "true",
            "k8s.io/cluster-autoscaler/node-template/label/workload": "true",
        },
    }

    return merge_override(nodegroup, worker_settings)


def apply_clusterconfig(nodegroup, config):
    clusterconfig_settings = {
        "instanceType": config["instance_type"],
        "volumeSize": config["instance_volume_size"],
        "minSize": config["min_instances"],
        "maxSize": config["max_instances"],
        "volumeType": config["instance_volume_type"],
        "desiredCapacity": 1 if config["min_instances"] == 0 else config["min_instances"],
    }
    # add iops to settings if volume_type is io1/gp3
    if config["instance_volume_type"] in ["io1", "gp3"]:
        clusterconfig_settings["volumeIOPS"] = config["instance_volume_iops"]
    if config["instance_volume_type"] == "gp3":
        clusterconfig_settings["volumeThroughput"] = config["instance_volume_throughput"]

    return merge_override(nodegroup, clusterconfig_settings)


def apply_spot_settings(nodegroup, config):
    spot_settings = {
        "name": "cx-ws-" + config["name"],
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
        "taints": [
            {
                "key": "nvidia.com/gpu",
                "value": "true",
                "effect": "NoSchedule",
            },
        ],
    }

    return merge_override(nodegroup, gpu_settings)


def is_gpu(instance_type):
    parsed_instance_type = parse_instance_type(instance_type)
    return parsed_instance_type.family in ["g", "p"]


def apply_inf_settings(nodegroup, config):
    instance_type = config["instance_type"]

    num_chips, num_hugepages = get_inf_resources(instance_type)
    inf_settings = {
        "tags": {
            "k8s.io/cluster-autoscaler/node-template/label/aws.amazon.com/neuron": "true",
            "k8s.io/cluster-autoscaler/node-template/taint/dedicated": "aws.amazon.com/neuron=true",
            "k8s.io/cluster-autoscaler/node-template/resources/aws.amazon.com/neuron": str(
                num_chips
            ),
            "k8s.io/cluster-autoscaler/node-template/resources/hugepages-2Mi": num_hugepages,
        },
        "labels": {"aws.amazon.com/neuron": "true"},
        "taints": [
            {
                "key": "aws.amazon.com/neuron",
                "value": "true",
                "effect": "NoSchedule",
            },
        ],
    }
    return merge_override(nodegroup, inf_settings)


def is_inf(instance_type):
    parsed_instance_type = parse_instance_type(instance_type)
    return parsed_instance_type.family == "inf"


def get_inf_resources(instance_type):
    num_chips = 0
    if instance_type in ["inf1.xlarge", "inf1.2xlarge"]:
        num_chips = 1
    elif instance_type == "inf1.6xlarge":
        num_chips = 4
    elif instance_type == "inf1.24xlarge":
        num_chips = 16

    return num_chips, f"{128 * num_chips}Mi"


def is_arm64(instance_type: str):
    parsed_instance_type = parse_instance_type(instance_type)
    return parsed_instance_type.family == "a" or "g" in parsed_instance_type.capabilities


def get_all_worker_nodegroups(ami_map: dict, cluster_config: dict) -> list:
    """
    Gets all node groups in EKS-dict format.
    """
    worker_nodegroups = []
    for ng in cluster_config["node_groups"]:
        worker_nodegroups.append(get_worker_nodegroup(ami_map, ng, cluster_config))
    return worker_nodegroups


def get_worker_nodegroup(ami_map: dict, nodegroup_config: dict, cluster_config: dict) -> dict:
    """
    Converts Cortex-dict nodegroup config to EKS-dict format.
    """
    worker_nodegroup = default_nodegroup(cluster_config)
    worker_nodegroup["ami"] = get_ami(ami_map, nodegroup_config["instance_type"])
    worker_nodegroup["amiFamily"] = AMI_FAMILY

    apply_worker_settings(worker_nodegroup, nodegroup_config)
    apply_clusterconfig(worker_nodegroup, nodegroup_config)

    if nodegroup_config["spot"]:
        apply_spot_settings(worker_nodegroup, nodegroup_config)

    if is_gpu(nodegroup_config["instance_type"]):
        apply_gpu_settings(worker_nodegroup)

    if is_inf(nodegroup_config["instance_type"]):
        apply_inf_settings(worker_nodegroup, nodegroup_config)

    return worker_nodegroup


def get_nodegroup_config_by_name(cluster_config: dict, ng_name: str) -> dict:
    """
    Gets a nodegroup in Cortex-dict format from Cortex cluster config.
    """
    for ng in cluster_config["node_groups"]:
        if ng["name"] == ng_name:
            return ng


def get_empty_eks_nodegroup(name: str) -> dict:
    """
    Gets an empty nodegroup in EKS-dict format that only has the Cortex nodegroup name filled out.
    """
    return {"name": name}


def get_ami(ami_map: dict, instance_type: str) -> str:
    if is_gpu(instance_type) or is_inf(instance_type):
        return ami_map["accelerated_amd64"]
    if is_arm64(instance_type):
        return ami_map["cpu_arm64"]
    return ami_map["cpu_amd64"]


@click.command()
@click.argument("cluster-config_file", type=click.File("r"))
@click.argument("ami-json-file", type=click.File("r"))
@click.option(
    "--add-cortex-node-groups",
    type=str,
    help="specific cortex nodegroups to add to the generated eks file; use this for existing clusters",
)
@click.option(
    "--remove-eks-node-groups",
    type=str,
    help="specific eks nodegroup stacks to add to the generated eks file; use this for existing clusters",
)
def generate_eks(
    cluster_config_file, ami_json_file, add_cortex_node_groups: str, remove_eks_node_groups: str
):
    cluster_config = yaml.safe_load(cluster_config_file)
    region = cluster_config["region"]
    name = cluster_config["cluster_name"]
    prometheus_instance_type = cluster_config["prometheus_instance_type"]
    ami_map = json.load(ami_json_file)[K8S_VERSION][region]

    eks = {
        "apiVersion": "eksctl.io/v1alpha5",
        "kind": "ClusterConfig",
        "metadata": {
            "name": name,
            "region": region,
            "version": K8S_VERSION,
        },
    }

    if add_cortex_node_groups:
        node_group_names = add_cortex_node_groups.split(",")
        eks["nodeGroups"] = []
        for node_group_name in node_group_names:
            nodegroup_config = get_nodegroup_config_by_name(cluster_config, node_group_name)
            eks["nodeGroups"].append(
                get_worker_nodegroup(ami_map, nodegroup_config, cluster_config)
            )
        click.echo(yaml.dump(eks, Dumper=IgnoreAliases, default_flow_style=False, default_style=""))
        return

    if remove_eks_node_groups:
        stacks_names = remove_eks_node_groups.split(",")
        eks["nodeGroups"] = []
        for stack_name in stacks_names:
            eks["nodeGroups"].append(get_empty_eks_nodegroup(stack_name))
        click.echo(yaml.dump(eks, Dumper=IgnoreAliases, default_flow_style=False, default_style=""))
        return

    operator_nodegroup = default_nodegroup(cluster_config)
    operator_settings = {
        "ami": get_ami(ami_map, "t3.medium"),
        "amiFamily": AMI_FAMILY,
        "name": "cx-operator",
        "instanceType": "t3.medium",
        "minSize": 2,
        "maxSize": 25,
        "desiredCapacity": 2,
        "volumeType": "gp3",
        "volumeSize": 20,
        "volumeIOPS": 3000,
        "volumeThroughput": 125,
        "labels": {"operator": "true"},
    }
    operator_nodegroup = merge_override(operator_nodegroup, operator_settings)

    prometheus_nodegroup = default_nodegroup(cluster_config)
    prometheus_settings = {
        "ami": get_ami(ami_map, prometheus_instance_type),
        "amiFamily": AMI_FAMILY,
        "name": "cx-prometheus",
        "instanceType": prometheus_instance_type,
        "minSize": 1,
        "maxSize": 1,
        "desiredCapacity": 1,
        "volumeType": "gp3",
        "volumeSize": 20,
        "volumeIOPS": 3000,
        "volumeThroughput": 125,
        "labels": {"prometheus": "true"},
        "taints": [
            {
                "key": "prometheus",
                "value": "true",
                "effect": "NoSchedule",
            },
        ],
    }
    prometheus_nodegroup = merge_override(prometheus_nodegroup, prometheus_settings)

    worker_nodegroups = get_all_worker_nodegroups(ami_map, cluster_config)

    nat_gateway = "Disable"
    if cluster_config["nat_gateway"] == "single":
        nat_gateway = "Single"
    elif cluster_config["nat_gateway"] == "highly_available":
        nat_gateway = "HighlyAvailable"

    eks = {
        "apiVersion": "eksctl.io/v1alpha5",
        "kind": "ClusterConfig",
        "metadata": {
            "name": name,
            "region": region,
            "version": K8S_VERSION,
            "tags": cluster_config["tags"],
        },
        "vpc": {"nat": {"gateway": nat_gateway}},
        "nodeGroups": [operator_nodegroup, prometheus_nodegroup] + worker_nodegroups,
        "addons": [
            {
                "name": "vpc-cni",
                "version": "1.12.6",
            },
        ],
    }

    if (
        len(cluster_config.get("availability_zones", [])) > 0
        and len(cluster_config.get("subnets", [])) == 0
    ):
        eks["availabilityZones"] = cluster_config["availability_zones"]

    if len(cluster_config.get("subnets", [])) > 0:
        eks_subnet_configs = {}
        for subnet_config in cluster_config["subnets"]:
            eks_subnet_configs[subnet_config["availability_zone"]] = {
                "id": subnet_config["subnet_id"]
            }

        if cluster_config.get("subnet_visibility", "public") == "private":
            eks["vpc"]["subnets"] = {"private": eks_subnet_configs}
        else:
            eks["vpc"]["subnets"] = {"public": eks_subnet_configs}

    if cluster_config.get("vpc_cidr", "") != "":
        eks["vpc"]["cidr"] = cluster_config["vpc_cidr"]

    click.echo(yaml.dump(eks, Dumper=IgnoreAliases, default_flow_style=False, default_style=""))


class IgnoreAliases(yaml.Dumper):
    """By default, yaml dumper tries to compress yaml by annotating collections (lists and maps)
    and replacing subsequent identical collections with aliases. This class overrides the default
    behaviour to preserve the duplication of arrays.
    """

    def ignore_aliases(self, data):
        return True


if __name__ == "__main__":
    generate_eks()
