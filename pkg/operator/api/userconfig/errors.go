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

	"github.com/cortexlabs/cortex/pkg/lib/configreader"
	"github.com/cortexlabs/cortex/pkg/lib/sets/strset"
	s "github.com/cortexlabs/cortex/pkg/lib/strings"
	"github.com/cortexlabs/cortex/pkg/operator/api/resource"
)

type ErrorKind int

const (
	ErrUnknown ErrorKind = iota
	ErrDuplicateResourceName
	ErrDuplicateResourceValue
	ErrDuplicateConfig
	ErrMalformedConfig
	ErrParseConfig
	ErrReadConfig
	ErrMissingAppDefinition
	ErrUndefinedConfig
	ErrRawColumnNotInEnv
	ErrUndefinedResource
	ErrResourceWrongType
	ErrSpecifyAllOrNone
	ErrSpecifyOnlyOne
	ErrOneOfPrerequisitesNotDefined
	ErrTemplateExtraArg
	ErrTemplateMissingArg
	ErrInvalidCompoundType
	ErrDuplicateTypeInTypeString
	ErrCannotMixValueAndColumnTypes
	ErrColumnTypeLiteral
	ErrColumnTypeNotAllowed
	ErrCompoundTypeInOutputType
	ErrUserKeysCannotStartWithUnderscore
	ErrMixedInputArgOptionsAndUserKeys
	ErrOptionOnNonIterable
	ErrMinCountGreaterThanMaxCount
	ErrTooManyElements
	ErrTooFewElements
	ErrInvalidInputType
	ErrInvalidOutputType
	ErrUnsupportedLiteralType
	ErrUnsupportedLiteralMapKey
	ErrUnsupportedOutputType
	ErrMustBeDefined
	ErrCannotBeNull
	ErrUnsupportedConfigKey
	ErrTypeListLength
	ErrTypeMapZeroLength
	ErrGenericTypeMapLength
	ErrK8sQuantityMustBeInt
	ErrPredictionKeyOnModelWithEstimator
	ErrSpecifyOnlyOneMissing
	ErrEnvSchemaMismatch
	ErrExtraResourcesWithExternalAPIs
	ErrImplDoesNotExist
)

var errorKinds = []string{
	"err_unknown",
	"err_duplicate_resource_name",
	"err_duplicate_resource_value",
	"err_duplicate_config",
	"err_malformed_config",
	"err_parse_config",
	"err_read_config",
	"err_missing_app_definition",
	"err_undefined_config",
	"err_raw_column_not_in_env",
	"err_undefined_resource",
	"err_resource_wrong_type",
	"err_specify_all_or_none",
	"err_specify_only_one",
	"err_one_of_prerequisites_not_defined",
	"err_template_extra_arg",
	"err_template_missing_arg",
	"err_invalid_compound_type",
	"err_duplicate_type_in_type_string",
	"err_cannot_mix_value_and_column_types",
	"err_column_type_literal",
	"err_column_type_not_allowed",
	"err_compound_type_in_output_type",
	"err_user_keys_cannot_start_with_underscore",
	"err_mixed_input_arg_options_and_user_keys",
	"err_option_on_non_iterable",
	"err_min_count_greater_than_max_count",
	"err_too_many_elements",
	"err_too_few_elements",
	"err_invalid_input_type",
	"err_invalid_output_type",
	"err_unsupported_literal_type",
	"err_unsupported_literal_map_key",
	"err_unsupported_output_type",
	"err_must_be_defined",
	"err_cannot_be_null",
	"error_unsupported_config_key",
	"err_type_list_length",
	"err_type_map_zero_length",
	"err_generic_type_map_length",
	"err_k8s_quantity_must_be_int",
	"err_prediction_key_on_model_with_estimator",
	"err_specify_only_one_missing",
	"err_env_schema_mismatch",
	"err_extra_resources_with_external_a_p_is",
	"err_impl_does_not_exist",
}

var _ = [1]int{}[int(ErrImplDoesNotExist)-(len(errorKinds)-1)] // Ensure list length matches

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

func ErrorDuplicateResourceName(resources ...Resource) error {
	filePaths := strset.New()
	embededFilePaths := strset.New()
	templates := strset.New()
	resourceTypes := strset.New()

	for _, res := range resources {
		resourceTypes.Add(res.GetResourceType().Plural())
		if emb := res.GetEmbed(); emb != nil {
			embededFilePaths.Add(res.GetFilePath())
			templates.Add(emb.Template)
		} else {
			filePaths.Add(res.GetFilePath())
		}
	}

	var pathStrs []string

	if len(filePaths) > 0 {
		pathStrs = append(pathStrs, "defined in "+s.StrsAnd(filePaths.Slice()))
	}

	if len(embededFilePaths) > 0 {
		embStr := "embedded in " + s.StrsAnd(embededFilePaths.Slice())
		if len(templates) > 1 {
			embStr += " via templates "
		} else {
			embStr += " via template "
		}
		embStr += s.UserStrsAnd(templates.Slice())
		pathStrs = append(pathStrs, embStr)
	}

	pathStr := strings.Join(pathStrs, ", ")

	return Error{
		Kind:    ErrDuplicateResourceName,
		message: fmt.Sprintf("name %s must be unique across %s (%s)", s.UserStr(resources[0].GetName()), s.StrsAnd(resourceTypes.Slice()), pathStr),
	}
}

func ErrorDuplicateResourceValue(value string, keys ...string) error {
	return Error{
		Kind:    ErrDuplicateResourceValue,
		message: fmt.Sprintf("%s is defined in %s, but may only be specified in one", s.UserStr(value), s.StrsAnd(keys)),
	}
}

func ErrorDuplicateConfig(resourceType resource.Type) error {
	return Error{
		Kind:    ErrDuplicateConfig,
		message: fmt.Sprintf("%s resource may only be defined once", resourceType.String()),
	}
}

func ErrorMalformedConfig() error {
	return Error{
		Kind:    ErrMalformedConfig,
		message: fmt.Sprintf("cortex YAML configuration files must contain a list of maps"),
	}
}

func ErrorParseConfig() error {
	parseErr := Error{
		Kind:    ErrParseConfig,
		message: fmt.Sprintf("failed to parse config file"),
	}

	return parseErr
}

func ErrorReadConfig() error {
	readErr := Error{
		Kind:    ErrReadConfig,
		message: fmt.Sprintf("failed to read config file"),
	}

	return readErr
}

func ErrorMissingAppDefinition() error {
	return Error{
		Kind:    ErrMissingAppDefinition,
		message: fmt.Sprintf("app.yaml must define an app resource"),
	}
}

func ErrorUndefinedConfig(resourceType resource.Type) error {
	return Error{
		Kind:    ErrUndefinedConfig,
		message: fmt.Sprintf("%s resource is not defined", resourceType.String()),
	}
}

func ErrorRawColumnNotInEnv(envName string) error {
	return Error{
		Kind:    ErrRawColumnNotInEnv,
		message: fmt.Sprintf("not defined in the schema for the %s %s", s.UserStr(envName), resource.EnvironmentType.String()),
	}
}

func ErrorUndefinedResource(resourceName string, resourceTypes ...resource.Type) error {
	message := fmt.Sprintf("%s is not defined", s.UserStr(resourceName))

	if len(resourceTypes) == 1 {
		message = fmt.Sprintf("%s %s is not defined", resourceTypes[0].String(), s.UserStr(resourceName))
	} else if len(resourceTypes) > 1 {
		message = fmt.Sprintf("%s is not defined as a %s", s.UserStr(resourceName), s.StrsOr(resource.Types(resourceTypes).StringList()))
	}

	if strings.HasPrefix(resourceName, "cortex.") {
		if len(resourceTypes) == 0 {
			message = fmt.Sprintf("%s is not defined in the Cortex namespace", s.UserStr(resourceName))
		} else {
			message = fmt.Sprintf("%s is not defined as a built-in %s in the Cortex namespace", s.UserStr(resourceName), s.StrsOr(resource.Types(resourceTypes).StringList()))
		}
	}

	return Error{
		Kind:    ErrUndefinedResource,
		message: message,
	}
}

func ErrorResourceWrongType(resources []Resource, validResourceTypes ...resource.Type) error {
	name := resources[0].GetName()
	resourceTypeStrs := make([]string, len(resources))
	for i, res := range resources {
		resourceTypeStrs[i] = res.GetResourceType().String()
	}

	return Error{
		Kind:    ErrResourceWrongType,
		message: fmt.Sprintf("%s is a %s, but only %s are allowed in this context", s.UserStr(name), s.StrsAnd(resourceTypeStrs), s.StrsOr(resource.Types(validResourceTypes).PluralList())),
	}
}

func ErrorSpecifyAllOrNone(vals ...string) error {
	message := fmt.Sprintf("please specify all or none of %s", s.UserStrsAnd(vals))
	if len(vals) == 2 {
		message = fmt.Sprintf("please specify both %s and %s or neither of them", s.UserStr(vals[0]), s.UserStr(vals[1]))
	}

	return Error{
		Kind:    ErrSpecifyAllOrNone,
		message: message,
	}
}

func ErrorSpecifyOnlyOne(vals ...string) error {
	message := fmt.Sprintf("please specify exactly one of %s", s.UserStrsOr(vals))
	if len(vals) == 2 {
		message = fmt.Sprintf("please specify either %s or %s, but not both", s.UserStr(vals[0]), s.UserStr(vals[1]))
	}

	return Error{
		Kind:    ErrSpecifyOnlyOne,
		message: message,
	}
}

func ErrorOneOfPrerequisitesNotDefined(argName string, prerequisites ...string) error {
	message := fmt.Sprintf("%s specified without specifying %s", s.UserStr(argName), s.UserStrsOr(prerequisites))

	return Error{
		Kind:    ErrOneOfPrerequisitesNotDefined,
		message: message,
	}
}

func ErrorTemplateExtraArg(template *Template, argName string) error {
	return Error{
		Kind:    ErrTemplateExtraArg,
		message: fmt.Sprintf("%s %s does not support an arg named %s", resource.TemplateType.String(), s.UserStr(template.Name), s.UserStr(argName)),
	}
}

func ErrorTemplateMissingArg(template *Template, argName string) error {
	return Error{
		Kind:    ErrTemplateMissingArg,
		message: fmt.Sprintf("%s %s requires an arg named %s", resource.TemplateType.String(), s.UserStr(template.Name), s.UserStr(argName)),
	}
}

func ErrorInvalidCompoundType(provided interface{}) error {
	return Error{
		Kind:    ErrInvalidCompoundType,
		message: fmt.Sprintf("invalid type (got %s, expected %s, or a combination of these types (separated by |)", DataTypeUserStr(provided), strings.Join(s.UserStrs(append(ValueTypeStrings(), ValidColumnTypeStrings()...)), ", ")),
	}
}

func ErrorDuplicateTypeInTypeString(duplicated string, provided string) error {
	return Error{
		Kind:    ErrDuplicateTypeInTypeString,
		message: fmt.Sprintf("invalid type (%s is duplicated in %s)", DataTypeUserStr(duplicated), DataTypeUserStr(provided)),
	}
}

func ErrorCannotMixValueAndColumnTypes(provided interface{}) error {
	return Error{
		Kind:    ErrCannotMixValueAndColumnTypes,
		message: fmt.Sprintf("invalid type (%s contains both column and value types)", DataTypeUserStr(provided)),
	}
}

func ErrorColumnTypeLiteral(provided interface{}) error {
	colName := "column_name"
	if providedStr, ok := provided.(string); ok {
		colName = providedStr
	}
	return Error{
		Kind:    ErrColumnTypeLiteral,
		message: fmt.Sprintf("%s: literal values cannot be provided for column input types (use a reference to a column, e.g. \"@%s\")", s.UserStrStripped(provided), colName),
	}
}

func ErrorColumnTypeNotAllowed(provided interface{}) error {
	return Error{
		Kind:    ErrColumnTypeNotAllowed,
		message: fmt.Sprintf("%s: column types cannot be used in this context, only value types are allowed (e.g. INT)", DataTypeUserStr(provided)),
	}
}

func ErrorCompoundTypeInOutputType(provided interface{}) error {
	return Error{
		Kind:    ErrCompoundTypeInOutputType,
		message: fmt.Sprintf("%s: compound types (i.e. multiple types separated by \"|\") cannot be used in output type schemas", DataTypeUserStr(provided)),
	}
}

func ErrorUserKeysCannotStartWithUnderscore(key string) error {
	return Error{
		Kind:    ErrUserKeysCannotStartWithUnderscore,
		message: fmt.Sprintf("%s: keys cannot start with underscores", key),
	}
}

func ErrorMixedInputArgOptionsAndUserKeys() error {
	return Error{
		Kind:    ErrMixedInputArgOptionsAndUserKeys,
		message: "input arguments cannot contain both Cortex argument options (which start with underscores) and user-provided keys (which don't start with underscores)",
	}
}

func ErrorOptionOnNonIterable(key string) error {
	return Error{
		Kind:    ErrOptionOnNonIterable,
		message: fmt.Sprintf("the %s option can only be used on list or maps", key),
	}
}

func ErrorMinCountGreaterThanMaxCount() error {
	return Error{
		Kind:    ErrMinCountGreaterThanMaxCount,
		message: fmt.Sprintf("the value provided for %s cannot be greater than the value provided for %s", MinCountOptKey, MaxCountOptKey),
	}
}

func ErrorTooManyElements(t configreader.PrimitiveType, maxCount int64) error {
	return Error{
		Kind:    ErrTooManyElements,
		message: fmt.Sprintf("the provided %s contains more than the maximum allowed number of elements (%s), which is specified via %s", string(t), s.Int64(maxCount), MaxCountOptKey),
	}
}

func ErrorTooFewElements(t configreader.PrimitiveType, minCount int64) error {
	return Error{
		Kind:    ErrTooFewElements,
		message: fmt.Sprintf("the provided %s contains fewer than the minimum allowed number of elements (%s), which is specified via %s", string(t), s.Int64(minCount), MinCountOptKey),
	}
}

func ErrorInvalidInputType(provided interface{}) error {
	return Error{
		Kind:    ErrInvalidInputType,
		message: fmt.Sprintf("invalid type (got %s, expected %s, a combination of these types (separated by |), or a list or map containing these types", DataTypeUserStr(provided), strings.Join(s.UserStrs(append(ValueTypeStrings(), ValidColumnTypeStrings()...)), ", ")),
	}
}

func ErrorInvalidOutputType(provided interface{}) error {
	return Error{
		Kind:    ErrInvalidOutputType,
		message: fmt.Sprintf("invalid type (got %s, expected %s, or a list or map containing these types", DataTypeUserStr(provided), strings.Join(s.UserStrs(ValueTypeStrings()), ", ")),
	}
}

func ErrorUnsupportedLiteralType(provided interface{}, allowedType interface{}) error {
	message := fmt.Sprintf("input value's type is not supported by the schema (got %s, expected input with type %s)", DataTypeStr(provided), DataTypeStr(allowedType))
	if str, ok := provided.(string); ok {
		message += fmt.Sprintf(" (note: if you are trying to reference a Cortex resource named %s, use \"@%s\")", str, str)
	}
	return Error{
		Kind:    ErrUnsupportedLiteralType,
		message: message,
	}
}

func ErrorUnsupportedLiteralMapKey(key interface{}, allowedType interface{}) error {
	return Error{
		Kind:    ErrUnsupportedLiteralMapKey,
		message: fmt.Sprintf("%s: map key is not supported by the schema (%s)", s.UserStrStripped(key), DataTypeStr(allowedType)),
	}
}

func ErrorUnsupportedOutputType(provided interface{}, allowedType interface{}) error {
	return Error{
		Kind:    ErrUnsupportedOutputType,
		message: fmt.Sprintf("unsupported type (got %s, expected %s)", DataTypeStr(provided), DataTypeStr(allowedType)),
	}
}

func ErrorMustBeDefined(allowedType interface{}) error {
	return Error{
		Kind:    ErrMustBeDefined,
		message: fmt.Sprintf("must be defined (and it's value must fit the schema %s)", DataTypeStr(allowedType)),
	}
}

func ErrorCannotBeNull() error {
	return Error{
		Kind:    ErrCannotBeNull,
		message: "cannot be null",
	}
}

func ErrorUnsupportedConfigKey() error {
	return Error{
		Kind:    ErrUnsupportedConfigKey,
		message: "is not supported for this resource",
	}
}

func ErrorTypeListLength(provided interface{}) error {
	return Error{
		Kind:    ErrTypeListLength,
		message: fmt.Sprintf("type lists must contain exactly one element (i.e. the desired data type) (got %s)", DataTypeStr(provided)),
	}
}

func ErrorTypeMapZeroLength(provided interface{}) error {
	return Error{
		Kind:    ErrTypeMapZeroLength,
		message: fmt.Sprintf("type maps must cannot have zero length (got %s)", DataTypeStr(provided)),
	}
}

func ErrorGenericTypeMapLength(provided interface{}) error {
	return Error{
		Kind:    ErrGenericTypeMapLength,
		message: fmt.Sprintf("maps with type keys (e.g. \"STRING\") must contain exactly one element (got %s)", DataTypeStr(provided)),
	}
}

func ErrorK8sQuantityMustBeInt(quantityStr string) error {
	return Error{
		Kind:    ErrK8sQuantityMustBeInt,
		message: fmt.Sprintf("resource compute quantity must be an integer-valued string, e.g. \"2\") (got %s)", DataTypeStr(quantityStr)),
	}
}

func ErrorPredictionKeyOnModelWithEstimator() error {
	return Error{
		Kind:    ErrPredictionKeyOnModelWithEstimator,
		message: fmt.Sprintf("models which use a pre-defined \"%s\" cannot define \"%s\" themselves (\"%s\" should be defined on the \"%s\", not the \"%s\")", EstimatorKey, PredictionKeyKey, PredictionKeyKey, resource.EstimatorType.String(), resource.ModelType.String()),
	}
}

func ErrorSpecifyOnlyOneMissing(vals ...string) error {
	message := fmt.Sprintf("please specify one of %s", s.UserStrsOr(vals))
	if len(vals) == 2 {
		message = fmt.Sprintf("please specify either %s or %s", s.UserStr(vals[0]), s.UserStr(vals[1]))
	}

	return Error{
		Kind:    ErrSpecifyOnlyOneMissing,
		message: message,
	}
}

func ErrorEnvSchemaMismatch(env1, env2 *Environment) error {
	return Error{
		Kind: ErrEnvSchemaMismatch,
		message: fmt.Sprintf("schemas diverge between environments (%s lists %s, and %s lists %s)",
			env1.Name,
			s.StrsAnd(env1.Data.GetIngestedColumnNames()),
			env2.Name,
			s.StrsAnd(env2.Data.GetIngestedColumnNames()),
		),
	}
}

func ErrorExtraResourcesWithExternalAPIs(res Resource) error {
	return Error{
		Kind:    ErrExtraResourcesWithExternalAPIs,
		message: fmt.Sprintf("only apis can be defined if environment is not defined (found %s)", Identify(res)),
	}
}

func ErrorImplDoesNotExist(path string) error {
	return Error{
		Kind:    ErrImplDoesNotExist,
		message: fmt.Sprintf("%s: implementation file does not exist", path),
	}
}
