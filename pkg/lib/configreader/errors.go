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
	ErrCortexResourceOnlyAllowed
	ErrCortexResourceNotAllowed
)

var _errorKinds = []string{
	"err_unknown",
	"err_parse_config",
	"err_unsupported_key",
	"err_invalid_yaml",
	"err_too_long",
	"err_too_short",
	"err_alpha_numeric_dash_underscore",
	"err_alpha_numeric_dash_dot_underscore",
	"err_must_have_prefix",
	"err_invalid_interface",
	"err_invalid_float64",
	"err_invalid_float32",
	"err_invalid_int64",
	"err_invalid_int32",
	"err_invalid_int",
	"err_invalid_str",
	"err_must_be_less_than_or_equal_to",
	"err_must_be_less_than",
	"err_must_be_greater_than_or_equal_to",
	"err_must_be_greater_than",
	"err_is_not_multiple",
	"err_non_string_key_found",
	"err_invalid_primitive_type",
	"err_duplicated_value",
	"err_cannot_set_struct_field",
	"err_cannot_be_null",
	"err_cannot_be_empty",
	"err_must_be_defined",
	"err_map_must_be_defined",
	"err_must_be_empty",
	"err_cortex_resource_only_allowed",
	"err_cortex_resource_not_allowed",
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

type Error struct {
	Kind    ErrorKind
	message string
}

func (e Error) Error() string {
	return e.message
}

func ErrorParseConfig() error {
	return errors.WithStack(Error{
		Kind:    ErrParseConfig,
		message: fmt.Sprintf("failed to parse config file"),
	})
}

func ErrorUnsupportedKey(key interface{}) error {
	return errors.WithStack(Error{
		Kind:    ErrUnsupportedKey,
		message: fmt.Sprintf("key %s is not supported", s.UserStr(key)),
	})
}

func ErrorInvalidYAML(err error) error {
	str := strings.TrimPrefix(errors.Message(err), "yaml: ")
	return errors.WithStack(Error{
		Kind:    ErrInvalidYAML,
		message: fmt.Sprintf("invalid yaml: %s", str),
	})
}

func ErrorTooLong(provided string, maxLen int) error {
	return errors.WithStack(Error{
		Kind:    ErrTooLong,
		message: fmt.Sprintf("%s must be no more than %d characters", s.UserStr(provided), maxLen),
	})
}

func ErrorTooShort(provided string, minLen int) error {
	return errors.WithStack(Error{
		Kind:    ErrTooShort,
		message: fmt.Sprintf("%s must be no fewer than %d characters", s.UserStr(provided), minLen),
	})
}

func ErrorAlphaNumericDashUnderscore(provided string) error {
	return errors.WithStack(Error{
		Kind:    ErrAlphaNumericDashUnderscore,
		message: fmt.Sprintf("%s must contain only letters, numbers, underscores, and dashes", s.UserStr(provided)),
	})
}

func ErrorAlphaNumericDashDotUnderscore(provided string) error {
	return errors.WithStack(Error{
		Kind:    ErrAlphaNumericDashDotUnderscore,
		message: fmt.Sprintf("%s must contain only letters, numbers, underscores, dashes, and periods", s.UserStr(provided)),
	})
}

func ErrorMustHavePrefix(provided string, prefix string) error {
	return errors.WithStack(Error{
		Kind:    ErrMustHavePrefix,
		message: fmt.Sprintf("%s must start with %s", s.UserStr(provided), s.UserStr(prefix)),
	})
}

func ErrorInvalidInterface(provided interface{}, allowed interface{}, allowedVals ...interface{}) error {
	allAllowedVals := append(allowedVals, allowed)
	return errors.WithStack(Error{
		Kind:    ErrInvalidInterface,
		message: fmt.Sprintf("invalid value (got %s, must be %s)", s.UserStr(provided), s.UserStrsOr(allAllowedVals)),
	})
}

func ErrorInvalidFloat64(provided float64, allowed float64, allowedVals ...float64) error {
	allAllowedVals := append(allowedVals, allowed)
	return errors.WithStack(Error{
		Kind:    ErrInvalidFloat64,
		message: fmt.Sprintf("invalid value (got %s, must be %s)", s.UserStr(provided), s.UserStrsOr(allAllowedVals)),
	})
}

func ErrorInvalidFloat32(provided float32, allowed float32, allowedVals ...float32) error {
	allAllowedVals := append(allowedVals, allowed)
	return errors.WithStack(Error{
		Kind:    ErrInvalidFloat32,
		message: fmt.Sprintf("invalid value (got %s, must be %s)", s.UserStr(provided), s.UserStrsOr(allAllowedVals)),
	})
}

func ErrorInvalidInt64(provided int64, allowed int64, allowedVals ...int64) error {
	allAllowedVals := append(allowedVals, allowed)
	return errors.WithStack(Error{
		Kind:    ErrInvalidInt64,
		message: fmt.Sprintf("invalid value (got %s, must be %s)", s.UserStr(provided), s.UserStrsOr(allAllowedVals)),
	})
}

func ErrorInvalidInt32(provided int32, allowed int32, allowedVals ...int32) error {
	allAllowedVals := append(allowedVals, allowed)
	return errors.WithStack(Error{
		Kind:    ErrInvalidInt32,
		message: fmt.Sprintf("invalid value (got %s, must be %s)", s.UserStr(provided), s.UserStrsOr(allAllowedVals)),
	})
}

func ErrorInvalidInt(provided int, allowed int, allowedVals ...int) error {
	allAllowedVals := append(allowedVals, allowed)
	return errors.WithStack(Error{
		Kind:    ErrInvalidInt,
		message: fmt.Sprintf("invalid value (got %s, must be %s)", s.UserStr(provided), s.UserStrsOr(allAllowedVals)),
	})
}

func ErrorInvalidStr(provided string, allowed string, allowedVals ...string) error {
	allAllowedVals := append(allowedVals, allowed)
	return errors.WithStack(Error{
		Kind:    ErrInvalidStr,
		message: fmt.Sprintf("invalid value (got %s, must be %s)", s.UserStr(provided), s.UserStrsOr(allAllowedVals)),
	})
}

func ErrorMustBeLessThanOrEqualTo(provided interface{}, boundary interface{}) error {
	return errors.WithStack(Error{
		Kind:    ErrMustBeLessThanOrEqualTo,
		message: fmt.Sprintf("%s must be less than or equal to %s", s.UserStr(provided), s.UserStr(boundary)),
	})
}

func ErrorMustBeLessThan(provided interface{}, boundary interface{}) error {
	return errors.WithStack(Error{
		Kind:    ErrMustBeLessThan,
		message: fmt.Sprintf("%s must be less than %s", s.UserStr(provided), s.UserStr(boundary)),
	})
}

func ErrorMustBeGreaterThanOrEqualTo(provided interface{}, boundary interface{}) error {
	return errors.WithStack(Error{
		Kind:    ErrMustBeGreaterThanOrEqualTo,
		message: fmt.Sprintf("%s must be greater than or equal to %s", s.UserStr(provided), s.UserStr(boundary)),
	})
}

func ErrorMustBeGreaterThan(provided interface{}, boundary interface{}) error {
	return errors.WithStack(Error{
		Kind:    ErrMustBeGreaterThan,
		message: fmt.Sprintf("%s must be greater than %s", s.UserStr(provided), s.UserStr(boundary)),
	})
}

func ErrorIsNotMultiple(provided interface{}, multiple interface{}) error {
	return errors.WithStack(Error{
		Kind:    ErrIsNotMultiple,
		message: fmt.Sprintf("%s is not a multiple of %s", s.UserStr(provided), s.UserStr(multiple)),
	})
}

func ErrorNonStringKeyFound(key interface{}) error {
	return errors.WithStack(Error{
		Kind:    ErrNonStringKeyFound,
		message: fmt.Sprintf("non string key found: %s", s.ObjFlat(key)),
	})
}

func ErrorInvalidPrimitiveType(provided interface{}, allowedType PrimitiveType, allowedTypes ...PrimitiveType) error {
	allAllowedTypes := append(allowedTypes, allowedType)
	return errors.WithStack(Error{
		Kind:    ErrInvalidPrimitiveType,
		message: fmt.Sprintf("%s: invalid type (expected %s)", s.UserStr(provided), s.StrsOr(PrimitiveTypes(allAllowedTypes).StringList())),
	})
}

func ErrorDuplicatedValue(val interface{}) error {
	return errors.WithStack(Error{
		Kind:    ErrDuplicatedValue,
		message: fmt.Sprintf("%s is duplicated", s.UserStr(val)),
	})
}

func ErrorCannotSetStructField() error {
	return errors.WithStack(Error{
		Kind:    ErrCannotSetStructField,
		message: "unable to set struct field",
	})
}

func ErrorCannotBeNull() error {
	return errors.WithStack(Error{
		Kind:    ErrCannotBeNull,
		message: "cannot be null",
	})
}

func ErrorCannotBeEmpty() error {
	return errors.WithStack(Error{
		Kind:    ErrCannotBeEmpty,
		message: "cannot be empty",
	})
}

func ErrorMustBeDefined() error {
	return errors.WithStack(Error{
		Kind:    ErrMustBeDefined,
		message: "must be defined",
	})
}

func ErrorMapMustBeDefined(keys ...string) error {
	message := "must be defined"
	if len(keys) > 0 {
		message = fmt.Sprintf("must be defined, and contain the following keys: %s", s.UserStrsAnd(keys))
	}
	return errors.WithStack(Error{
		Kind:    ErrMapMustBeDefined,
		message: message,
	})
}

func ErrorMustBeEmpty() error {
	return errors.WithStack(Error{
		Kind:    ErrMustBeEmpty,
		message: "must be empty",
	})
}

func ErrorCortexResourceOnlyAllowed(invalidStr string) error {
	return errors.WithStack(Error{
		Kind:    ErrCortexResourceOnlyAllowed,
		message: fmt.Sprintf("%s: only cortex resource references (which start with @) are allowed in this context", invalidStr),
	})
}

func ErrorCortexResourceNotAllowed(resourceName string) error {
	return errors.WithStack(Error{
		Kind:    ErrCortexResourceNotAllowed,
		message: fmt.Sprintf("@%s: cortex resource references (which start with @) are not allowed in this context", resourceName),
	})
}
