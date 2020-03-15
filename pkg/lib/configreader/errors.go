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

package configreader

import (
	"fmt"
	"strings"

	"github.com/cortexlabs/cortex/pkg/lib/errors"
	s "github.com/cortexlabs/cortex/pkg/lib/strings"
)

type ErrorKind int

const (
	ErrUnknown ErrorKind = iota
	ErrParseConfig
	ErrUnsupportedFieldValidation
	ErrUnsupportedKey
	ErrInvalidYAML
	ErrTooLong
	ErrTooShort
	ErrAlphaNumericDashUnderscore
	ErrAlphaNumericDashDotUnderscore
	ErrMustHavePrefix
	ErrInvalidInterface
	ErrInvalidFloat64
	ErrInvalidFloat32
	ErrInvalidInt64
	ErrInvalidInt32
	ErrInvalidInt
	ErrInvalidStr
	ErrMustBeLessThanOrEqualTo
	ErrMustBeLessThan
	ErrMustBeGreaterThanOrEqualTo
	ErrMustBeGreaterThan
	ErrIsNotMultiple
	ErrNonStringKeyFound
	ErrInvalidPrimitiveType
	ErrDuplicatedValue
	ErrCannotSetStructField
	ErrCannotBeNull
	ErrCannotBeEmpty
	ErrMustBeDefined
	ErrMapMustBeDefined
	ErrMustBeEmpty
	ErrEmailTooLong
	ErrEmailInvalid
	ErrCortexResourceOnlyAllowed
	ErrCortexResourceNotAllowed
)

var _errorKinds = []string{
	"configreader.unknown",
	"configreader.parse_config",
	"configreader.unsupported_field_validation",
	"configreader.unsupported_key",
	"configreader.invalid_yaml",
	"configreader.too_long",
	"configreader.too_short",
	"configreader.alpha_numeric_dash_underscore",
	"configreader.alpha_numeric_dash_dot_underscore",
	"configreader.must_have_prefix",
	"configreader.invalid_interface",
	"configreader.invalid_float64",
	"configreader.invalid_float32",
	"configreader.invalid_int64",
	"configreader.invalid_int32",
	"configreader.invalid_int",
	"configreader.invalid_str",
	"configreader.must_be_less_than_or_equal_to",
	"configreader.must_be_less_than",
	"configreader.must_be_greater_than_or_equal_to",
	"configreader.must_be_greater_than",
	"configreader.is_not_multiple",
	"configreader.non_string_key_found",
	"configreader.invalid_primitive_type",
	"configreader.duplicated_value",
	"configreader.cannot_set_struct_field",
	"configreader.cannot_be_null",
	"configreader.cannot_be_empty",
	"configreader.must_be_defined",
	"configreader.map_must_be_defined",
	"configreader.must_be_empty",
	"configreader.email_too_long",
	"configreader.email_invalid",
	"configreader.cortex_resource_only_allowed",
	"configreader.cortex_resource_not_allowed",
}

var _ = [1]int{}[int(ErrCortexResourceNotAllowed)-(len(_errorKinds)-1)] // Ensure list length matches

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

func ErrorParseConfig() error {
	return errors.WithStack(&errors.CortexError{
		Kind:    ErrParseConfig,
		Message: fmt.Sprintf("failed to parse config file"),
	})
}

func ErrorUnsupportedFieldValidation() error {
	return errors.WithStack(&errors.CortexError{
		Kind:    ErrUnsupportedFieldValidation,
		Message: fmt.Sprintf("undefined or unsupported field validation"),
	})
}

func ErrorUnsupportedKey(key interface{}) error {
	return errors.WithStack(&errors.CortexError{
		Kind:    ErrUnsupportedKey,
		Message: fmt.Sprintf("key %s is not supported", s.UserStr(key)),
	})
}

func ErrorInvalidYAML(err error) error {
	str := strings.TrimPrefix(errors.Message(err), "yaml: ")
	return errors.WithStack(&errors.CortexError{
		Kind:    ErrInvalidYAML,
		Message: fmt.Sprintf("invalid yaml: %s", str),
	})
}

func ErrorTooLong(provided string, maxLen int) error {
	return errors.WithStack(&errors.CortexError{
		Kind:    ErrTooLong,
		Message: fmt.Sprintf("%s must be no more than %d characters", s.UserStr(provided), maxLen),
	})
}

func ErrorTooShort(provided string, minLen int) error {
	return errors.WithStack(&errors.CortexError{
		Kind:    ErrTooShort,
		Message: fmt.Sprintf("%s must be no fewer than %d characters", s.UserStr(provided), minLen),
	})
}

func ErrorAlphaNumericDashUnderscore(provided string) error {
	return errors.WithStack(&errors.CortexError{
		Kind:    ErrAlphaNumericDashUnderscore,
		Message: fmt.Sprintf("%s must contain only letters, numbers, underscores, and dashes", s.UserStr(provided)),
	})
}

func ErrorAlphaNumericDashDotUnderscore(provided string) error {
	return errors.WithStack(&errors.CortexError{
		Kind:    ErrAlphaNumericDashDotUnderscore,
		Message: fmt.Sprintf("%s must contain only letters, numbers, underscores, dashes, and periods", s.UserStr(provided)),
	})
}

func ErrorMustHavePrefix(provided string, prefix string) error {
	return errors.WithStack(&errors.CortexError{
		Kind:    ErrMustHavePrefix,
		Message: fmt.Sprintf("%s must start with %s", s.UserStr(provided), s.UserStr(prefix)),
	})
}

func ErrorInvalidInterface(provided interface{}, allowed interface{}, allowedVals ...interface{}) error {
	allAllowedVals := append(allowedVals, allowed)
	return errors.WithStack(&errors.CortexError{
		Kind:    ErrInvalidInterface,
		Message: fmt.Sprintf("invalid value (got %s, must be %s)", s.UserStr(provided), s.UserStrsOr(allAllowedVals)),
	})
}

func ErrorInvalidFloat64(provided float64, allowed float64, allowedVals ...float64) error {
	allAllowedVals := append(allowedVals, allowed)
	return errors.WithStack(&errors.CortexError{
		Kind:    ErrInvalidFloat64,
		Message: fmt.Sprintf("invalid value (got %s, must be %s)", s.UserStr(provided), s.UserStrsOr(allAllowedVals)),
	})
}

func ErrorInvalidFloat32(provided float32, allowed float32, allowedVals ...float32) error {
	allAllowedVals := append(allowedVals, allowed)
	return errors.WithStack(&errors.CortexError{
		Kind:    ErrInvalidFloat32,
		Message: fmt.Sprintf("invalid value (got %s, must be %s)", s.UserStr(provided), s.UserStrsOr(allAllowedVals)),
	})
}

func ErrorInvalidInt64(provided int64, allowed int64, allowedVals ...int64) error {
	allAllowedVals := append(allowedVals, allowed)
	return errors.WithStack(&errors.CortexError{
		Kind:    ErrInvalidInt64,
		Message: fmt.Sprintf("invalid value (got %s, must be %s)", s.UserStr(provided), s.UserStrsOr(allAllowedVals)),
	})
}

func ErrorInvalidInt32(provided int32, allowed int32, allowedVals ...int32) error {
	allAllowedVals := append(allowedVals, allowed)
	return errors.WithStack(&errors.CortexError{
		Kind:    ErrInvalidInt32,
		Message: fmt.Sprintf("invalid value (got %s, must be %s)", s.UserStr(provided), s.UserStrsOr(allAllowedVals)),
	})
}

func ErrorInvalidInt(provided int, allowed int, allowedVals ...int) error {
	allAllowedVals := append(allowedVals, allowed)
	return errors.WithStack(&errors.CortexError{
		Kind:    ErrInvalidInt,
		Message: fmt.Sprintf("invalid value (got %s, must be %s)", s.UserStr(provided), s.UserStrsOr(allAllowedVals)),
	})
}

func ErrorInvalidStr(provided string, allowed string, allowedVals ...string) error {
	allAllowedVals := append(allowedVals, allowed)
	return errors.WithStack(&errors.CortexError{
		Kind:    ErrInvalidStr,
		Message: fmt.Sprintf("invalid value (got %s, must be %s)", s.UserStr(provided), s.UserStrsOr(allAllowedVals)),
	})
}

func ErrorMustBeLessThanOrEqualTo(provided interface{}, boundary interface{}) error {
	return errors.WithStack(&errors.CortexError{
		Kind:    ErrMustBeLessThanOrEqualTo,
		Message: fmt.Sprintf("%s must be less than or equal to %s", s.UserStr(provided), s.UserStr(boundary)),
	})
}

func ErrorMustBeLessThan(provided interface{}, boundary interface{}) error {
	return errors.WithStack(&errors.CortexError{
		Kind:    ErrMustBeLessThan,
		Message: fmt.Sprintf("%s must be less than %s", s.UserStr(provided), s.UserStr(boundary)),
	})
}

func ErrorMustBeGreaterThanOrEqualTo(provided interface{}, boundary interface{}) error {
	return errors.WithStack(&errors.CortexError{
		Kind:    ErrMustBeGreaterThanOrEqualTo,
		Message: fmt.Sprintf("%s must be greater than or equal to %s", s.UserStr(provided), s.UserStr(boundary)),
	})
}

func ErrorMustBeGreaterThan(provided interface{}, boundary interface{}) error {
	return errors.WithStack(&errors.CortexError{
		Kind:    ErrMustBeGreaterThan,
		Message: fmt.Sprintf("%s must be greater than %s", s.UserStr(provided), s.UserStr(boundary)),
	})
}

func ErrorIsNotMultiple(provided interface{}, multiple interface{}) error {
	return errors.WithStack(&errors.CortexError{
		Kind:    ErrIsNotMultiple,
		Message: fmt.Sprintf("%s is not a multiple of %s", s.UserStr(provided), s.UserStr(multiple)),
	})
}

func ErrorNonStringKeyFound(key interface{}) error {
	return errors.WithStack(&errors.CortexError{
		Kind:    ErrNonStringKeyFound,
		Message: fmt.Sprintf("non string key found: %s", s.ObjFlat(key)),
	})
}

func ErrorInvalidPrimitiveType(provided interface{}, allowedType PrimitiveType, allowedTypes ...PrimitiveType) error {
	allAllowedTypes := append(allowedTypes, allowedType)
	return errors.WithStack(&errors.CortexError{
		Kind:    ErrInvalidPrimitiveType,
		Message: fmt.Sprintf("%s: invalid type (expected %s)", s.UserStr(provided), s.StrsOr(PrimitiveTypes(allAllowedTypes).StringList())),
	})
}

func ErrorDuplicatedValue(val interface{}) error {
	return errors.WithStack(&errors.CortexError{
		Kind:    ErrDuplicatedValue,
		Message: fmt.Sprintf("%s is duplicated", s.UserStr(val)),
	})
}

func ErrorCannotSetStructField() error {
	return errors.WithStack(&errors.CortexError{
		Kind:    ErrCannotSetStructField,
		Message: "unable to set struct field",
	})
}

func ErrorCannotBeNull() error {
	return errors.WithStack(&errors.CortexError{
		Kind:    ErrCannotBeNull,
		Message: "cannot be null",
	})
}

func ErrorCannotBeEmpty() error {
	return errors.WithStack(&errors.CortexError{
		Kind:    ErrCannotBeEmpty,
		Message: "cannot be empty",
	})
}

func ErrorMustBeDefined() error {
	return errors.WithStack(&errors.CortexError{
		Kind:    ErrMustBeDefined,
		Message: "must be defined",
	})
}

func ErrorMapMustBeDefined(keys ...string) error {
	message := "must be defined"
	if len(keys) > 0 {
		message = fmt.Sprintf("must be defined, and contain the following keys: %s", s.UserStrsAnd(keys))
	}
	return errors.WithStack(&errors.CortexError{
		Kind:    ErrMapMustBeDefined,
		Message: message,
	})
}

func ErrorMustBeEmpty() error {
	return errors.WithStack(&errors.CortexError{
		Kind:    ErrMustBeEmpty,
		Message: "must be empty",
	})
}

func ErrorEmailTooLong() error {
	return errors.WithStack(&errors.CortexError{
		Kind:    ErrEmailTooLong,
		Message: "email address exceeds maximum length",
	})
}

func ErrorEmailInvalid() error {
	return errors.WithStack(&errors.CortexError{
		Kind:    ErrEmailInvalid,
		Message: "invalid email address",
	})
}

func ErrorCortexResourceOnlyAllowed(invalidStr string) error {
	return errors.WithStack(&errors.CortexError{
		Kind:    ErrCortexResourceOnlyAllowed,
		Message: fmt.Sprintf("%s: only cortex resource references (which start with @) are allowed in this context", invalidStr),
	})
}

func ErrorCortexResourceNotAllowed(resourceName string) error {
	return errors.WithStack(&errors.CortexError{
		Kind:    ErrCortexResourceNotAllowed,
		Message: fmt.Sprintf("@%s: cortex resource references (which start with @) are not allowed in this context", resourceName),
	})
}
