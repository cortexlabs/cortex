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

	"github.com/cortexlabs/cortex/pkg/consts"
	"github.com/cortexlabs/cortex/pkg/lib/aws"
	"github.com/cortexlabs/cortex/pkg/lib/errors"
	s "github.com/cortexlabs/cortex/pkg/lib/strings"
)

const (
	ErrInstanceTypeTooSmall               = "clusterconfig.instance_type_too_small"
	ErrMinInstancesGreaterThanMax         = "clusterconfig.min_instances_greater_than_max"
	ErrInstanceTypeNotSupportedInRegion   = "clusterconfig.instance_type_not_supported_in_region"
	ErrIncompatibleSpotInstanceTypeMemory = "clusterconfig.incompatible_spot_instance_type_memory"
	ErrIncompatibleSpotInstanceTypeCPU    = "clusterconfig.incompatible_spot_instance_type_cpu"
	ErrIncompatibleSpotInstanceTypeGPU    = "clusterconfig.incompatible_spot_instance_type_gpu"
	ErrSpotPriceGreaterThanTargetOnDemand = "clusterconfig.spot_price_greater_than_target_on_demand"
	ErrSpotPriceGreaterThanMaxPrice       = "clusterconfig.spot_price_greater_than_max_price"
	ErrInstanceTypeNotSupported           = "clusterconfig.instance_type_not_supported"
	ErrAtLeastOneInstanceDistribution     = "clusterconfig.at_least_one_instance_distribution"
	ErrNoCompatibleSpotInstanceFound      = "clusterconfig.no_compatible_spot_instance_found"
	ErrConfiguredWhenSpotIsNotEnabled     = "clusterconfig.configured_when_spot_is_not_enabled"
	ErrOnDemandBaseCapacityGreaterThanMax = "clusterconfig.on_demand_base_capacity_greater_than_max"
	ErrConfigCannotBeChangedOnUpdate      = "clusterconfig.config_cannot_be_changed_on_update"
	ErrInvalidAvailabilityZone            = "clusterconfig.invalid_availability_zone"
	ErrDidNotMatchStrictS3Regex           = "clusterconfig.did_not_match_strict_s3_regex"
	ErrS3RegionDiffersFromCluster         = "clusterconfig.s3_region_differs_from_cluster"
	ErrImageVersionMismatch               = "clusterconfig.image_version_mismatch"
	ErrInvalidInstanceType                = "clusterconfig.invalid_instance_type"
)

func ErrorInstanceTypeTooSmall() error {
	return errors.WithStack(&errors.CortexError{
		Kind:    ErrInstanceTypeTooSmall,
		Message: "Cortex does not support nano or micro instances - please specify a larger instance type",
	})
}

func ErrorMinInstancesGreaterThanMax(min int64, max int64) error {
	return errors.WithStack(&errors.CortexError{
		Kind:    ErrMinInstancesGreaterThanMax,
		Message: fmt.Sprintf("%s cannot be greater than %s (%d > %d)", MinInstancesKey, MaxInstancesKey, min, max),
	})
}

func ErrorInstanceTypeNotSupportedInRegion(instanceType string, region string) error {
	return errors.WithStack(&errors.CortexError{
		Kind:    ErrInstanceTypeNotSupportedInRegion,
		Message: fmt.Sprintf("%s instances are not available in %s", instanceType, region),
	})
}

func ErrorIncompatibleSpotInstanceTypeMemory(target aws.InstanceMetadata, suggested aws.InstanceMetadata) error {
	return errors.WithStack(&errors.CortexError{
		Kind:    ErrIncompatibleSpotInstanceTypeMemory,
		Message: fmt.Sprintf("all instances must have at least as much memory as %s (%s has %s memory, but %s only has %s memory)", target.Type, target.Type, target.Memory.String(), suggested.Type, suggested.Memory.String()),
	})
}

func ErrorIncompatibleSpotInstanceTypeCPU(target aws.InstanceMetadata, suggested aws.InstanceMetadata) error {
	return errors.WithStack(&errors.CortexError{
		Kind:    ErrIncompatibleSpotInstanceTypeCPU,
		Message: fmt.Sprintf("all instances must have at least as much CPU as %s (%s has %s CPU, but %s only has %s CPU)", target.Type, target.Type, target.CPU.String(), suggested.Type, suggested.CPU.String()),
	})
}

func ErrorIncompatibleSpotInstanceTypeGPU(target aws.InstanceMetadata, suggested aws.InstanceMetadata) error {
	return errors.WithStack(&errors.CortexError{
		Kind:    ErrIncompatibleSpotInstanceTypeGPU,
		Message: fmt.Sprintf("all instances must have at least as much GPU as %s (%s has %d GPU, but %s only has %d GPU)", target.Type, target.Type, target.GPU, suggested.Type, suggested.GPU),
	})
}

func ErrorSpotPriceGreaterThanTargetOnDemand(suggestedSpotPrice float64, target aws.InstanceMetadata, suggested aws.InstanceMetadata) error {
	return errors.WithStack(&errors.CortexError{
		Kind:    ErrSpotPriceGreaterThanTargetOnDemand,
		Message: fmt.Sprintf("%s will not be allocated because its current spot price is $%g which is greater than than %s's on-demand price of $%g", suggested.Type, suggestedSpotPrice, target.Type, target.Price),
	})
}

func ErrorSpotPriceGreaterThanMaxPrice(suggestedSpotPrice float64, maxPrice float64, suggested aws.InstanceMetadata) error {
	return errors.WithStack(&errors.CortexError{
		Kind:    ErrSpotPriceGreaterThanMaxPrice,
		Message: fmt.Sprintf("%s will not be allocated because its current spot price is $%g which is greater than the configured max price $%g", suggested.Type, suggestedSpotPrice, maxPrice),
	})
}

func ErrorInstanceTypeNotSupported(instanceType string) error {
	return errors.WithStack(&errors.CortexError{
		Kind:    ErrInstanceTypeNotSupported,
		Message: fmt.Sprintf("instance type %s is not supported", instanceType),
	})
}

func ErrorAtLeastOneInstanceDistribution(instanceType string, suggestion string, suggestions ...string) error {
	allSuggestions := append(suggestions, suggestion)
	message := strings.Join(allSuggestions, ", ")
	return errors.WithStack(&errors.CortexError{
		Kind:    ErrAtLeastOneInstanceDistribution,
		Message: fmt.Sprintf("at least one compatible instance type other than %s must be specified (suggestions: %s)", instanceType, message),
	})
}

func ErrorNoCompatibleSpotInstanceFound(instanceType string) error {
	return errors.WithStack(&errors.CortexError{
		Kind:    ErrNoCompatibleSpotInstanceFound,
		Message: fmt.Sprintf("unable to find compatible spot instance types for %s", instanceType),
	})
}

func ErrorConfiguredWhenSpotIsNotEnabled(configKey string) error {
	return errors.WithStack(&errors.CortexError{
		Kind:    ErrConfiguredWhenSpotIsNotEnabled,
		Message: fmt.Sprintf("%s cannot be specified unless spot is enabled", configKey),
	})
}

func ErrorOnDemandBaseCapacityGreaterThanMax(onDemandBaseCapacity int64, max int64) error {
	return errors.WithStack(&errors.CortexError{
		Kind:    ErrOnDemandBaseCapacityGreaterThanMax,
		Message: fmt.Sprintf("%s cannot be greater than %s (%d > %d)", OnDemandBaseCapacityKey, MaxInstancesKey, onDemandBaseCapacity, max),
	})
}

func ErrorConfigCannotBeChangedOnUpdate(configKey string, prevVal interface{}) error {
	return errors.WithStack(&errors.CortexError{
		Kind:    ErrConfigCannotBeChangedOnUpdate,
		Message: fmt.Sprintf("modifying %s in a running cluster is not supported, please set %s to its previous value: %s", configKey, configKey, s.UserStr(prevVal)),
	})
}

func ErrorInvalidAvailabilityZone(invalidZone string, validAvailabilityZones []string) error {
	return errors.WithStack(&errors.CortexError{
		Kind:    ErrInvalidAvailabilityZone,
		Message: fmt.Sprintf("%s is an invalid availability zone; please choose from the following valid zones %s", invalidZone, s.UserStrsOr(validAvailabilityZones)),
	})
}

func ErrorDidNotMatchStrictS3Regex() error {
	return errors.WithStack(&errors.CortexError{
		Kind:    ErrDidNotMatchStrictS3Regex,
		Message: "only lowercase alphanumeric characters and dashes are allowed, with no consecutive dashes and no leading or trailing dashes",
	})
}

func ErrorS3RegionDiffersFromCluster(bucketName string, bucketRegion string, clusterRegion string) error {
	return errors.WithStack(&errors.CortexError{
		Kind:    ErrS3RegionDiffersFromCluster,
		Message: fmt.Sprintf("the %s bucket is in %s, but your cluster is in %s; either change the region of your cluster to %s, use a bucket that is in %s, or remove your bucket configuration to allow cortex to make the bucket for you", bucketName, bucketRegion, clusterRegion, bucketRegion, clusterRegion),
	})
}

func ErrorImageVersionMismatch(image string, tag string) error {
	return errors.WithStack(&errors.CortexError{
		Kind:    ErrImageVersionMismatch,
		Message: fmt.Sprintf("the specified image (%s) has a tag (%s) which does not match the version of your CLI (%s); please update the image tag, remove the image from the cluster config file (to use the default value), or update your CLI by following the instructions at https://www.cortex.dev/install", image, tag, consts.CortexVersion),
	})
}

func ErrorInvalidInstanceType(instanceType string) error {
	return errors.WithStack(&errors.CortexError{
		Kind:    ErrInvalidInstanceType,
		Message: fmt.Sprintf("%s is not a valid instance type", instanceType),
	})
}
