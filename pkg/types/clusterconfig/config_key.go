/*
Copyright 2021 Cortex Labs, Inc.

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
	NodeGroupsKey                          = "node_groups"
	NodePoolsKey                           = "node_pools"
	InstanceTypeKey                        = "instance_type"
	AcceleratorTypeKey                     = "accelerator_type"
	AcceleratorsPerInstanceKey             = "accelerators_per_instance"
	MinInstancesKey                        = "min_instances"
	MaxInstancesKey                        = "max_instances"
	SpotKey                                = "spot"
	SpotConfigKey                          = "spot_config"
	InstanceDistributionKey                = "instance_distribution"
	OnDemandBaseCapacityKey                = "on_demand_base_capacity"
	OnDemandPercentageAboveBaseCapacityKey = "on_demand_percentage_above_base_capacity"
	InstanceVolumeSizeKey                  = "instance_volume_size"
	InstanceVolumeTypeKey                  = "instance_volume_type"
	InstanceVolumeIOPSKey                  = "instance_volume_iops"
	InstancePoolsKey                       = "instance_pools"
	MaxPriceKey                            = "max_price"

	NetworkKey                    = "network"
	SubnetKey                     = "subnet"
	TagsKey                       = "tags"
	ClusterNameKey                = "cluster_name"
	RegionKey                     = "region"
	AvailabilityZonesKey          = "availability_zones"
	SubnetsKey                    = "subnets"
	AvailabilityZoneKey           = "availability_zone"
	SubnetIDKey                   = "subnet_id"
	SSLCertificateARNKey          = "ssl_certificate_arn"
	CortexPolicyARNKey            = "cortex_policy_arn"
	IAMPolicyARNsKey              = "iam_policy_arns"
	BucketKey                     = "bucket"
	SubnetVisibilityKey           = "subnet_visibility"
	NATGatewayKey                 = "nat_gateway"
	APILoadBalancerSchemeKey      = "api_load_balancer_scheme"
	OperatorLoadBalancerSchemeKey = "operator_load_balancer_scheme"
	VPCCIDRKey                    = "vpc_cidr"
	TelemetryKey                  = "telemetry"

	// User facing string
	NodeGroupsUserKey                          = "node groups"
	SpotUserKey                                = "use spot instances"
	InstanceTypeUserKey                        = "instance type"
	AcceleratorTypeUserKey                     = "accelerator type"
	AcceleratorsPerInstanceUserKey             = "accelerators per instance"
	MinInstancesUserKey                        = "min instances"
	MaxInstancesUserKey                        = "max instances"
	InstanceVolumeSizeUserKey                  = "instance volume size (Gi)"
	InstanceVolumeTypeUserKey                  = "instance volume type"
	InstanceVolumeIOPSUserKey                  = "instance volume iops"
	InstanceDistributionUserKey                = "spot instance distribution"
	OnDemandBaseCapacityUserKey                = "spot on demand base capacity"
	OnDemandPercentageAboveBaseCapacityUserKey = "spot on demand percentage above base capacity"
	MaxPriceUserKey                            = "spot max price ($ per hour)"
	InstancePoolsUserKey                       = "spot instance pools"

	APIVersionUserKey                 = "cluster version"
	ClusterNameUserKey                = "cluster name"
	RegionUserKey                     = "aws region"
	AvailabilityZonesUserKey          = "availability zones"
	AvailabilityZoneUserKey           = "availability zone"
	SubnetsUserKey                    = "subnets"
	SubnetIDUserKey                   = "subnet id"
	TagsUserKey                       = "tags"
	SSLCertificateARNUserKey          = "ssl certificate arn"
	CortexPolicyARNUserKey            = "cortex policy arn"
	IAMPolicyARNsUserKey              = "iam policy arns"
	BucketUserKey                     = "s3 bucket"
	NetworkUserKey                    = "network"
	SubnetUserKey                     = "subnet"
	SubnetVisibilityUserKey           = "subnet visibility"
	NATGatewayUserKey                 = "nat gateway"
	APILoadBalancerSchemeUserKey      = "api load balancer scheme"
	OperatorLoadBalancerSchemeUserKey = "operator load balancer scheme"
	VPCCIDRUserKey                    = "vpc cidr"
	TelemetryUserKey                  = "telemetry"
)
