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

package configreader

import (
	"fmt"
	"reflect"
	"strings"

	"github.com/cortexlabs/cortex/pkg/lib/errors"
	s "github.com/cortexlabs/cortex/pkg/lib/strings"
)

const (
	ErrParseConfig                   = "configreader.parse_config"
	ErrUnsupportedFieldValidation    = "configreader.unsupported_field_validation"
	ErrUnsupportedKey                = "configreader.unsupported_key"
	ErrInvalidYAML                   = "configreader.invalid_yaml"
	ErrTooLong                       = "configreader.too_long"
	ErrTooShort                      = "configreader.too_short"
	ErrLeadingWhitespace             = "configreader.leading_whitespace"
	ErrTrailingWhitespace            = "configreader.trailing_whitespace"
	ErrAlphaNumericDashUnderscore    = "configreader.alpha_numeric_dash_underscore"
	ErrAlphaNumericDashDotUnderscore = "configreader.alpha_numeric_dash_dot_underscore"
	ErrInvalidAWSTag                 = "configreader.invalid_aws_tag"
	ErrInvalidDockerImage            = "configreader.invalid_docker_image"
	ErrMustHavePrefix                = "configreader.must_have_prefix"
	ErrCantHavePrefix                = "configreader.cant_have_prefix"
	ErrInvalidInterface              = "configreader.invalid_interface"
	ErrInvalidFloat64                = "configreader.invalid_float64"
	ErrInvalidFloat32                = "configreader.invalid_float32"
	ErrInvalidInt64                  = "configreader.invalid_int64"
	ErrInvalidInt32                  = "configreader.invalid_int32"
	ErrInvalidInt                    = "configreader.invalid_int"
	ErrInvalidStr                    = "configreader.invalid_str"
	ErrDisallowedValue               = "configreader.disallowed_value"
	ErrMustBeLessThanOrEqualTo       = "configreader.must_be_less_than_or_equal_to"
	ErrMustBeLessThan                = "configreader.must_be_less_than"
	ErrMustBeGreaterThanOrEqualTo    = "configreader.must_be_greater_than_or_equal_to"
	ErrMustBeGreaterThan             = "configreader.must_be_greater_than"
	ErrIsNotMultiple                 = "configreader.is_not_multiple"
	ErrNonStringKeyFound             = "configreader.non_string_key_found"
	ErrInvalidPrimitiveType          = "configreader.invalid_primitive_type"
	ErrDuplicatedValue               = "configreader.duplicated_value"
	ErrTooFewElements                = "configreader.too_few_elements"
	ErrTooManyElements               = "configreader.too_many_elements"
	ErrWrongNumberOfElements         = "configreader.wrong_number_of_elements"
	ErrCannotSetStructField          = "configreader.cannot_set_struct_field"
	ErrCannotBeNull                  = "configreader.cannot_be_null"
	ErrCannotBeEmptyOrNull           = "configreader.cannot_be_empty_or_null"
	ErrCannotBeEmpty                 = "configreader.cannot_be_empty"
	ErrMustBeDefined                 = "configreader.must_be_defined"
	ErrMapMustBeDefined              = "configreader.map_must_be_defined"
	ErrMustBeEmpty                   = "configreader.must_be_empty"
	ErrEmailTooLong                  = "configreader.email_too_long"
	ErrEmailInvalid                  = "configreader.email_invalid"
	ErrCortexResourceOnlyAllowed     = "configreader.cortex_resource_only_allowed"
	ErrCortexResourceNotAllowed      = "configreader.cortex_resource_not_allowed"
	ErrImageVersionMismatch          = "configreader.image_version_mismatch"
	ErrFieldCantBeSpecified          = "configreader.field_cant_be_specified"
)

func ErrorParseConfig() error {
	return errors.WithStack(&errors.Error{
		Kind:    ErrParseConfig,
		Message: fmt.Sprintf("failed to parse config file"),
	})
}

func ErrorUnsupportedFieldValidation() error {
	return errors.WithStack(&errors.Error{
		Kind:    ErrUnsupportedFieldValidation,
		Message: fmt.Sprintf("undefined or unsupported field validation"),
	})
}

func ErrorUnsupportedKey(key interface{}) error {
	return errors.WithStack(&errors.Error{
		Kind:    ErrUnsupportedKey,
		Message: fmt.Sprintf("key %s is not supported", s.UserStr(key)),
	})
}

func ErrorInvalidYAML(err error) error {
	str := strings.TrimPrefix(errors.Message(err), "yaml: ")
	return errors.WithStack(&errors.Error{
		Kind:    ErrInvalidYAML,
		Message: fmt.Sprintf("invalid yaml: %s", str),
	})
}

func ErrorTooLong(provided string, maxLen int) error {
	return errors.WithStack(&errors.Error{
		Kind:    ErrTooLong,
		Message: fmt.Sprintf("%s must be no more than %d characters", s.UserStr(provided), maxLen),
	})
}

func ErrorTooShort(provided string, minLen int) error {
	return errors.WithStack(&errors.Error{
		Kind:    ErrTooShort,
		Message: fmt.Sprintf("%s must be at least %d characters", s.UserStr(provided), minLen),
	})
}

func ErrorLeadingWhitespace(provided string) error {
	return errors.WithStack(&errors.Error{
		Kind:    ErrLeadingWhitespace,
		Message: fmt.Sprintf("%s cannot start with whitespace", s.UserStr(provided)),
	})
}

func ErrorTrailingWhitespace(provided string) error {
	return errors.WithStack(&errors.Error{
		Kind:    ErrTrailingWhitespace,
		Message: fmt.Sprintf("%s cannot end with whitespace", s.UserStr(provided)),
	})
}

func ErrorAlphaNumericDashUnderscore(provided string) error {
	return errors.WithStack(&errors.Error{
		Kind:    ErrAlphaNumericDashUnderscore,
		Message: fmt.Sprintf("%s must contain only letters, numbers, underscores, and dashes", s.UserStr(provided)),
	})
}

func ErrorAlphaNumericDashDotUnderscore(provided string) error {
	return errors.WithStack(&errors.Error{
		Kind:    ErrAlphaNumericDashDotUnderscore,
		Message: fmt.Sprintf("%s must contain only letters, numbers, underscores, dashes, and periods", s.UserStr(provided)),
	})
}

func ErrorInvalidAWSTag(provided string) error {
	return errors.WithStack(&errors.Error{
		Kind:    ErrInvalidAWSTag,
		Message: fmt.Sprintf("%s must contain only letters, numbers, spaces, and the following characters: _ . : / + - @", s.UserStr(provided)),
	})
}

func ErrorInvalidDockerImage(provided string) error {
	return errors.WithStack(&errors.Error{
		Kind:    ErrInvalidDockerImage,
		Message: fmt.Sprintf("%s is not a valid docker image path", s.UserStr(provided)),
	})
}

func ErrorMustHavePrefix(provided string, prefix string) error {
	return errors.WithStack(&errors.Error{
		Kind:    ErrMustHavePrefix,
		Message: fmt.Sprintf("%s must start with %s", s.UserStr(provided), s.UserStr(prefix)),
	})
}

func ErrorCantHavePrefix(provided string, prefix string) error {
	return errors.WithStack(&errors.Error{
		Kind:    ErrCantHavePrefix,
		Message: fmt.Sprintf("%s cannot start with %s", s.UserStr(provided), s.UserStr(prefix)),
	})
}

func ErrorInvalidInterface(provided interface{}, allowed interface{}, allowedVals ...interface{}) error {
	allAllowedVals := append([]interface{}{allowed}, allowedVals...)
	return errors.WithStack(&errors.Error{
		Kind:    ErrInvalidInterface,
		Message: fmt.Sprintf("invalid value (got %s, must be %s)", s.UserStr(provided), s.UserStrsOr(allAllowedVals)),
	})
}

func ErrorInvalidFloat64(provided float64, allowed float64, allowedVals ...float64) error {
	allAllowedVals := append([]float64{allowed}, allowedVals...)
	return errors.WithStack(&errors.Error{
		Kind:    ErrInvalidFloat64,
		Message: fmt.Sprintf("invalid value (got %s, must be %s)", s.UserStr(provided), s.UserStrsOr(allAllowedVals)),
	})
}

func ErrorInvalidFloat32(provided float32, allowed float32, allowedVals ...float32) error {
	allAllowedVals := append([]float32{allowed}, allowedVals...)
	return errors.WithStack(&errors.Error{
		Kind:    ErrInvalidFloat32,
		Message: fmt.Sprintf("invalid value (got %s, must be %s)", s.UserStr(provided), s.UserStrsOr(allAllowedVals)),
	})
}

func ErrorInvalidInt64(provided int64, allowed int64, allowedVals ...int64) error {
	allAllowedVals := append([]int64{allowed}, allowedVals...)
	return errors.WithStack(&errors.Error{
		Kind:    ErrInvalidInt64,
		Message: fmt.Sprintf("invalid value (got %s, must be %s)", s.UserStr(provided), s.UserStrsOr(allAllowedVals)),
	})
}

func ErrorInvalidInt32(provided int32, allowed int32, allowedVals ...int32) error {
	allAllowedVals := append([]int32{allowed}, allowedVals...)
	return errors.WithStack(&errors.Error{
		Kind:    ErrInvalidInt32,
		Message: fmt.Sprintf("invalid value (got %s, must be %s)", s.UserStr(provided), s.UserStrsOr(allAllowedVals)),
	})
}

func ErrorInvalidInt(provided int, allowed int, allowedVals ...int) error {
	allAllowedVals := append([]int{allowed}, allowedVals...)
	return errors.WithStack(&errors.Error{
		Kind:    ErrInvalidInt,
		Message: fmt.Sprintf("invalid value (got %s, must be %s)", s.UserStr(provided), s.UserStrsOr(allAllowedVals)),
	})
}

func ErrorInvalidStr(provided string, allowed string, allowedVals ...string) error {
	allAllowedVals := append([]string{allowed}, allowedVals...)
	return errors.WithStack(&errors.Error{
		Kind:    ErrInvalidStr,
		Message: fmt.Sprintf("invalid value (got %s, must be %s)", s.UserStr(provided), s.UserStrsOr(allAllowedVals)),
	})
}

func ErrorDisallowedValue(provided interface{}) error {
	return errors.WithStack(&errors.Error{
		Kind:    ErrDisallowedValue,
		Message: fmt.Sprintf("%s is not allowed, please use a different value", s.UserStr(provided)),
	})
}

func ErrorMustBeLessThanOrEqualTo(provided interface{}, boundary interface{}) error {
	return errors.WithStack(&errors.Error{
		Kind:    ErrMustBeLessThanOrEqualTo,
		Message: fmt.Sprintf("must be less than or equal to %s (got %s)", s.UserStr(boundary), s.UserStr(provided)),
	})
}

func ErrorMustBeLessThan(provided interface{}, boundary interface{}) error {
	return errors.WithStack(&errors.Error{
		Kind:    ErrMustBeLessThan,
		Message: fmt.Sprintf("must be less than %s (got %s)", s.UserStr(boundary), s.UserStr(provided)),
	})
}

func ErrorMustBeGreaterThanOrEqualTo(provided interface{}, boundary interface{}) error {
	return errors.WithStack(&errors.Error{
		Kind:    ErrMustBeGreaterThanOrEqualTo,
		Message: fmt.Sprintf("must be greater than or equal to %s (got %s)", s.UserStr(boundary), s.UserStr(provided)),
	})
}

func ErrorMustBeGreaterThan(provided interface{}, boundary interface{}) error {
	return errors.WithStack(&errors.Error{
		Kind:    ErrMustBeGreaterThan,
		Message: fmt.Sprintf("must be greater than %s (got %s)", s.UserStr(boundary), s.UserStr(provided)),
	})
}

func ErrorIsNotMultiple(provided interface{}, multiple interface{}) error {
	return errors.WithStack(&errors.Error{
		Kind:    ErrIsNotMultiple,
		Message: fmt.Sprintf("%s is not a multiple of %s", s.UserStr(provided), s.UserStr(multiple)),
	})
}

func ErrorNonStringKeyFound(key interface{}) error {
	return errors.WithStack(&errors.Error{
		Kind:    ErrNonStringKeyFound,
		Message: fmt.Sprintf("non string key found: %s", s.ObjFlat(key)),
	})
}

func ErrorInvalidPrimitiveType(provided interface{}, allowedType PrimitiveType, allowedTypes ...PrimitiveType) error {
	allAllowedTypes := append([]PrimitiveType{allowedType}, allowedTypes...)
	return errors.WithStack(&errors.Error{
		Kind:    ErrInvalidPrimitiveType,
		Message: fmt.Sprintf("%s: invalid type (expected %s)", s.UserStr(provided), s.StrsOr(PrimitiveTypes(allAllowedTypes).StringList())),
	})
}

func ErrorDuplicatedValue(val interface{}) error {
	return errors.WithStack(&errors.Error{
		Kind:    ErrDuplicatedValue,
		Message: fmt.Sprintf("%s is duplicated", s.UserStr(val)),
	})
}

func ErrorTooFewElements(minLength int) error {
	return errors.WithStack(&errors.Error{
		Kind:    ErrTooFewElements,
		Message: fmt.Sprintf("must contain at least %d elements", minLength),
	})
}

func ErrorTooManyElements(maxLength int) error {
	return errors.WithStack(&errors.Error{
		Kind:    ErrTooManyElements,
		Message: fmt.Sprintf("must contain at most %d elements", maxLength),
	})
}

func ErrorWrongNumberOfElements(invalidLengths []int) error {
	invalidElementsStr := "elements"
	if len(invalidLengths) == 1 && invalidLengths[0] == 1 {
		invalidElementsStr = "element"
	}

	invalidLengthStrs := make([]string, len(invalidLengths))
	for i, length := range invalidLengths {
		invalidLengthStrs[i] = s.Int(length)
	}

	return errors.WithStack(&errors.Error{
		Kind:    ErrWrongNumberOfElements,
		Message: fmt.Sprintf("cannot contain %s %s", s.StrsOr(invalidLengthStrs), invalidElementsStr),
	})
}

func ErrorCannotSetStructField() error {
	return errors.WithStack(&errors.Error{
		Kind:    ErrCannotSetStructField,
		Message: "unable to set struct field",
	})
}

func ErrorCannotBeNull(isRequired bool) error {
	msg := "cannot be null"
	if !isRequired {
		msg = "cannot be null (specify a value, or remove the key to use the default value)"
	}
	return errors.WithStack(&errors.Error{
		Kind:    ErrCannotBeNull,
		Message: msg,
	})
}

func ErrorCannotBeEmptyOrNull(isRequired bool) error {
	msg := "cannot be empty or null"
	if !isRequired {
		msg = "cannot be empty or null (specify a value, or remove the key to use the default value)"
	}
	return errors.WithStack(&errors.Error{
		Kind:    ErrCannotBeEmptyOrNull,
		Message: msg,
	})
}

func ErrorCannotBeEmpty() error {
	return errors.WithStack(&errors.Error{
		Kind:    ErrCannotBeEmpty,
		Message: "cannot be empty",
	})
}

func ErrorMustBeDefined(validValues ...interface{}) error {
	msg := "must be defined"
	if len(validValues) > 0 && !reflect.ValueOf(validValues[0]).IsNil() { // reflect is necessary here
		msg = fmt.Sprintf("must be defined, and set to %s", s.UserStrsOr(validValues))
	}

	return errors.WithStack(&errors.Error{
		Kind:    ErrMustBeDefined,
		Message: msg,
	})
}

func ErrorMapMustBeDefined(keys ...string) error {
	message := "must be defined"
	if len(keys) > 0 {
		message = fmt.Sprintf("must be defined, and contain the following keys: %s", s.UserStrsAnd(keys))
	}
	return errors.WithStack(&errors.Error{
		Kind:    ErrMapMustBeDefined,
		Message: message,
	})
}

func ErrorMustBeEmpty() error {
	return errors.WithStack(&errors.Error{
		Kind:    ErrMustBeEmpty,
		Message: "must be empty",
	})
}

func ErrorEmailTooLong() error {
	return errors.WithStack(&errors.Error{
		Kind:    ErrEmailTooLong,
		Message: "email address exceeds maximum length",
	})
}

func ErrorEmailInvalid() error {
	return errors.WithStack(&errors.Error{
		Kind:    ErrEmailInvalid,
		Message: "invalid email address",
	})
}

func ErrorCortexResourceOnlyAllowed(invalidStr string) error {
	return errors.WithStack(&errors.Error{
		Kind:    ErrCortexResourceOnlyAllowed,
		Message: fmt.Sprintf("%s: only cortex resource references (which start with @) are allowed in this context", invalidStr),
	})
}

func ErrorCortexResourceNotAllowed(resourceName string) error {
	return errors.WithStack(&errors.Error{
		Kind:    ErrCortexResourceNotAllowed,
		Message: fmt.Sprintf("@%s: cortex resource references (which start with @) are not allowed in this context", resourceName),
	})
}

func ErrorImageVersionMismatch(image, tag, cortexVersion string) error {
	return errors.WithStack(&errors.Error{
		Kind:    ErrImageVersionMismatch,
		Message: fmt.Sprintf("the specified image (%s) has a tag (%s) which does not match your Cortex version (%s); please update the image tag, remove the image registry path from your configuration file (to use the default value), or update your CLI (pip install cortex==%s)", image, tag, cortexVersion, cortexVersion),
	})
}

func ErrorFieldCantBeSpecified(errMsg string) error {
	message := errMsg
	if message == "" {
		message = "cannot be specified"
	}
	return errors.WithStack(&errors.Error{
		Kind:    ErrFieldCantBeSpecified,
		Message: message,
	})
}
