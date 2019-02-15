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

package userconfig

import (
	"fmt"
	"strings"

	"github.com/cortexlabs/cortex/pkg/api/resource"
	s "github.com/cortexlabs/cortex/pkg/api/strings"
	"github.com/cortexlabs/cortex/pkg/utils/cast"
	"github.com/cortexlabs/cortex/pkg/utils/util"
)

type ErrorKind int

const (
	ErrUnknown ErrorKind = iota
	ErrDuplicateConfigName
	ErrDuplicateResourceValue
	ErrDuplicateConfig
	ErrMalformedConfig
	ErrParseConfig
	ErrReadConfig
	ErrMissingAppDefinition
	ErrUndefinedConfig
	ErrMissingRawColumns
	ErrUndefinedResource
	ErrUndefinedResourceBuiltin
	ErrColumnMustBeRaw
	ErrSpecifyAllOrNone
	ErrSpecifyOnlyOne
	ErrTemplateExtraArg
	ErrTemplateMissingArg
	ErrInvalidColumnInputType
	ErrInvalidColumnRuntimeType
	ErrInvalidValueDataType
	ErrUnsupportedColumnType
	ErrUnsupportedDataType
	ErrArgNameCannotBeType
	ErrTypeListLength
	ErrGenericTypeMapLength
	ErrK8sQuantityMustBeInt
)

var errorKinds = []string{
	"err_unknown",
	"err_duplicate_config_name",
	"err_duplicate_resource_value",
	"err_duplicate_config",
	"err_malformed_config",
	"err_parse_config",
	"err_read_config",
	"err_missing_app_definition",
	"err_undefined_config",
	"err_missing_raw_columns",
	"err_undefined_resource",
	"err_undefined_resource_builtin",
	"err_column_must_be_raw",
	"err_specify_all_or_none",
	"err_specify_only_one",
	"err_template_extra_arg",
	"err_template_missing_arg",
	"err_invalid_column_input_type",
	"err_invalid_column_runtime_type",
	"err_invalid_value_data_type",
	"err_unsupported_column_type",
	"err_unsupported_data_type",
	"err_arg_name_cannot_be_type",
	"err_type_list_length",
	"err_generic_type_map_length",
	"err_k8s_quantity_must_be_int",
}

var _ = [1]int{}[int(ErrK8sQuantityMustBeInt)-(len(errorKinds)-1)] // Ensure list length matches

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

type ConfigError struct {
	Kind    ErrorKind
	message string
}

func (e ConfigError) Error() string {
	return e.message
}

func ErrorDuplicateResourceName(resources ...Resource) error {
	filePaths := make([]string, len(resources))
	resourceTypes := make(resource.Types, len(resources))
	for i, res := range resources {
		filePaths[i] = res.GetFilePath()
		resourceTypes[i] = res.GetResourceType()
	}

	return ConfigError{
		Kind:    ErrDuplicateConfigName,
		message: fmt.Sprintf("name %s must be unique across %s (defined in %s)", s.UserStr(resources[0].GetName()), s.StrsAnd(util.UniqueStrs(resourceTypes.PluralList())), s.StrsAnd(util.UniqueStrs(filePaths))),
	}
}

func ErrorDuplicateResourceValue(value string, keys ...string) error {
	return ConfigError{
		Kind:    ErrDuplicateResourceValue,
		message: fmt.Sprintf("%s is defined in %s, but may only be specified in one", s.UserStr(value), s.StrsAnd(keys)),
	}
}

func ErrorDuplicateConfig(resourceType resource.Type) error {
	return ConfigError{
		Kind:    ErrDuplicateConfig,
		message: fmt.Sprintf("%s resource may only be defined once", resourceType.String()),
	}
}

func ErrorMalformedConfig() error {
	return ConfigError{
		Kind:    ErrMalformedConfig,
		message: fmt.Sprintf("cortex YAML configuration files must contain a list of maps"),
	}
}

func ErrorParseConfig() error {
	parseErr := ConfigError{
		Kind:    ErrParseConfig,
		message: fmt.Sprintf("failed to parse config file"),
	}

	return parseErr
}

func ErrorReadConfig() error {
	readErr := ConfigError{
		Kind:    ErrReadConfig,
		message: fmt.Sprintf("failed to read config file"),
	}

	return readErr
}

func ErrorMissingAppDefinition() error {
	return ConfigError{
		Kind:    ErrMissingAppDefinition,
		message: fmt.Sprintf("app.yaml must define an app resource"),
	}
}

func ErrorUndefinedConfig(resourceType resource.Type) error {
	return ConfigError{
		Kind:    ErrUndefinedConfig,
		message: fmt.Sprintf("%s resource is not defined", resourceType.String()),
	}
}

func ErrorMissingRawColumns(missingColumns []string) error {
	return ConfigError{
		Kind:    ErrMissingRawColumns,
		message: fmt.Sprintf("all raw columns must be ingested (missing: %s)", s.UserStrsAnd(missingColumns)),
	}
}

func ErrorUndefinedResource(resourceName string, resourceTypes ...resource.Type) error {
	return ConfigError{
		Kind:    ErrUndefinedResource,
		message: fmt.Sprintf("%s %s is not defined", s.StrsOr(resource.Types(resourceTypes).StringList()), s.UserStr(resourceName)),
	}
}

func ErrorUndefinedResourceBuiltin(resourceName string, resourceTypes ...resource.Type) error {
	return ConfigError{
		Kind:    ErrUndefinedResourceBuiltin,
		message: fmt.Sprintf("%s %s is not defined in the Cortex namespace", s.StrsOr(resource.Types(resourceTypes).StringList()), s.UserStr(resourceName)),
	}
}

func ErrorColumnMustBeRaw(columnName string) error {
	return ConfigError{
		Kind:    ErrColumnMustBeRaw,
		message: fmt.Sprintf("%s is a transformed column, but only raw columns are allowed", s.UserStr(columnName)),
	}
}

func ErrorSpecifyAllOrNone(vals ...string) error {
	message := fmt.Sprintf("please specify all or none of %s", s.UserStrsAnd(vals))
	if len(vals) == 2 {
		message = fmt.Sprintf("please specify both %s and %s or neither of them", s.UserStr(vals[0]), s.UserStr(vals[1]))
	}

	return ConfigError{
		Kind:    ErrSpecifyAllOrNone,
		message: message,
	}
}

func ErrorSpecifyOnlyOne(vals ...string) error {
	message := fmt.Sprintf("please specify exactly one of %s", s.UserStrsOr(vals))
	if len(vals) == 2 {
		message = fmt.Sprintf("please specify either %s or %s, but not both", s.UserStr(vals[0]), s.UserStr(vals[1]))
	}

	return ConfigError{
		Kind:    ErrSpecifyOnlyOne,
		message: message,
	}
}

func ErrorTemplateExtraArg(template *Template, argName string) error {
	return ConfigError{
		Kind:    ErrTemplateExtraArg,
		message: fmt.Sprintf("%s %s does not support an arg named %s", resource.TemplateType.String(), s.UserStr(template.Name), s.UserStr(argName)),
	}
}

func ErrorTemplateMissingArg(template *Template, argName string) error {
	return ConfigError{
		Kind:    ErrTemplateMissingArg,
		message: fmt.Sprintf("%s %s requires an arg named %s", resource.TemplateType.String(), s.UserStr(template.Name), s.UserStr(argName)),
	}
}

func ErrorInvalidColumnInputType(provided interface{}) error {
	return ConfigError{
		Kind:    ErrInvalidColumnInputType,
		message: fmt.Sprintf("invalid column input type (got %s, expected %s, a combination of these types (separated by |), or a list of one of these types", s.DataTypeUserStr(provided), strings.Join(s.UserStrs(ColumnTypeStrings()), ", ")),
	}
}

func ErrorInvalidColumnRuntimeType(provided interface{}) error {
	return ConfigError{
		Kind:    ErrInvalidColumnRuntimeType,
		message: fmt.Sprintf("invalid column type (got %s, expected %s)", s.DataTypeStr(provided), s.StrsOr(ColumnTypeStrings())),
	}
}

func ErrorInvalidValueDataType(provided interface{}) error {
	return ConfigError{
		Kind:    ErrInvalidValueDataType,
		message: fmt.Sprintf("invalid value data type (got %s, expected %s, a combination of these types (separated by |), a list of one of these types, or a map containing these types", s.DataTypeUserStr(provided), strings.Join(s.UserStrs(ValueTypeStrings()), ", ")),
	}
}

func ErrorUnsupportedColumnType(provided interface{}, allowedTypes []string) error {
	allowedTypesInterface, _ := cast.InterfaceToInterfaceSlice(allowedTypes)
	return ConfigError{
		Kind:    ErrUnsupportedColumnType,
		message: fmt.Sprintf("unsupported column type (got %s, expected %s)", s.DataTypeStr(provided), s.DataTypeStrsOr(allowedTypesInterface)),
	}
}

func ErrorUnsupportedDataType(provided interface{}, allowedType interface{}) error {
	return ConfigError{
		Kind:    ErrUnsupportedDataType,
		message: fmt.Sprintf("unsupported data type (got %s, expected %s)", s.DataTypeStr(provided), s.DataTypeStr(allowedType)),
	}
}

func ErrorArgNameCannotBeType(provided string) error {
	return ConfigError{
		Kind:    ErrArgNameCannotBeType,
		message: fmt.Sprintf("data types cannot be used as arg names (got %s)", s.UserStr(provided)),
	}
}

func ErrorTypeListLength(provided interface{}) error {
	return ConfigError{
		Kind:    ErrTypeListLength,
		message: fmt.Sprintf("type lists must contain exactly one element (i.e. the desired data type) (got %s)", s.DataTypeStr(provided)),
	}
}

func ErrorGenericTypeMapLength(provided interface{}) error {
	return ConfigError{
		Kind:    ErrGenericTypeMapLength,
		message: fmt.Sprintf("generic type maps must contain exactly one key (i.e. the desired data type of all keys in the map) (got %s)", s.DataTypeStr(provided)),
	}
}

func ErrorK8sQuantityMustBeInt(quantityStr string) error {
	return ConfigError{
		Kind:    ErrK8sQuantityMustBeInt,
		message: fmt.Sprintf("resource compute quantity must be an integer-valued string, e.g. \"2\") (got %s)", s.DataTypeStr(quantityStr)),
	}
}
