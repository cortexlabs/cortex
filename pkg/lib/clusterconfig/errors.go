/*
Copyright 2019 Cortex Labs, Inc.

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
)

type ErrorKind int

const (
	ErrUnknown ErrorKind = iota
	ErrInstanceTypeTooSmall
	ErrInvalidAWSCredentials
	ErrMinInstancesGreaterThanMax
	ErrInstanceTypeNotSupportedInRegion
	ErrIncompatibleSpotInstanceTypeMemory
	ErrIncompatibleSpotInstanceTypeCPU
	ErrIncompatibleSpotInstanceTypeGPU
	ErrGPUInstanceTypeNotSupported
	ErrAtLeastOneInstanceDistribution
	ErrNoCompatibleSpotInstanceFound
	ErrConfiguredWhenSpotIsNotEnabled
	ErrInvalidInstanceType
)

var (
	errorKinds = []string{
		"err_unknown",
		"err_instance_type_too_small",
		"err_invalid_aws_credentials",
		"err_min_instances_greater_than_max",
		"err_instance_type_not_supported_in_region",
		"err_incompatible_spot_instance_type_memory",
		"err_incompatible_spot_instance_type_cpu",
		"err_incompatible_spot_instance_type_gpu",
		"err_gpu_instance_type_not_supported",
		"err_at_least_one_instance_distribution",
		"err_no_compatible_spot_instance_found",
		"err_configured_when_spot_is_not_enabled",
		"err_invalid_instance_type",
	}
)

var _ = [1]int{}[int(ErrInvalidInstanceType)-(len(errorKinds)-1)] // Ensure list length matches

func (t ErrorKind) String() string {
	return errorKinds[t]
}

// MarshalText satisfies TextMarshaler
func (t ErrorKind) MarshalText() ([]byte, error) {
	return []byte(t.String()), nil
}

// UnmarshalText satisfies TextUnmarshaler
func (t *ErrorKind) UnmarshalText(text []byte) error {
	enum := string(text)
	for i := 0; i < len(errorKinds); i++ {
		if enum == errorKinds[i] {
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
	return Error{
		Kind:    ErrInstanceTypeTooSmall,
		message: "Cortex does not support nano, micro, or small instances - please specify a larger instance type",
	}
}

func ErrorInvalidAWSCredentials() error {
	return Error{
		Kind:    ErrInvalidAWSCredentials,
		message: "invalid AWS credentials",
	}
}

func ErrorMinInstancesGreaterThanMax(min int64, max int64) error {
	return Error{
		Kind:    ErrMinInstancesGreaterThanMax,
		message: fmt.Sprintf("%s cannot be greater than %s (%d > %d)", MinInstancesKey, MaxInstancesKey, min, max),
	}
}

func ErrorInstanceTypeNotSupportedInRegion(instanceType string, region string) error {
	return Error{
		Kind:    ErrInstanceTypeNotSupportedInRegion,
		message: fmt.Sprintf("%s is not supported in region %s", instanceType, region),
	}
}

func ErrorIncompatibleSpotInstanceTypeMemory(target aws.InstanceMetadata, suggested aws.InstanceMetadata) error {
	return Error{
		Kind:    ErrIncompatibleSpotInstanceTypeMemory,
		message: fmt.Sprintf("memory of %s can not be less than %s (%s < %s)", suggested.Type, target.Type, suggested.Memory.String(), target.Memory.String()),
	}
}

func ErrorIncompatibleSpotInstanceTypeCPU(target aws.InstanceMetadata, suggested aws.InstanceMetadata) error {
	return Error{
		Kind:    ErrIncompatibleSpotInstanceTypeCPU,
		message: fmt.Sprintf("CPU of %s can not be less than %s (%s < %s)", suggested.Type, target.Type, suggested.CPU.String(), target.CPU.String()),
	}
}

func ErrorIncompatibleSpotInstanceTypeGPU(target aws.InstanceMetadata, suggested aws.InstanceMetadata) error {
	return Error{
		Kind:    ErrIncompatibleSpotInstanceTypeGPU,
		message: fmt.Sprintf("GPU of %s can not be less than %s (%d GPU < %d GPU)", suggested.Type, target.Type, suggested.GPU, target.GPU),
	}
}

func ErrorGPUInstanceTypeNotSupported(instanceType string) error {
	return Error{
		Kind:    ErrGPUInstanceTypeNotSupported,
		message: fmt.Sprintf("GPU instance type %s is not supported", instanceType),
	}
}

func ErrorAtLeastOneInstanceDistribution(instanceType string, suggestions ...string) error {
	message := strings.Join(suggestions, ", ")
	return Error{
		Kind:    ErrAtLeastOneInstanceDistribution,
		message: fmt.Sprintf("at least one compatible instance type other than %s must be specified (suggestions: %s)", instanceType, message),
	}
}

func ErrorNoCompatibleSpotInstanceFound(instanceType string) error {
	return Error{
		Kind:    ErrNoCompatibleSpotInstanceFound,
		message: fmt.Sprintf("unable to find compatible spot instance types for %s", instanceType),
	}
}

func ErrorConfiguredWhenSpotIsNotEnabled(configKey string) error {
	return Error{
		Kind:    ErrConfiguredWhenSpotIsNotEnabled,
		message: fmt.Sprintf("%s is configured when spot is not enabled", configKey),
	}
}

func ErrorInvalidInstanceType(instanceType string) error {
	return Error{
		Kind:    ErrInvalidInstanceType,
		message: fmt.Sprintf("%s is an invalid instance type", instanceType),
	}
}
