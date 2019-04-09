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

package strings

import (
	"fmt"
)

var (
	ErrPending       = "pending"
	ErrNotFound      = "not found"
	ErrMissing       = "missing"
	ErrMustBeDefined = "must be defined"

	ErrCannotBeEmpty = "cannot be empty"
	ErrMustBeEmpty   = "must be empty"
	ErrCannotBeNull  = "cannot be null"

	ErrRead            = "unable to read"
	ErrUnzip           = "unable to unzip file"
	ErrCreateZip       = "unable to create zip file"
	ErrCantMakeRequest = "unable to make request"

	ErrMarshalJSON      = "invalid json cannot be serialized"
	ErrUnmarshalJSON    = "invalid json"
	ErrMarshalMsgpack   = "invalid messagepack cannot be serialized"
	ErrUnmarshalMsgpack = "invalid messagepack"

	ErrClassificationTargetType = "classification models can only predict integer target values (i.e. {0, 1, ..., num_classes-1})"
	ErrRegressionTargetType     = "regression models can only predict float or integer target values"
	ErrCliNotInAppDir           = "your current working directory is not in or under a cortex app directory (identified via a top-level app.yaml file)"
	ErrLoadBalancerInitializing = "load balancer is still initializing"
	ErrUnableToAuthAws          = "unable to authenticate with AWS"
	ErrCortexInstallationBroken = "cortex is out of date, or not installed properly on your cluster; run `./cortex-installer.sh uninstall operator && ./cortex-installer.sh install operator`"

	// internal only

	ErrUnexpected           = "an unexpected error occurred"
	ErrWorkflowAppMismatch  = "workflow apps do not match"
	ErrContextAppMismatch   = "context apps do not match"
	ErrMoreThanOneWorkflow  = "there is more than one workflow"
	ErrCannotSetStructField = "unable to set struct field"
)

func Index(index int) string {
	return fmt.Sprintf("index %d", index)
}

func MapMustBeDefined(keys ...string) string {
	if len(keys) == 0 {
		return fmt.Sprintf("must be defined")
	}
	return fmt.Sprintf("must be defined, and contain the following keys: %s", UserStrsAnd(keys))
}

func ErrDuplicatedValue(val interface{}) string {
	return fmt.Sprintf("%s is duplicated", UserStr(val))
}

func ErrInvalidPrimitiveType(provided interface{}, allowedTypes ...PrimitiveType) string {
	return fmt.Sprintf("%s: invalid type (expected %s)", UserStr(provided), StrsOr(PrimitiveTypes(allowedTypes).StringList()))
}

func ErrMustBeGreaterThan(provided interface{}, boundary interface{}) string {
	return fmt.Sprintf("%s must be greater than %s", UserStr(provided), UserStr(boundary))
}
func ErrMustBeGreaterThanOrEqualTo(provided interface{}, boundary interface{}) string {
	return fmt.Sprintf("%s must be greater than or equal to %s", UserStr(provided), UserStr(boundary))
}
func ErrMustBeLessThan(provided interface{}, boundary interface{}) string {
	return fmt.Sprintf("%s must be less than %s", UserStr(provided), UserStr(boundary))
}
func ErrMustBeLessThanOrEqualTo(provided interface{}, boundary interface{}) string {
	return fmt.Sprintf("%s must be less than or equal to %s", UserStr(provided), UserStr(boundary))
}

func ErrInvalidStr(provided string, allowed ...string) string {
	return fmt.Sprintf("invalid value (got %s, must be %s)", UserStr(provided), UserStrsOr(allowed))
}
func ErrInvalidInt(provided int, allowed ...int) string {
	return fmt.Sprintf("invalid value (got %s, must be %s)", UserStr(provided), UserStrsOr(allowed))
}
func ErrInvalidInt32(provided int32, allowed ...int32) string {
	return fmt.Sprintf("invalid value (got %s, must be %s)", UserStr(provided), UserStrsOr(allowed))
}
func ErrInvalidInt64(provided int64, allowed ...int64) string {
	return fmt.Sprintf("invalid value (got %s, must be %s)", UserStr(provided), UserStrsOr(allowed))
}
func ErrInvalidFloat32(provided float32, allowed ...float32) string {
	return fmt.Sprintf("invalid value (got %s, must be %s)", UserStr(provided), UserStrsOr(allowed))
}
func ErrInvalidFloat64(provided float64, allowed ...float64) string {
	return fmt.Sprintf("invalid value (got %s, must be %s)", UserStr(provided), UserStrsOr(allowed))
}
func ErrInvalidInterface(provided interface{}, allowed ...interface{}) string {
	return fmt.Sprintf("invalid value (got %s, must be %s)", UserStr(provided), UserStrsOr(allowed))
}

func ErrMustHavePrefix(provided string, prefix string) string {
	return fmt.Sprintf("%s must start with %s", UserStr(provided), UserStr(prefix))
}
func ErrAlphaNumericDashDotUnderscore(provided string) string {
	return fmt.Sprintf("%s must contain only letters, numbers, underscores, dashes, and periods", UserStr(provided))
}
func ErrAlphaNumericDashUnderscore(provided string) string {
	return fmt.Sprintf("%s must contain only letters, numbers, underscores, and dashes", UserStr(provided))
}
func ErrDNS1035(provided string) string {
	return fmt.Sprintf("%s must contain only lower case letters, numbers, and dashes, start with a letter, and cannot end with a dash", UserStr(provided))
}
func ErrInvalidURL(provided string) string {
	return fmt.Sprintf("%s is not a valid URL", UserStr(provided))
}
func ErrInvalidS3aPath(provided string) string {
	return fmt.Sprintf("%s is not a valid s3a path", UserStr(provided))
}

func ErrFileDoesNotExist(path string) string {
	return fmt.Sprintf("%s: file does not exist", path)
}

func ErrDirDoesNotExist(path string) string {
	return fmt.Sprintf("%s: directory does not exist", path)
}

func ErrFileAlreadyExists(path string) string {
	return fmt.Sprintf("%s: file already exists", path)
}

func ErrReadFile(path string) string {
	return fmt.Sprintf("%s: unable to read file", path)
}

func ErrReadDir(path string) string {
	return fmt.Sprintf("%s: unable to read directory", path)
}
