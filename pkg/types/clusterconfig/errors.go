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

import (
	"fmt"
	"strings"

	"github.com/cortexlabs/cortex/pkg/consts"
	"github.com/cortexlabs/cortex/pkg/lib/aws"
	"github.com/cortexlabs/cortex/pkg/lib/errors"
	"github.com/cortexlabs/cortex/pkg/lib/sets/strset"
	s "github.com/cortexlabs/cortex/pkg/lib/strings"
)

const (
	ErrInvalidProvider                        = "clusterconfig.invalid_provider"
	ErrInvalidLegacyProvider                  = "clusterconfig.invalid_legacy_provider"
	ErrDisallowedField                        = "clusterconfig.disallowed_field"
	ErrInvalidRegion                          = "clusterconfig.invalid_region"
	ErrNodeGroupMaxInstancesIsZero            = "clusterconfig.node_group_max_instances_is_zero"
	ErrMaxNumOfNodeGroupsReached              = "clusterconfig.max_num_of_nodegroups_reached"
	ErrDuplicateNodeGroupName                 = "clusterconfig.duplicate_nodegroup_name"
	ErrMaxNodesToAddOnClusterUp               = "clusterconfig.max_nodes_to_add_on_cluster_up"
	ErrMaxNodesToAddOnClusterConfigure        = "clusterconfig.max_nodes_to_add_on_cluster_configure"
	ErrInstanceTypeTooSmall                   = "clusterconfig.instance_type_too_small"
	ErrMinInstancesGreaterThanMax             = "clusterconfig.min_instances_greater_than_max"
	ErrInstanceTypeNotSupportedInRegion       = "clusterconfig.instance_type_not_supported_in_region"
	ErrIncompatibleSpotInstanceTypeMemory     = "clusterconfig.incompatible_spot_instance_type_memory"
	ErrIncompatibleSpotInstanceTypeCPU        = "clusterconfig.incompatible_spot_instance_type_cpu"
	ErrIncompatibleSpotInstanceTypeGPU        = "clusterconfig.incompatible_spot_instance_type_gpu"
	ErrIncompatibleSpotInstanceTypeInf        = "clusterconfig.incompatible_spot_instance_type_inf"
	ErrSpotPriceGreaterThanTargetOnDemand     = "clusterconfig.spot_price_greater_than_target_on_demand"
	ErrSpotPriceGreaterThanMaxPrice           = "clusterconfig.spot_price_greater_than_max_price"
	ErrInstanceTypeNotSupportedByCortex       = "clusterconfig.instance_type_not_supported_by_cortex"
	ErrAMDGPUInstancesNotSupported            = "clusterconfig.amd_gpu_instances_not_supported"
	ErrGPUInstancesNotSupported               = "clusterconfig.gpu_instance_not_supported"
	ErrInferentiaInstancesNotSupported        = "clusterconfig.inferentia_instances_not_supported"
	ErrMacInstancesNotSupported               = "clusterconfig.mac_instances_not_supported"
	ErrFPGAInstancesNotSupported              = "clusterconfig.fpga_instances_not_supported"
	ErrAlevoInstancesNotSupported             = "clusterconfig.alevo_instances_not_supported"
	ErrGaudiInstancesNotSupported             = "clusterconfig.gaudi_instances_not_supported"
	ErrTrainiumInstancesNotSupported          = "clusterconfig.trainium_instances_not_supported"
	ErrAtLeastOneInstanceDistribution         = "clusterconfig.at_least_one_instance_distribution"
	ErrNoCompatibleSpotInstanceFound          = "clusterconfig.no_compatible_spot_instance_found"
	ErrConfiguredWhenSpotIsNotEnabled         = "clusterconfig.configured_when_spot_is_not_enabled"
	ErrOnDemandBaseCapacityGreaterThanMax     = "clusterconfig.on_demand_base_capacity_greater_than_max"
	ErrInvalidAvailabilityZone                = "clusterconfig.invalid_availability_zone"
	ErrAvailabilityZoneSpecifiedTwice         = "clusterconfig.availability_zone_specified_twice"
	ErrUnsupportedAvailabilityZone            = "clusterconfig.unsupported_availability_zone"
	ErrNotEnoughValidDefaultAvailibilityZones = "clusterconfig.not_enough_valid_default_availability_zones"
	ErrNoNATGatewayWithSubnets                = "clusterconfig.no_nat_gateway_with_subnets"
	ErrSubnetMaskOutOfRange                   = "clusterconfig.subnet_mask_out_of_range"
	ErrConfigCannotBeChangedOnConfigure       = "clusterconfig.config_cannot_be_changed_on_configure"
	ErrNodeGroupCanOnlyBeScaled               = "clusterconfig.node_group_can_only_be_scaled"
	ErrSpecifyOneOrNone                       = "clusterconfig.specify_one_or_none"
	ErrSpecifyTwoOrNone                       = "clusterconfig.specify_two_or_none"
	ErrDependentFieldMustBeSpecified          = "clusterconfig.dependent_field_must_be_specified"
	ErrFieldConfigurationDependentOnCondition = "clusterconfig.field_configuration_dependent_on_condition"
	ErrDidNotMatchStrictS3Regex               = "clusterconfig.did_not_match_strict_s3_regex"
	ErrNATRequiredWithPrivateSubnetVisibility = "clusterconfig.nat_required_with_private_subnet_visibility"
	ErrS3RegionDiffersFromCluster             = "clusterconfig.s3_region_differs_from_cluster"
	ErrIOPSNotSupported                       = "clusterconfig.iops_not_supported"
	ErrThroughputNotSupported                 = "clusterconfig.throughput_not_supported"
	ErrIOPSTooSmall                           = "clusterconfig.iops_too_small"
	ErrIOPSTooLarge                           = "clusterconfig.iops_too_large"
	ErrIOPSToVolumeSizeRatio                  = "clusterconfig.iops_to_volume_size_ratio"
	ErrIOPSToThroughputRatio                  = "clusterconfig.iops_to_throughput_ratio"
	ErrCantOverrideDefaultTag                 = "clusterconfig.cant_override_default_tag"
	ErrSSLCertificateARNNotFound              = "clusterconfig.ssl_certificate_arn_not_found"
	ErrIAMPolicyARNNotFound                   = "clusterconfig.iam_policy_arn_not_found"
)

func ErrorInvalidProvider(providerStr string) error {
	return errors.WithStack(&errors.Error{
		Kind:    ErrInvalidProvider,
		Message: fmt.Sprintf("\"%s\" is not a supported provider; only aws is supported, so the provider field may be removed from your cluster configuration file", providerStr),
	})
}

func ErrorInvalidLegacyProvider(providerStr string) error {
	return errors.WithStack(&errors.Error{
		Kind:    ErrInvalidLegacyProvider,
		Message: fmt.Sprintf("the %s provider is no longer supported on cortex v%s; only aws is supported, so the provider field may be removed from your cluster configuration file", providerStr, consts.CortexVersionMinor),
	})
}

func ErrorDisallowedField(field string) error {
	return errors.WithStack(&errors.Error{
		Kind:    ErrDisallowedField,
		Message: fmt.Sprintf("the %s field cannot be configured by the user", field),
	})
}

func ErrorInvalidRegion(region string) error {
	return errors.WithStack(&errors.Error{
		Kind:    ErrInvalidRegion,
		Message: fmt.Sprintf("%s is not a valid AWS region, or is an AWS region which is not supported by AWS EKS; please choose one of the following regions: %s", s.UserStr(region), strings.Join(aws.EKSSupportedRegions.SliceSorted(), ", ")),
	})
}

func ErrorNodeGroupMaxInstancesIsZero() error {
	return errors.WithStack(&errors.Error{
		Kind:    ErrNodeGroupMaxInstancesIsZero,
		Message: fmt.Sprintf("nodegroups cannot be created with `%s` set to 0 (but `%s` can be scaled to 0 after the nodegroup has been created)", MaxInstancesKey, MaxInstancesKey),
	})
}

func ErrorMaxNumOfNodeGroupsReached(maxNodeGroups int64) error {
	return errors.WithStack(&errors.Error{
		Kind:    ErrMaxNumOfNodeGroupsReached,
		Message: fmt.Sprintf("cannot have more than %d nodegroups", maxNodeGroups),
	})
}

func ErrorDuplicateNodeGroupName(duplicateNgName string) error {
	return errors.WithStack(&errors.Error{
		Kind:    ErrDuplicateNodeGroupName,
		Message: fmt.Sprintf("cannot have multiple nodegroups with the same name (%s)", duplicateNgName),
	})
}

func ErrorMaxNodesToAddOnClusterUp(requestedNodes, maxNodes int64) error {
	return errors.WithStack(&errors.Error{
		Kind:    ErrMaxNodesToAddOnClusterUp,
		Message: fmt.Sprintf("cannot create a cluster with %d instances (at most %d instances can be created initially); reduce %s for your nodegroups (you may add additional instances via the `cortex cluster configure` command after your cluster has been created)", requestedNodes, maxNodes, MinInstancesKey),
	})
}

func ErrorMaxNodesToAddOnClusterConfigure(requestedNodes, currentNodes, maxNodes int64) error {
	return errors.WithStack(&errors.Error{
		Kind:    ErrMaxNodesToAddOnClusterConfigure,
		Message: fmt.Sprintf("cannot add %d instances to your cluster (you requested %d total instances, but your cluster currently has %d instances); only %d instances can be added at time, so reduce the sum of %s across all nodegroups by %d", requestedNodes-currentNodes, requestedNodes, currentNodes, maxNodes, MinInstancesKey, requestedNodes-currentNodes-maxNodes),
	})
}

func ErrorInstanceTypeTooSmall(instanceType string) error {
	return errors.WithStack(&errors.Error{
		Kind:    ErrInstanceTypeTooSmall,
		Message: fmt.Sprintf("%s: cortex does not support nano or micro instances - please specify a larger instance type", instanceType),
	})
}

func ErrorMinInstancesGreaterThanMax(min int64, max int64) error {
	return errors.WithStack(&errors.Error{
		Kind:    ErrMinInstancesGreaterThanMax,
		Message: fmt.Sprintf("%s cannot be greater than %s (%d > %d)", MinInstancesKey, MaxInstancesKey, min, max),
	})
}

func ErrorInstanceTypeNotSupportedInRegion(instanceType string, region string) error {
	return errors.WithStack(&errors.Error{
		Kind:    ErrInstanceTypeNotSupportedInRegion,
		Message: fmt.Sprintf("%s instances are not available in %s", instanceType, region),
	})
}

func ErrorIncompatibleSpotInstanceTypeMemory(target aws.InstanceMetadata, suggested aws.InstanceMetadata) error {
	return errors.WithStack(&errors.Error{
		Kind:    ErrIncompatibleSpotInstanceTypeMemory,
		Message: fmt.Sprintf("all instances must have at least as much memory as %s (%s has %s memory, but %s only has %s memory)", target.Type, target.Type, target.Memory.String(), suggested.Type, suggested.Memory.String()),
	})
}

func ErrorIncompatibleSpotInstanceTypeCPU(target aws.InstanceMetadata, suggested aws.InstanceMetadata) error {
	return errors.WithStack(&errors.Error{
		Kind:    ErrIncompatibleSpotInstanceTypeCPU,
		Message: fmt.Sprintf("all instances must have at least as much CPU as %s (%s has %s CPU, but %s only has %s CPU)", target.Type, target.Type, target.CPU.String(), suggested.Type, suggested.CPU.String()),
	})
}

func ErrorIncompatibleSpotInstanceTypeGPU(target aws.InstanceMetadata, suggested aws.InstanceMetadata) error {
	return errors.WithStack(&errors.Error{
		Kind:    ErrIncompatibleSpotInstanceTypeGPU,
		Message: fmt.Sprintf("all instances must have at least as much GPU as %s (%s has %d GPU, but %s only has %d GPU)", target.Type, target.Type, target.GPU, suggested.Type, suggested.GPU),
	})
}

func ErrorIncompatibleSpotInstanceTypeInf(suggested aws.InstanceMetadata) error {
	return errors.WithStack(&errors.Error{
		Kind:    ErrIncompatibleSpotInstanceTypeInf,
		Message: fmt.Sprintf("all instances must have at least 1 Inferentia chip, but %s doesn't have any", suggested.Type),
	})
}

func ErrorSpotPriceGreaterThanTargetOnDemand(spotPrice float64, target aws.InstanceMetadata, suggested aws.InstanceMetadata) error {
	return errors.WithStack(&errors.Error{
		Kind:    ErrSpotPriceGreaterThanTargetOnDemand,
		Message: fmt.Sprintf("%s will not be allocated because its current spot price is $%g which is greater than %s's on-demand price of $%g", suggested.Type, spotPrice, target.Type, target.Price),
	})
}

func ErrorSpotPriceGreaterThanMaxPrice(suggestedSpotPrice float64, maxPrice float64, suggested aws.InstanceMetadata) error {
	return errors.WithStack(&errors.Error{
		Kind:    ErrSpotPriceGreaterThanMaxPrice,
		Message: fmt.Sprintf("%s will not be allocated because its current spot price is $%g which is greater than the configured max price $%g", suggested.Type, suggestedSpotPrice, maxPrice),
	})
}

func ErrorInstanceTypeNotSupportedByCortex(instanceType string) error {
	return errors.WithStack(&errors.Error{
		Kind:    ErrInstanceTypeNotSupportedByCortex,
		Message: fmt.Sprintf("instance type %s is not supported by cortex", instanceType),
	})
}

func ErrorAMDGPUInstancesNotSupported(instanceType string) error {
	return errors.WithStack(&errors.Error{
		Kind:    ErrAMDGPUInstancesNotSupported,
		Message: fmt.Sprintf("AMD GPU instances (including %s) are not supported by cortex", instanceType),
	})
}

func ErrorGPUInstancesNotSupported(instanceType string) error {
	return errors.WithStack(&errors.Error{
		Kind:    ErrGPUInstancesNotSupported,
		Message: fmt.Sprintf("GPU instances (including %s) are not supported", instanceType),
	})
}

func ErrorInferentiaInstancesNotSupported(instanceType string) error {
	return errors.WithStack(&errors.Error{
		Kind:    ErrInferentiaInstancesNotSupported,
		Message: fmt.Sprintf("Inferentia instances (including %s) are not supported", instanceType),
	})
}

func ErrorMacInstancesNotSupported(instanceType string) error {
	return errors.WithStack(&errors.Error{
		Kind:    ErrMacInstancesNotSupported,
		Message: fmt.Sprintf("mac instances (including %s) are not supported by cortex", instanceType),
	})
}

func ErrorFPGAInstancesNotSupported(instanceType string) error {
	return errors.WithStack(&errors.Error{
		Kind:    ErrFPGAInstancesNotSupported,
		Message: fmt.Sprintf("FPGA instances (including %s) are not supported by cortex", instanceType),
	})
}

func ErrorAlevoInstancesNotSupported(instanceType string) error {
	return errors.WithStack(&errors.Error{
		Kind:    ErrAlevoInstancesNotSupported,
		Message: fmt.Sprintf("Alevo instances (including %s) are not supported by cortex", instanceType),
	})
}

func ErrorGaudiInstancesNotSupported(instanceType string) error {
	return errors.WithStack(&errors.Error{
		Kind:    ErrGaudiInstancesNotSupported,
		Message: fmt.Sprintf("Gaudi instances (including %s) are not supported by cortex", instanceType),
	})
}

func ErrorTrainiumInstancesNotSupported(instanceType string) error {
	return errors.WithStack(&errors.Error{
		Kind:    ErrTrainiumInstancesNotSupported,
		Message: fmt.Sprintf("Trainium instances (including %s) are not supported by cortex", instanceType),
	})
}

func ErrorConfiguredWhenSpotIsNotEnabled(configKey string) error {
	return errors.WithStack(&errors.Error{
		Kind:    ErrConfiguredWhenSpotIsNotEnabled,
		Message: fmt.Sprintf("%s cannot be specified unless spot is enabled (to enable spot instances, set `%s: true` in your cluster configuration file)", configKey, SpotKey),
	})
}

func ErrorOnDemandBaseCapacityGreaterThanMax(onDemandBaseCapacity int64, max int64) error {
	return errors.WithStack(&errors.Error{
		Kind:    ErrOnDemandBaseCapacityGreaterThanMax,
		Message: fmt.Sprintf("%s cannot be greater than %s (%d > %d)", OnDemandBaseCapacityKey, MaxInstancesKey, onDemandBaseCapacity, max),
	})
}

func ErrorInvalidAvailabilityZone(userZone string, allZones strset.Set, region string) error {
	return errors.WithStack(&errors.Error{
		Kind:    ErrInvalidAvailabilityZone,
		Message: fmt.Sprintf("%s is not an availability zone in %s; please choose from the following availability zones: %s", userZone, region, s.StrsOr(allZones.SliceSorted())),
	})
}

func ErrorAvailabilityZoneSpecifiedTwice(zone string) error {
	return errors.WithStack(&errors.Error{
		Kind:    ErrAvailabilityZoneSpecifiedTwice,
		Message: fmt.Sprintf("availability zone \"%s\" is specified twice", zone),
	})
}

func ErrorUnsupportedAvailabilityZone(userZone string, instanceType string, instanceTypes ...string) error {
	msg := fmt.Sprintf("the %s availability zone does not support EKS and the %s instance type; please choose a different availability zone, instance type, or region", userZone, instanceType)

	if len(instanceTypes) > 0 {
		allInstanceTypes := append([]string{instanceType}, instanceTypes...)
		msg = fmt.Sprintf("the %s availability zone does not support EKS and %s instance types; please choose a different availability zone, instance types, or region", userZone, s.StrsAnd(allInstanceTypes))
	}

	return errors.WithStack(&errors.Error{
		Kind:    ErrUnsupportedAvailabilityZone,
		Message: msg,
	})
}

func ErrorNotEnoughDefaultSupportedZones(region string, validZones strset.Set, instanceType string, instanceTypes ...string) error {
	areNoStr := "are no"
	if len(validZones) > 0 {
		areNoStr = "aren't enough"
	}

	msg := fmt.Sprintf("there %s availability zones in %s which support EKS and the %s instance type; please choose a different instance type or a different region", areNoStr, region, instanceType)

	if len(instanceTypes) > 0 {
		allInstanceTypes := append([]string{instanceType}, instanceTypes...)
		msg = fmt.Sprintf("there %s availability zones in %s which support EKS and the %s instance types; please choose different instance types or a different region", areNoStr, region, s.StrsAnd(allInstanceTypes))
	}

	return errors.WithStack(&errors.Error{
		Kind:    ErrNotEnoughValidDefaultAvailibilityZones,
		Message: msg,
	})
}

func ErrorNoNATGatewayWithSubnets() error {
	return errors.WithStack(&errors.Error{
		Kind:    ErrNoNATGatewayWithSubnets,
		Message: fmt.Sprintf("nat gateway cannot be automatically created when specifying subnets for your cluster; please unset %s or %s", NATGatewayKey, SubnetsKey),
	})
}

func ErrorSubnetMaskOutOfRange(requestedMaskSize, minMaskSize, maxMaskSize int) error {
	return errors.WithStack(&errors.Error{
		Kind:    ErrSubnetMaskOutOfRange,
		Message: fmt.Sprintf("invalid network size /%d; the network size must be between /%d and /%d", requestedMaskSize, minMaskSize, maxMaskSize),
	})
}

func ErrorConfigCannotBeChangedOnConfigure() error {
	return errors.WithStack(&errors.Error{
		Kind:    ErrConfigCannotBeChangedOnConfigure,
		Message: fmt.Sprintf("in a running cluster, only %s can be modified", s.StrsAnd([]string{NodeGroupsKey, SSLCertificateARNKey, OperatorLoadBalancerCIDRWhiteListKey, APILoadBalancerCIDRWhiteListKey})),
	})
}

func ErrorNodeGroupCanOnlyBeScaled() error {
	return errors.WithStack(&errors.Error{
		Kind:    ErrNodeGroupCanOnlyBeScaled,
		Message: "in a running cluster, nodegroups can only be scaled, added, or deleted",
	})
}

func ErrorSpecifyOneOrNone(fieldName1 string, fieldName2 string, fieldNames ...string) error {
	fieldNames = append([]string{fieldName1, fieldName2}, fieldNames...)

	message := fmt.Sprintf("specify exactly one or none of the following fields: %s", s.StrsAnd(fieldNames))
	if len(fieldNames) == 2 {
		message = fmt.Sprintf("cannot specify both %s and %s; specify only one (or neither)", fieldNames[0], fieldNames[1])
	}

	return errors.WithStack(&errors.Error{
		Kind:    ErrSpecifyOneOrNone,
		Message: message,
	})
}

func ErrorSpecifyTwoOrNone(fieldName1 string, fieldName2 string, fieldNames ...string) error {
	fieldNames = append([]string{fieldName1, fieldName2}, fieldNames...)
	return errors.WithStack(&errors.Error{
		Kind:    ErrSpecifyTwoOrNone,
		Message: fmt.Sprintf("specify exactly two or none of the following fields: %s", s.StrsAnd(fieldNames)),
	})
}

func ErrorDependentFieldMustBeSpecified(configuredField string, dependencyField string) error {
	return errors.WithStack(&errors.Error{
		Kind:    ErrDependentFieldMustBeSpecified,
		Message: fmt.Sprintf("%s must be specified when configuring %s", dependencyField, configuredField),
	})
}

func ErrorFieldConfigurationDependentOnCondition(configuredField string, configuredFieldValue string, dependencyField string, dependencyFieldValue string) error {
	return errors.WithStack(&errors.Error{
		Kind:    ErrFieldConfigurationDependentOnCondition,
		Message: fmt.Sprintf("cannot set %s = %s when %s = %s", configuredField, configuredFieldValue, dependencyField, dependencyFieldValue),
	})
}

func ErrorDidNotMatchStrictS3Regex() error {
	return errors.WithStack(&errors.Error{
		Kind:    ErrDidNotMatchStrictS3Regex,
		Message: "only lowercase alphanumeric characters and dashes are allowed, with no consecutive dashes and no leading or trailing dashes",
	})
}

func ErrorNATRequiredWithPrivateSubnetVisibility() error {
	return errors.WithStack(&errors.Error{
		Kind:    ErrNATRequiredWithPrivateSubnetVisibility,
		Message: fmt.Sprintf("a nat gateway is required when `%s: %s` is specified; either set %s to %s or %s, or set %s to %s", SubnetVisibilityKey, PrivateSubnetVisibility, NATGatewayKey, s.UserStr(SingleNATGateway), s.UserStr(HighlyAvailableNATGateway), SubnetVisibilityKey, s.UserStr(PublicSubnetVisibility)),
	})
}

func ErrorS3RegionDiffersFromCluster(bucketName string, bucketRegion string, clusterRegion string) error {
	return errors.WithStack(&errors.Error{
		Kind:    ErrS3RegionDiffersFromCluster,
		Message: fmt.Sprintf("the %s bucket already exists but is in %s (your cluster is in %s); either change the region of your cluster to %s or delete your bucket to allow cortex to create the bucket for you in %s", bucketName, bucketRegion, clusterRegion, bucketRegion, clusterRegion),
	})
}

func ErrorIOPSNotSupported(volumeType VolumeType) error {
	return errors.WithStack(&errors.Error{
		Kind:    ErrIOPSNotSupported,
		Message: fmt.Sprintf("IOPS cannot be configured for volume type %s; set `%s: %s`, `%s: %s`, or remove `%s` from your cluster configuration file", volumeType, InstanceVolumeTypeKey, IO1VolumeType, InstanceVolumeTypeKey, GP3VolumeType, InstanceVolumeIOPSKey),
	})
}

func ErrorThroughputNotSupported(volumeType VolumeType) error {
	return errors.WithStack(&errors.Error{
		Kind:    ErrThroughputNotSupported,
		Message: fmt.Sprintf("throughput cannot be configured for volume type %s; set `%s: %s` or remove `%s` from your cluster configuration file", volumeType, InstanceVolumeTypeKey, GP3VolumeType, InstanceVolumeThroughputKey),
	})
}

func ErrorIOPSTooSmall(volumeType VolumeType, iops, minIOPS int64) error {
	return errors.WithStack(&errors.Error{
		Kind:    ErrIOPSTooSmall,
		Message: fmt.Sprintf("for %s volume type, %s (%d) cannot be smaller than %d", volumeType, InstanceVolumeIOPSKey, iops, minIOPS),
	})
}

func ErrorIOPSTooLarge(volumeType VolumeType, iops, maxIOPS int64) error {
	return errors.WithStack(&errors.Error{
		Kind:    ErrIOPSTooLarge,
		Message: fmt.Sprintf("for %s volume type, %s (%d) cannot be larger than %d", volumeType, InstanceVolumeIOPSKey, iops, maxIOPS),
	})
}

func ErrorIOPSToVolumeSizeRatio(volumeType VolumeType, ratio, iops int64, volumeSize int64) error {
	return errors.WithStack(&errors.Error{
		Kind:    ErrIOPSToVolumeSizeRatio,
		Message: fmt.Sprintf("for %s volume type, %s (%d) cannot be more than %d times larger than %s (%d); increase `%s` or decrease `%s` in your cluster configuration file", volumeType, InstanceVolumeIOPSKey, iops, ratio, InstanceVolumeSizeKey, volumeSize, InstanceVolumeSizeKey, InstanceVolumeIOPSKey),
	})
}

func ErrorIOPSToThroughputRatio(volumeType VolumeType, ratio, iops, throughput int64) error {
	return errors.WithStack(&errors.Error{
		Kind:    ErrIOPSToThroughputRatio,
		Message: fmt.Sprintf("for %s volume type, %s (%d) must be at least %d times larger than %s (%d); decrease `%s` or increase `%s` in your cluster configuration file", volumeType, InstanceVolumeIOPSKey, iops, ratio, InstanceVolumeThroughputKey, throughput, InstanceVolumeThroughputKey, InstanceVolumeIOPSKey),
	})
}

func ErrorCantOverrideDefaultTag() error {
	return errors.WithStack(&errors.Error{
		Kind:    ErrCantOverrideDefaultTag,
		Message: fmt.Sprintf("the \"%s\" tag cannot be overridden (it is set by default, and will always be equal to your cluster name)", ClusterNameTag),
	})
}

func ErrorSSLCertificateARNNotFound(sslCertificateARN string, region string) error {
	return errors.WithStack(&errors.Error{
		Kind:    ErrSSLCertificateARNNotFound,
		Message: fmt.Sprintf("unable to find the specified ssl certificate in %s: %s", region, sslCertificateARN),
	})
}

func ErrorIAMPolicyARNNotFound(policyARN string) error {
	return errors.WithStack(&errors.Error{
		Kind:    ErrIAMPolicyARNNotFound,
		Message: fmt.Sprintf("unable to find iam policy %s", policyARN),
	})
}
