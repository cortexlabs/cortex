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

import (
	"fmt"
	"strings"

	"github.com/cortexlabs/cortex/pkg/lib/aws"
	"github.com/cortexlabs/cortex/pkg/lib/errors"
	"github.com/cortexlabs/cortex/pkg/lib/sets/strset"
	s "github.com/cortexlabs/cortex/pkg/lib/strings"
	"github.com/cortexlabs/cortex/pkg/types"
)

const (
	ErrInvalidRegion                          = "clusterconfig.invalid_region"
	ErrNoNodeGroupSpecified                   = "clusterconfig.no_nodegroup_specified"
	ErrMaxNumOfNodeGroupsReached              = "clusterconfig.max_num_of_nodegroups_reached"
	ErrDuplicateNodeGroupName                 = "clusterconfig.duplicate_nodegroup_name"
	ErrInstanceTypeTooSmall                   = "clusterconfig.instance_type_too_small"
	ErrMinInstancesGreaterThanMax             = "clusterconfig.min_instances_greater_than_max"
	ErrInstanceTypeNotSupportedInRegion       = "clusterconfig.instance_type_not_supported_in_region"
	ErrIncompatibleSpotInstanceTypeMemory     = "clusterconfig.incompatible_spot_instance_type_memory"
	ErrIncompatibleSpotInstanceTypeCPU        = "clusterconfig.incompatible_spot_instance_type_cpu"
	ErrIncompatibleSpotInstanceTypeGPU        = "clusterconfig.incompatible_spot_instance_type_gpu"
	ErrIncompatibleSpotInstanceTypeInf        = "clusterconfig.incompatible_spot_instance_type_inf"
	ErrSpotPriceGreaterThanTargetOnDemand     = "clusterconfig.spot_price_greater_than_target_on_demand"
	ErrSpotPriceGreaterThanMaxPrice           = "clusterconfig.spot_price_greater_than_max_price"
	ErrInstanceTypeNotSupported               = "clusterconfig.instance_type_not_supported"
	ErrARMInstancesNotSupported               = "clusterconfig.arm_instances_not_supported"
	ErrAtLeastOneInstanceDistribution         = "clusterconfig.at_least_one_instance_distribution"
	ErrNoCompatibleSpotInstanceFound          = "clusterconfig.no_compatible_spot_instance_found"
	ErrConfiguredWhenSpotIsNotEnabled         = "clusterconfig.configured_when_spot_is_not_enabled"
	ErrOnDemandBaseCapacityGreaterThanMax     = "clusterconfig.on_demand_base_capacity_greater_than_max"
	ErrConfigCannotBeChangedOnUpdate          = "clusterconfig.config_cannot_be_changed_on_update"
	ErrInvalidAvailabilityZone                = "clusterconfig.invalid_availability_zone"
	ErrAvailabilityZoneSpecifiedTwice         = "clusterconfig.availability_zone_specified_twice"
	ErrUnsupportedAvailabilityZone            = "clusterconfig.unsupported_availability_zone"
	ErrNotEnoughValidDefaultAvailibilityZones = "clusterconfig.not_enough_valid_default_availability_zones"
	ErrNoNATGatewayWithSubnets                = "clusterconfig.no_nat_gateway_with_subnets"
	ErrSpecifyOneOrNone                       = "clusterconfig.specify_one_or_none"
	ErrDependentFieldMustBeSpecified          = "clusterconfig.dependent_field_must_be_specified"
	ErrFieldConfigurationDependentOnCondition = "clusterconfig.field_configuration_dependent_on_condition"
	ErrDidNotMatchStrictS3Regex               = "clusterconfig.did_not_match_strict_s3_regex"
	ErrNATRequiredWithPrivateSubnetVisibility = "clusterconfig.nat_required_with_private_subnet_visibility"
	ErrS3RegionDiffersFromCluster             = "clusterconfig.s3_region_differs_from_cluster"
	ErrInvalidInstanceType                    = "clusterconfig.invalid_instance_type"
	ErrIOPSNotSupported                       = "clusterconfig.iops_not_supported"
	ErrIOPSTooLarge                           = "clusterconfig.iops_too_large"
	ErrCantOverrideDefaultTag                 = "clusterconfig.cant_override_default_tag"
	ErrSSLCertificateARNNotFound              = "clusterconfig.ssl_certificate_arn_not_found"
	ErrIAMPolicyARNNotFound                   = "clusterconfig.iam_policy_arn_not_found"
	ErrProviderMismatch                       = "clusterconfig.provider_mismatch"

	ErrGCPInvalidProjectID                        = "clusterconfig.gcp_invalid_project_id"
	ErrGCPProjectMustBeSpecified                  = "clusterconfig.gcp_project_must_be_specified"
	ErrGCPInvalidZone                             = "clusterconfig.gcp_invalid_zone"
	ErrGCPNoNodePoolSpecified                     = "clusterconfig.gcp_no_nodepool_specified"
	ErrGCPMaxNumOfNodePoolsReached                = "clusterconfig.gcp_max_num_of_nodepools_reached"
	ErrGCPDuplicateNodePoolName                   = "clusterconfig.gcp_duplicate_nodepool_name"
	ErrGCPInvalidInstanceType                     = "clusterconfig.gcp_invalid_instance_type"
	ErrGCPInvalidAcceleratorType                  = "clusterconfig.gcp_invalid_accelerator_type"
	ErrGCPIncompatibleInstanceTypeWithAccelerator = "clusterconfig.gcp_incompatible_instance_type_with_accelerator"
)

func ErrorInvalidRegion(region string) error {
	return errors.WithStack(&errors.Error{
		Kind:    ErrInvalidRegion,
		Message: fmt.Sprintf("%s is not a valid AWS region, or is an AWS region which is not supported by AWS EKS; please choose one of the following regions: %s", s.UserStr(region), strings.Join(aws.EKSSupportedRegions.SliceSorted(), ", ")),
	})
}

func ErrorNoNodeGroupSpecified() error {
	return errors.WithStack(&errors.Error{
		Kind:    ErrNoNodeGroupSpecified,
		Message: "no nodegroup was specified; please specify at least 1 nodegroup",
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

func ErrorInstanceTypeTooSmall() error {
	return errors.WithStack(&errors.Error{
		Kind:    ErrInstanceTypeTooSmall,
		Message: "cortex does not support nano or micro instances - please specify a larger instance type",
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

func ErrorInstanceTypeNotSupported(instanceType string) error {
	return errors.WithStack(&errors.Error{
		Kind:    ErrInstanceTypeNotSupported,
		Message: fmt.Sprintf("instance type %s is not supported", instanceType),
	})
}

func ErrorARMInstancesNotSupported(instanceType string) error {
	return errors.WithStack(&errors.Error{
		Kind:    ErrARMInstancesNotSupported,
		Message: fmt.Sprintf("ARM-based instances (including %s) are not supported", instanceType),
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

func ErrorConfigCannotBeChangedOnUpdate() error {
	return errors.WithStack(&errors.Error{
		Kind:    ErrConfigCannotBeChangedOnUpdate,
		Message: fmt.Sprintf("in a running cluster, only the %s and %s fields in the %s section can be modified", MinInstancesKey, MaxInstancesKey, NodeGroupsKey),
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
		Message: fmt.Sprintf("the %s bucket is in %s, but your cluster is in %s; either change the region of your cluster to %s, use a bucket that is in %s, or remove your bucket configuration to allow cortex to make the bucket for you", bucketName, bucketRegion, clusterRegion, bucketRegion, clusterRegion),
	})
}

func ErrorInvalidInstanceType(instanceType string) error {
	return errors.WithStack(&errors.Error{
		Kind:    ErrInvalidInstanceType,
		Message: fmt.Sprintf("%s is not a valid instance type", instanceType),
	})
}

func ErrorIOPSNotSupported(volumeType VolumeType) error {
	return errors.WithStack(&errors.Error{
		Kind:    ErrIOPSNotSupported,
		Message: fmt.Sprintf("IOPS cannot be configured for volume type %s; set `%s: %s` or remove `%s` from your cluster configuration file", volumeType, InstanceVolumeTypeKey, IO1VolumeType, InstanceVolumeIOPSKey),
	})
}

func ErrorIOPSTooLarge(iops int64, volumeSize int64) error {
	return errors.WithStack(&errors.Error{
		Kind:    ErrIOPSTooLarge,
		Message: fmt.Sprintf("%s (%d) cannot be more than 50 times larger than %s (%d); increase `%s` or decrease `%s` in your cluster configuration file", InstanceVolumeIOPSKey, iops, InstanceVolumeSizeKey, volumeSize, InstanceVolumeSizeKey, InstanceVolumeIOPSKey),
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

func ErrorProviderMismatch(expectedProvider types.ProviderType, actualProvider string) error {
	return errors.WithStack(&errors.Error{
		Kind:    ErrProviderMismatch,
		Message: fmt.Sprintf("expected \"%s\" provider, but got \"%s\"; please use `cortex cluster` commands for aws clusters, and `cortex cluster-gcp` commands for gcp clusters", expectedProvider, actualProvider),
	})
}

func ErrorGCPInvalidProjectID(projectID string) error {
	return errors.WithStack(&errors.Error{
		Kind:    ErrGCPInvalidProjectID,
		Message: fmt.Sprintf("invalid project ID '%s'", projectID),
	})
}

func ErrorGCPProjectMustBeSpecified() error {
	return errors.WithStack(&errors.Error{
		Kind:    ErrGCPProjectMustBeSpecified,
		Message: fmt.Sprintf("please provide a cluster configuration file which specifies `%s` (e.g. via `--config cluster.yaml`) or enable prompts (i.e. omit the `--yes` flag)", ProjectKey),
	})
}

func ErrorGCPInvalidZone(zone string, suggestedZones ...string) error {
	errorMessage := fmt.Sprintf("invalid zone '%s'", zone)
	if len(suggestedZones) == 1 {
		errorMessage += fmt.Sprintf("; use zone '%s' instead", suggestedZones[0])
	}
	if len(suggestedZones) > 1 {
		errorMessage += fmt.Sprintf("; choose one of the following zones: %s", s.StrsOr(suggestedZones))
	}
	return errors.WithStack(&errors.Error{
		Kind:    ErrGCPInvalidZone,
		Message: errorMessage,
	})
}

func ErrorGCPNoNodePoolSpecified() error {
	return errors.WithStack(&errors.Error{
		Kind:    ErrGCPNoNodePoolSpecified,
		Message: "no nodepool was specified; please specify at least 1 nodepool",
	})
}

func ErrorGCPMaxNumOfNodePoolsReached(maxNodePools int64) error {
	return errors.WithStack(&errors.Error{
		Kind:    ErrGCPMaxNumOfNodePoolsReached,
		Message: fmt.Sprintf("cannot have more than %d nodepools", maxNodePools),
	})
}

func ErrorGCPDuplicateNodePoolName(duplicateNpName string) error {
	return errors.WithStack(&errors.Error{
		Kind:    ErrGCPDuplicateNodePoolName,
		Message: fmt.Sprintf("cannot have multiple nodepools with the same name (%s)", duplicateNpName),
	})
}

func ErrorGCPInvalidInstanceType(instanceType string, suggestedInstanceTypes ...string) error {
	errorMessage := fmt.Sprintf("invalid instance type '%s'", instanceType)
	if len(suggestedInstanceTypes) == 1 {
		errorMessage += fmt.Sprintf("; use instance type '%s' instead", suggestedInstanceTypes[0])
	}
	if len(suggestedInstanceTypes) > 1 {
		errorMessage += fmt.Sprintf("; choose one of the following instance types: %s", s.StrsOr(suggestedInstanceTypes))
	}
	return errors.WithStack(&errors.Error{
		Kind:    ErrGCPInvalidInstanceType,
		Message: errorMessage,
	})
}

func ErrorGCPInvalidAcceleratorType(acceleratorType string, zone string, suggestedAcceleratorsInZone []string, suggestedZonesForAccelerator []string) error {
	errorMessage := fmt.Sprintf("invalid accelerator type '%s'", acceleratorType)
	if len(suggestedAcceleratorsInZone) > 0 {
		errorMessage += fmt.Sprintf("\n\nfor zone %s, the following accelerators are available: %s", zone, s.StrsAnd(suggestedAcceleratorsInZone))
	}
	if len(suggestedZonesForAccelerator) > 0 {
		errorMessage += fmt.Sprintf("\n\nfor accelerator %s, the following zones are accepted: %s", acceleratorType, s.StrsAnd(suggestedZonesForAccelerator))
	}
	return errors.WithStack(&errors.Error{
		Kind:    ErrGCPInvalidAcceleratorType,
		Message: errorMessage,
	})
}

func ErrorGCPIncompatibleInstanceTypeWithAccelerator(instanceType, acceleratorType, zone string, compatibleInstances []string) error {
	return errors.WithStack(&errors.Error{
		Kind:    ErrGCPIncompatibleInstanceTypeWithAccelerator,
		Message: fmt.Sprintf("instance type %s is incompatible with the %s accelerator; the following instance types are compatible with the %s accelerator in zone %s: %s", instanceType, acceleratorType, acceleratorType, zone, s.StrsOr(compatibleInstances)),
	})
}
