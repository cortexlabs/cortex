package userconfig

import (
	"fmt"
	"strings"

	"github.com/cortexlabs/cortex/pkg/api/resource"
	s "github.com/cortexlabs/cortex/pkg/api/strings"
	"github.com/cortexlabs/cortex/pkg/utils/cast"
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
	ErrMissingRawFeatures
	ErrUndefinedResource
	ErrUndefinedResourceBuiltin
	ErrFeatureMustBeRaw
	ErrSpecifyAllOrNone
	ErrSpecifyOnlyOne
	ErrTemplateExtraArg
	ErrTemplateMissingArg
	ErrInvalidFeatureInputType
	ErrInvalidFeatureRuntimeType
	ErrInvalidValueDataType
	ErrUnsupportedFeatureType
	ErrUnsupportedDataType
	ErrArgNameCannotBeType
	ErrTypeListLength
	ErrGenericTypeMapLength
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
	"err_missing_raw_features",
	"err_undefined_resource",
	"err_undefined_resource_builtin",
	"err_feature_must_be_raw",
	"err_specify_all_or_none",
	"err_specify_only_one",
	"err_template_extra_arg",
	"err_template_missing_arg",
	"err_invalid_feature_input_type",
	"err_invalid_feature_runtime_type",
	"err_invalid_value_data_type",
	"err_unsupported_feature_type",
	"err_unsupported_data_type",
	"err_arg_name_cannot_be_type",
	"err_type_list_length",
	"err_generic_type_map_length",
}

var _ = [1]int{}[int(ErrGenericTypeMapLength)-(len(errorKinds)-1)] // Ensure list length matches

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

func ErrorDuplicateConfigName(name string, resourcesTypes ...resource.Type) error {
	resourceTypes := resource.Types(resourcesTypes)
	return ConfigError{
		Kind:    ErrDuplicateConfigName,
		message: fmt.Sprintf("name %s must be unique across %s", s.UserStr(name), s.StrsAnd(resourceTypes.PluralList())),
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

func ErrorMissingRawFeatures(missingFeatures []string) error {
	return ConfigError{
		Kind:    ErrMissingRawFeatures,
		message: fmt.Sprintf("all raw features must be ingested (missing: %s)", s.UserStrsAnd(missingFeatures)),
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

func ErrorFeatureMustBeRaw(featureName string) error {
	return ConfigError{
		Kind:    ErrFeatureMustBeRaw,
		message: fmt.Sprintf("%s is a transformed feature, but only raw features are allowed", s.UserStr(featureName)),
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

func ErrorInvalidFeatureInputType(provided interface{}) error {
	return ConfigError{
		Kind:    ErrInvalidFeatureInputType,
		message: fmt.Sprintf("invalid feature input type (got %s, expected %s, a combination of these types (separated by |), or a list of one of these types", s.DataTypeUserStr(provided), strings.Join(s.UserStrs(FeatureTypeStrings()), ", ")),
	}
}

func ErrorInvalidFeatureRuntimeType(provided interface{}) error {
	return ConfigError{
		Kind:    ErrInvalidFeatureRuntimeType,
		message: fmt.Sprintf("invalid feature type (got %s, expected %s)", s.DataTypeStr(provided), s.StrsOr(FeatureTypeStrings())),
	}
}

func ErrorInvalidValueDataType(provided interface{}) error {
	return ConfigError{
		Kind:    ErrInvalidValueDataType,
		message: fmt.Sprintf("invalid value data type (got %s, expected %s, a combination of these types (separated by |), a list of one of these types, or a map containing these types", s.DataTypeUserStr(provided), strings.Join(s.UserStrs(ValueTypeStrings()), ", ")),
	}
}

func ErrorUnsupportedFeatureType(provided interface{}, allowedTypes []string) error {
	allowedTypesInterface, _ := cast.InterfaceToInterfaceSlice(allowedTypes)
	return ConfigError{
		Kind:    ErrUnsupportedFeatureType,
		message: fmt.Sprintf("unsupported feature type (got %s, expected %s)", s.DataTypeStr(provided), s.DataTypeStrsOr(allowedTypesInterface)),
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

func (e ConfigError) Error() string {
	return e.message
}
