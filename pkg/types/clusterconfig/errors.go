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
	s "github.com/cortexlabs/cortex/pkg/lib/strings"
)

type ErrorKind int

const (
	ErrUnknown ErrorKind = iota
	ErrInstanceTypeTooSmall
	ErrMinInstancesGreaterThanMax
	ErrInstanceTypeNotSupportedInRegion
	ErrIncompatibleSpotInstanceTypeMemory
	ErrIncompatibleSpotInstanceTypeCPU
	ErrIncompatibleSpotInstanceTypeGPU
	ErrSpotPriceGreaterThanTargetOnDemand
	ErrSpotPriceGreaterThanMaxPrice
	ErrInstanceTypeNotSupported
	ErrAtLeastOneInstanceDistribution
	ErrNoCompatibleSpotInstanceFound
	ErrConfiguredWhenSpotIsNotEnabled
	ErrOnDemandBaseCapacityGreaterThanMax
	ErrConfigCannotBeChangedOnUpdate
	ErrInvalidAvailabilityZone
	ErrDidNotMatchStrictS3Regex
	ErrS3RegionDiffersFromCluster
	ErrInvalidInstanceType
)

var _errorKinds = []string{
	"err_unknown",
	"err_instance_type_too_small",
	"err_min_instances_greater_than_max",
	"err_instance_type_not_supported_in_region",
	"err_incompatible_spot_instance_type_memory",
	"err_incompatible_spot_instance_type_cpu",
	"err_incompatible_spot_instance_type_gpu",
	"err_spot_price_greater_than_target_on_demand",
	"err_spot_price_greater_than_max_price",
	"err_instance_type_not_supported",
	"err_at_least_one_instance_distribution",
	"err_no_compatible_spot_instance_found",
	"err_configured_when_spot_is_not_enabled",
	"err_on_demand_base_capacity_greater_than_max",
	"err_config_cannot_be_changed_on_update",
	"err_invalid_availability_zone",
	"err_did_not_match_strict_s3_regex",
	"err_s3_region_differs_from_cluster",
	"err_invalid_instance_type",
}

var _ = [1]int{}[int(ErrInvalidInstanceType)-(len(_errorKinds)-1)] // Ensure list length matches

func (t ErrorKind) String() string {
	return _errorKinds[t]
}

// MarshalText satisfies TextMarshaler
func (t ErrorKind) MarshalText() ([]byte, error) {
	return []byte(t.String()), nil
}

// UnmarshalText satisfies TextUnmarshaler
func (t *ErrorKind) UnmarshalText(text []byte) error {
	enum := string(text)
	for i := 0; i < len(_errorKinds); i++ {
		if enum == _errorKinds[i] {
			*t = ErrorKind(i)
			return nil
		}
	}

	*t = ErrUnknown
	return nil
}

// UnmarshalBinary satisfies BinaryUnmarshaler
// Needed for msgpack
func (t *ErrorKind) UnmarshalBinary(data []byte) error {
	return t.UnmarshalText(data)
}

// MarshalBinary satisfies BinaryMarshaler
func (t ErrorKind) MarshalBinary() ([]byte, error) {
	return []byte(t.String()), nil
}

type Error struct {
	Kind    ErrorKind
	message string
}

func (e Error) Error() string {
	return e.message
}

func ErrorInstanceTypeTooSmall() error {
	return errors.WithStack(Error{
		Kind:    ErrInstanceTypeTooSmall,
		message: "Cortex does not support nano or micro instances - please specify a larger instance type",
	})
}

func ErrorMinInstancesGreaterThanMax(min int64, max int64) error {
	return errors.WithStack(Error{
		Kind:    ErrMinInstancesGreaterThanMax,
		message: fmt.Sprintf("%s cannot be greater than %s (%d > %d)", MinInstancesKey, MaxInstancesKey, min, max),
	})
}

func ErrorInstanceTypeNotSupportedInRegion(instanceType string, region string) error {
	return errors.WithStack(Error{
		Kind:    ErrInstanceTypeNotSupportedInRegion,
		message: fmt.Sprintf("%s instances are not available in %s", instanceType, region),
	})
}

func ErrorIncompatibleSpotInstanceTypeMemory(target aws.InstanceMetadata, suggested aws.InstanceMetadata) error {
	return errors.WithStack(Error{
		Kind:    ErrIncompatibleSpotInstanceTypeMemory,
		message: fmt.Sprintf("all instances must have at least as much memory as %s (%s has %s memory, but %s only has %s memory)", target.Type, target.Type, target.Memory.String(), suggested.Type, suggested.Memory.String()),
	})
}

func ErrorIncompatibleSpotInstanceTypeCPU(target aws.InstanceMetadata, suggested aws.InstanceMetadata) error {
	return errors.WithStack(Error{
		Kind:    ErrIncompatibleSpotInstanceTypeCPU,
		message: fmt.Sprintf("all instances must have at least as much CPU as %s (%s has %s CPU, but %s only has %s CPU)", target.Type, target.Type, target.CPU.String(), suggested.Type, suggested.CPU.String()),
	})
}

func ErrorIncompatibleSpotInstanceTypeGPU(target aws.InstanceMetadata, suggested aws.InstanceMetadata) error {
	return errors.WithStack(Error{
		Kind:    ErrIncompatibleSpotInstanceTypeGPU,
		message: fmt.Sprintf("all instances must have at least as much GPU as %s (%s has %d GPU, but %s only has %d GPU)", target.Type, target.Type, target.GPU, suggested.Type, suggested.GPU),
	})
}

func ErrorSpotPriceGreaterThanTargetOnDemand(suggestedSpotPrice float64, target aws.InstanceMetadata, suggested aws.InstanceMetadata) error {
	return errors.WithStack(Error{
		Kind:    ErrSpotPriceGreaterThanTargetOnDemand,
		message: fmt.Sprintf("%s will not be allocated because its current spot price is $%g which is greater than than %s's on-demand price of $%g", suggested.Type, suggestedSpotPrice, target.Type, target.Price),
	})
}

func ErrorSpotPriceGreaterThanMaxPrice(suggestedSpotPrice float64, maxPrice float64, suggested aws.InstanceMetadata) error {
	return errors.WithStack(Error{
		Kind:    ErrSpotPriceGreaterThanMaxPrice,
		message: fmt.Sprintf("%s will not be allocated because its current spot price is $%g which is greater than the configured max price $%g", suggested.Type, suggestedSpotPrice, maxPrice),
	})
}

func ErrorInstanceTypeNotSupported(instanceType string) error {
	return errors.WithStack(Error{
		Kind:    ErrInstanceTypeNotSupported,
		message: fmt.Sprintf("instance type %s is not supported", instanceType),
	})
}

func ErrorAtLeastOneInstanceDistribution(instanceType string, suggestion string, suggestions ...string) error {
	allSuggestions := append(suggestions, suggestion)
	message := strings.Join(allSuggestions, ", ")
	return errors.WithStack(Error{
		Kind:    ErrAtLeastOneInstanceDistribution,
		message: fmt.Sprintf("at least one compatible instance type other than %s must be specified (suggestions: %s)", instanceType, message),
	})
}

func ErrorNoCompatibleSpotInstanceFound(instanceType string) error {
	return errors.WithStack(Error{
		Kind:    ErrNoCompatibleSpotInstanceFound,
		message: fmt.Sprintf("unable to find compatible spot instance types for %s", instanceType),
	})
}

func ErrorConfiguredWhenSpotIsNotEnabled(configKey string) error {
	return errors.WithStack(Error{
		Kind:    ErrConfiguredWhenSpotIsNotEnabled,
		message: fmt.Sprintf("%s cannot be specified unless spot is enabled", configKey),
	})
}

func ErrorOnDemandBaseCapacityGreaterThanMax(onDemandBaseCapacity int64, max int64) error {
	return errors.WithStack(Error{
		Kind:    ErrOnDemandBaseCapacityGreaterThanMax,
		message: fmt.Sprintf("%s cannot be greater than %s (%d > %d)", OnDemandBaseCapacityKey, MaxInstancesKey, onDemandBaseCapacity, max),
	})
}

func ErrorConfigCannotBeChangedOnUpdate(configKey string, prevVal interface{}) error {
	return errors.WithStack(Error{
		Kind:    ErrConfigCannotBeChangedOnUpdate,
		message: fmt.Sprintf("modifying %s in a running cluster is not supported, please set %s to its previous value: %s", configKey, configKey, s.UserStr(prevVal)),
	})
}

func ErrorInvalidAvailabilityZone(invalidZone string, validAvailabilityZones []string) error {
	return errors.WithStack(Error{
		Kind:    ErrInvalidAvailabilityZone,
		message: fmt.Sprintf("%s is an invalid availability zone; please choose from the following valid zones %s", invalidZone, s.UserStrsOr(validAvailabilityZones)),
	})
}

func ErrorDidNotMatchStrictS3Regex() error {
	return errors.WithStack(Error{
		Kind:    ErrDidNotMatchStrictS3Regex,
		message: "only lowercase alphanumeric characters and dashes are allowed, with no consecutive dashes and no leading or trailing dashes",
	})
}

func ErrorS3RegionDiffersFromCluster(bucketName string, bucketRegion string, clusterRegion string) error {
	return errors.WithStack(Error{
		Kind:    ErrS3RegionDiffersFromCluster,
		message: fmt.Sprintf("the %s bucket is in %s, but your cluster is in %s; either change the region of your cluster to %s, use a bucket that is in %s, or remove your bucket configuration to allow cortex to make the bucket for you", bucketName, bucketRegion, clusterRegion, bucketRegion, clusterRegion),
	})
}

func ErrorInvalidInstanceType(instanceType string) error {
	return errors.WithStack(Error{
		Kind:    ErrInvalidInstanceType,
		message: fmt.Sprintf("%s is not a valid instance type", instanceType),
	})
}
