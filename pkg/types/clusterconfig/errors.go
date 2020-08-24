/*
Copyright 2020 Cortex Labs, Inc.

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
)

const (
	ErrInvalidRegion                          = "clusterconfig.invalid_region"
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
	ErrAtLeastOneInstanceDistribution         = "clusterconfig.at_least_one_instance_distribution"
	ErrNoCompatibleSpotInstanceFound          = "clusterconfig.no_compatible_spot_instance_found"
	ErrConfiguredWhenSpotIsNotEnabled         = "clusterconfig.configured_when_spot_is_not_enabled"
	ErrOnDemandBaseCapacityGreaterThanMax     = "clusterconfig.on_demand_base_capacity_greater_than_max"
	ErrConfigCannotBeChangedOnUpdate          = "clusterconfig.config_cannot_be_changed_on_update"
	ErrInvalidAvailabilityZone                = "clusterconfig.invalid_availability_zone"
	ErrUnsupportedAvailabilityZone            = "clusterconfig.unsupported_availability_zone"
	ErrNotEnoughValidDefaultAvailibilityZones = "clusterconfig.not_enough_valid_default_availability_zones"
	ErrDidNotMatchStrictS3Regex               = "clusterconfig.did_not_match_strict_s3_regex"
	ErrNATRequiredWithPrivateSubnetVisibility = "clusterconfig.nat_required_with_private_subnet_visibility"
	ErrS3RegionDiffersFromCluster             = "clusterconfig.s3_region_differs_from_cluster"
	ErrInvalidInstanceType                    = "clusterconfig.invalid_instance_type"
	ErrIOPSNotSupported                       = "clusterconfig.iops_not_supported"
	ErrIOPSTooLarge                           = "clusterconfig.iops_too_large"
	ErrCantOverrideDefaultTag                 = "clusterconfig.cant_override_default_tag"
	ErrSSLCertificateARNNotFound              = "clusterconfig.ssl_certificate_arn_not_found"
)

func ErrorInvalidRegion(region string) error {
	return errors.WithStack(&errors.Error{
		Kind:    ErrInvalidRegion,
		Message: fmt.Sprintf("%s is not a valid AWS region, or is an AWS region which is not supported by AWS EKS; please choose one of the following regions: %s", s.UserStr(region), strings.Join(aws.EKSSupportedRegions.SliceSorted(), ", ")),
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

func ErrorConfigCannotBeChangedOnUpdate(configKey string, prevVal interface{}) error {
	return errors.WithStack(&errors.Error{
		Kind:    ErrConfigCannotBeChangedOnUpdate,
		Message: fmt.Sprintf("modifying %s in a running cluster is not supported, please set %s to its previous value (%s)", configKey, configKey, s.UserStr(prevVal)),
	})
}

func ErrorInvalidAvailabilityZone(userZone string, allZones strset.Set, region string) error {
	return errors.WithStack(&errors.Error{
		Kind:    ErrInvalidAvailabilityZone,
		Message: fmt.Sprintf("%s is not an availability zone in %s; please choose from the following availability zones: %s", s.UserStr(userZone), region, s.UserStrsOr(allZones.SliceSorted())),
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
		Message: fmt.Sprintf("the \"%s\" tag cannot be overridden (it is set by default, and it must always be equal to your cluster name)", ClusterNameTag),
	})
}

func ErrorSSLCertificateARNNotFound(sslCertificateARN string, region string) error {
	return errors.WithStack(&errors.Error{
		Kind:    ErrSSLCertificateARNNotFound,
		Message: fmt.Sprintf("unable to find the specified ssl certificate in %s: %s", region, sslCertificateARN),
	})
}
