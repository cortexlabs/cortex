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
	"strings"

	"github.com/cortexlabs/cortex/pkg/lib/cast"
	"github.com/cortexlabs/cortex/pkg/lib/errors"
	"github.com/cortexlabs/cortex/pkg/lib/exit"
	"github.com/cortexlabs/cortex/pkg/lib/files"
	"github.com/cortexlabs/cortex/pkg/lib/prompt"
	"github.com/cortexlabs/cortex/pkg/lib/regex"
	"github.com/cortexlabs/cortex/pkg/lib/slices"
	s "github.com/cortexlabs/cortex/pkg/lib/strings"
	"github.com/cortexlabs/cortex/pkg/lib/urls"
)

type StringValidation struct {
	Required                             bool
	Default                              string
	AllowEmpty                           bool // Allow `<field>: ""`
	TreatNullAsEmpty                     bool // `<field>: ` and `<field>: null` will be read as `<field>: ""`
	AllowedValues                        []string
	DisallowedValues                     []string
	CantBeSpecifiedErrStr                *string
	Prefix                               string
	InvalidPrefixes                      []string
	MaxLength                            int
	MinLength                            int
	DisallowLeadingWhitespace            bool
	DisallowTrailingWhitespace           bool
	AlphaNumericDashDotUnderscoreOrEmpty bool
	AlphaNumericDashDotUnderscore        bool
	AlphaNumericDashUnderscoreOrEmpty    bool
	AlphaNumericDashUnderscore           bool
	AWSTag                               bool
	DNS1035                              bool
	DNS1123                              bool
	CastInt                              bool
	CastNumeric                          bool
	CastScalar                           bool
	AllowCortexResources                 bool
	RequireCortexResources               bool
	DockerImageOrEmpty                   bool
	Validator                            func(string) (string, error)
}

func EnvVar(envVarName string) string {
	return fmt.Sprintf("environment variable \"%s\"", envVarName)
}

func String(inter interface{}, v *StringValidation) (string, error) {
	if inter == nil {
		if v.TreatNullAsEmpty {
			return ValidateStringProvided("", v)
		}
		return "", ErrorCannotBeNull(v.Required)
	}
	casted, castOk := inter.(string)
	if !castOk {
		if v.CastScalar {
			if !cast.IsScalarType(inter) {
				return "", ErrorInvalidPrimitiveType(inter, PrimTypeString, PrimTypeInt, PrimTypeFloat, PrimTypeBool)
			}
			casted = s.ObjFlatNoQuotes(inter)
		} else if v.CastNumeric {
			if !cast.IsNumericType(inter) {
				return "", ErrorInvalidPrimitiveType(inter, PrimTypeString, PrimTypeInt, PrimTypeFloat)
			}
			casted = s.ObjFlatNoQuotes(inter)
		} else if v.CastInt {
			if !cast.IsIntType(inter) {
				return "", ErrorInvalidPrimitiveType(inter, PrimTypeString, PrimTypeInt)
			}
			casted = s.ObjFlatNoQuotes(inter)
		} else {
			return "", ErrorInvalidPrimitiveType(inter, PrimTypeString)
		}
	}
	return ValidateStringProvided(casted, v)
}

func StringFromInterfaceMap(key string, iMap map[string]interface{}, v *StringValidation) (string, error) {
	inter, ok := ReadInterfaceMapValue(key, iMap)
	if !ok {
		val, err := ValidateStringMissing(v)
		if err != nil {
			return "", errors.Wrap(err, key)
		}
		return val, nil
	}
	val, err := String(inter, v)
	if err != nil {
		return "", errors.Wrap(err, key)
	}
	return val, nil
}

func StringFromStrMap(key string, sMap map[string]string, v *StringValidation) (string, error) {
	valStr, ok := sMap[key]
	if !ok {
		val, err := ValidateStringMissing(v)
		if err != nil {
			return "", errors.Wrap(err, key)
		}
		return val, nil
	}
	val, err := StringFromStr(valStr, v)
	if err != nil {
		return "", errors.Wrap(err, key)
	}
	return val, nil
}

func StringFromStr(valStr string, v *StringValidation) (string, error) {
	return ValidateStringProvided(valStr, v)
}

func StringFromEnv(envVarName string, v *StringValidation) (string, error) {
	valStr := ReadEnvVar(envVarName)
	if valStr == nil {
		val, err := ValidateStringMissing(v)
		if err != nil {
			return "", errors.Wrap(err, EnvVar(envVarName))
		}
		return val, nil
	}
	val, err := StringFromStr(*valStr, v)
	if err != nil {
		return "", errors.Wrap(err, EnvVar(envVarName))
	}
	return val, nil
}

func StringFromFile(filePath string, v *StringValidation) (string, error) {
	if !files.IsFile(filePath) {
		val, err := ValidateStringMissing(v)
		if err != nil {
			return "", errors.Wrap(err, filePath)
		}
		return val, nil
	}

	valStr, err := files.ReadFile(filePath)
	if err != nil {
		return "", err
	}
	if len(valStr) == 0 {
		val, err := ValidateStringMissing(v)
		if err != nil {
			return "", errors.Wrap(err, filePath)
		}
		return val, nil
	}

	val, err := StringFromStr(valStr, v)
	if err != nil {
		return "", errors.Wrap(err, filePath)
	}
	return val, nil
}

func StringFromEnvOrFile(envVarName string, filePath string, v *StringValidation) (string, error) {
	valStr := ReadEnvVar(envVarName)
	if valStr != nil {
		return StringFromEnv(envVarName, v)
	}
	return StringFromFile(filePath, v)
}

func StringFromPrompt(promptOpts *prompt.Options, v *StringValidation) (string, error) {
	promptOpts.DefaultStr = v.Default
	valStr := prompt.Prompt(promptOpts)
	if valStr == "" { // Treat empty prompt value as missing
		return ValidateStringMissing(v)
	}
	return StringFromStr(valStr, v)
}

func ValidateStringMissing(v *StringValidation) (string, error) {
	if v.Required {
		return "", ErrorMustBeDefined(v.AllowedValues)
	}
	return validateString(v.Default, v)
}

func ValidateStringProvided(val string, v *StringValidation) (string, error) {
	if v.CantBeSpecifiedErrStr != nil {
		return "", ErrorFieldCantBeSpecified(*v.CantBeSpecifiedErrStr)
	}
	return validateString(val, v)
}

func validateString(val string, v *StringValidation) (string, error) {
	err := ValidateStringVal(val, v)
	if err != nil {
		return "", err
	}

	if v.Validator != nil {
		return v.Validator(val)
	}
	return val, nil
}

func ValidateStringVal(val string, v *StringValidation) error {
	if v.RequireCortexResources {
		if err := checkOnlyCortexResources(val); err != nil {
			return err
		}
	} else if !v.AllowCortexResources {
		if err := checkNoCortexResources(val); err != nil {
			return err
		}
	}

	if !v.AllowEmpty {
		if len(val) == 0 {
			return ErrorCannotBeEmpty()
		}
	}

	if len(v.AllowedValues) > 0 {
		if !slices.HasString(v.AllowedValues, val) {
			return ErrorInvalidStr(val, v.AllowedValues[0], v.AllowedValues[1:]...)
		}
	}

	if len(v.DisallowedValues) > 0 {
		if slices.HasString(v.DisallowedValues, val) {
			return ErrorDisallowedValue(val)
		}
	}

	if v.MaxLength > 0 && len(val) > v.MaxLength {
		return ErrorTooLong(val, v.MaxLength)
	}

	if v.MinLength > 0 && len(val) < v.MinLength {
		return ErrorTooShort(val, v.MinLength)
	}

	if v.Prefix != "" {
		if !strings.HasPrefix(val, v.Prefix) {
			return ErrorMustHavePrefix(val, v.Prefix)
		}
	}

	for _, invalidPrefix := range v.InvalidPrefixes {
		if strings.HasPrefix(val, invalidPrefix) {
			return ErrorCantHavePrefix(val, invalidPrefix)
		}
	}

	if v.DisallowLeadingWhitespace {
		if regex.HasLeadingWhitespace(val) {
			return ErrorLeadingWhitespace(val)
		}
	}

	if v.DisallowTrailingWhitespace {
		if regex.HasTrailingWhitespace(val) {
			return ErrorTrailingWhitespace(val)
		}
	}

	if v.AlphaNumericDashDotUnderscore {
		if !regex.IsAlphaNumericDashDotUnderscore(val) {
			return ErrorAlphaNumericDashDotUnderscore(val)
		}
	}

	if v.AlphaNumericDashUnderscore {
		if !regex.IsAlphaNumericDashUnderscore(val) {
			return ErrorAlphaNumericDashUnderscore(val)
		}
	}

	if v.AlphaNumericDashUnderscoreOrEmpty {
		if !regex.IsAlphaNumericDashUnderscore(val) && val != "" {
			return ErrorAlphaNumericDashUnderscore(val)
		}
	}

	if v.AlphaNumericDashDotUnderscoreOrEmpty {
		if !regex.IsAlphaNumericDashDotUnderscore(val) && val != "" {
			return ErrorAlphaNumericDashDotUnderscore(val)
		}
	}

	if v.AWSTag {
		if !regex.IsValidAWSTag(val) && val != "" {
			return ErrorInvalidAWSTag(val)
		}
	}

	if v.DockerImageOrEmpty {
		if !regex.IsValidDockerImage(val) && val != "" {
			return ErrorInvalidDockerImage(val)
		}
	}

	if v.DNS1035 {
		if err := urls.CheckDNS1035(val); err != nil {
			return err
		}
	}

	if v.DNS1123 {
		if err := urls.CheckDNS1123(val); err != nil {
			return err
		}
	}

	return nil
}

//
// Musts
//

func MustStringFromEnv(envVarName string, v *StringValidation) string {
	val, err := StringFromEnv(envVarName, v)
	if err != nil {
		exit.Panic(err)
	}
	return val
}

func MustStringFromFile(filePath string, v *StringValidation) string {
	val, err := StringFromFile(filePath, v)
	if err != nil {
		exit.Panic(err)
	}
	return val
}

func MustStringFromEnvOrFile(envVarName string, filePath string, v *StringValidation) string {
	val, err := StringFromEnvOrFile(envVarName, filePath, v)
	if err != nil {
		exit.Panic(err)
	}
	return val
}
