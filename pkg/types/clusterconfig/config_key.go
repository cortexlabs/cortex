/*
Copyright 2022 Cortex Labs, Inc.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package clusterconfig

const (
	BucketKey     = "bucket"
	ClusterUIDKey = "cluster_uid"

	ClusterNameKey                         = "cluster_name"
	RegionKey                              = "region"
	PrometheusInstanceTypeKey              = "prometheus_instance_type"
	NodeGroupsKey                          = "node_groups"
	InstanceTypeKey                        = "instance_type"
	AcceleratorTypeKey                     = "accelerator_type"
	AcceleratorsPerInstanceKey             = "accelerators_per_instance"
	MinInstancesKey                        = "min_instances"
	MaxInstancesKey                        = "max_instances"
	PriorityKey                            = "priority"
	SpotKey                                = "spot"
	SpotConfigKey                          = "spot_config"
	InstanceDistributionKey                = "instance_distribution"
	OnDemandBaseCapacityKey                = "on_demand_base_capacity"
	OnDemandPercentageAboveBaseCapacityKey = "on_demand_percentage_above_base_capacity"
	InstanceVolumeSizeKey                  = "instance_volume_size"
	InstanceVolumeTypeKey                  = "instance_volume_type"
	InstanceVolumeIOPSKey                  = "instance_volume_iops"
	InstanceVolumeThroughputKey            = "instance_volume_throughput"
	InstancePoolsKey                       = "instance_pools"
	MaxPriceKey                            = "max_price"
	NetworkKey                             = "network"
	SubnetKey                              = "subnet"
	TagsKey                                = "tags"
	AvailabilityZonesKey                   = "availability_zones"
	SubnetsKey                             = "subnets"
	AvailabilityZoneKey                    = "availability_zone"
	SubnetIDKey                            = "subnet_id"
	SSLCertificateARNKey                   = "ssl_certificate_arn"
	CortexPolicyARNKey                     = "cortex_policy_arn"
	IAMPolicyARNsKey                       = "iam_policy_arns"
	SubnetVisibilityKey                    = "subnet_visibility"
	NATGatewayKey                          = "nat_gateway"
	APILoadBalancerSchemeKey               = "api_load_balancer_scheme"
	OperatorLoadBalancerSchemeKey          = "operator_load_balancer_scheme"
	APILoadBalancerCIDRWhiteListKey        = "api_load_balancer_cidr_white_list"
	OperatorLoadBalancerCIDRWhiteListKey   = "operator_load_balancer_cidr_white_list"
	VPCCIDRKey                             = "vpc_cidr"
	AccountIDKey                           = "account_id"
	TelemetryKey                           = "telemetry"
)
