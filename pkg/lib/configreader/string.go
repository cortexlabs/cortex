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

package configreader

import (
	"fmt"
	"io/ioutil"
	"strings"

	"github.com/cortexlabs/cortex/pkg/lib/errors"
	"github.com/cortexlabs/cortex/pkg/lib/regex"
	"github.com/cortexlabs/cortex/pkg/lib/slices"
	"github.com/cortexlabs/cortex/pkg/lib/urls"
)

type StringValidation struct {
	Required                             bool
	Default                              string
	AllowEmpty                           bool
	AllowedValues                        []string
	Prefix                               string
	AlphaNumericDashDotUnderscoreOrEmpty bool
	AlphaNumericDashDotUnderscore        bool
	AlphaNumericDashUnderscore           bool
	DNS1035                              bool
	Validator                            func(string) (string, error)
}

func EnvVar(envVarName string) string {
	return fmt.Sprintf("environment variable \"%s\"", envVarName)
}

func String(inter interface{}, v *StringValidation) (string, error) {
	if inter == nil {
		return "", ErrorCannotBeNull()
	}
	casted, castOk := inter.(string)
	if !castOk {
		return "", ErrorInvalidPrimitiveType(inter, PrimTypeString)
	}
	return ValidateString(casted, v)
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
	return ValidateString(valStr, v)
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
	valBytes, err := ioutil.ReadFile(filePath)
	if err != nil {
		val, err := ValidateStringMissing(v)
		if err != nil {
			return "", errors.Wrap(err, filePath)
		}
		return val, nil
	}
	valStr := string(valBytes)
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

func StringFromPrompt(promptOpts *PromptOptions, v *StringValidation) (string, error) {
	promptOpts.defaultStr = v.Default
	valStr := prompt(promptOpts)
	if valStr == "" { // Treat empty prompt value as missing
		return ValidateStringMissing(v)
	}
	return StringFromStr(valStr, v)
}

func ValidateStringMissing(v *StringValidation) (string, error) {
	if v.Required {
		return "", ErrorMustBeDefined()
	}
	return ValidateString(v.Default, v)
}

func ValidateString(val string, v *StringValidation) (string, error) {
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
	if !v.AllowEmpty {
		if len(val) == 0 {
			return ErrorCannotBeEmpty()
		}
	}

	if v.AllowedValues != nil {
		if !slices.HasString(v.AllowedValues, val) {
			return ErrorInvalidStr(val, v.AllowedValues...)
		}
	}

	if v.Prefix != "" {
		if !strings.HasPrefix(val, v.Prefix) {
			return ErrorMustHavePrefix(val, v.Prefix)
		}
	}

	if v.AlphaNumericDashDotUnderscore {
		if !regex.CheckAlphaNumericDashDotUnderscore(val) {
			return ErrorAlphaNumericDashDotUnderscore(val)
		}
	}

	if v.AlphaNumericDashUnderscore {
		if !regex.CheckAlphaNumericDashUnderscore(val) {
			return ErrorAlphaNumericDashUnderscore(val)
		}
	}

	if v.AlphaNumericDashDotUnderscoreOrEmpty {
		if !regex.CheckAlphaNumericDashDotUnderscore(val) && val != "" {
			return ErrorAlphaNumericDashDotUnderscore(val)
		}
	}

	if v.DNS1035 {
		if err := urls.CheckDNS1035(val); err != nil {
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
		errors.Panic(err)
	}
	return val
}

func MustStringFromFile(filePath string, v *StringValidation) string {
	val, err := StringFromFile(filePath, v)
	if err != nil {
		errors.Panic(err)
	}
	return val
}

func MustStringFromEnvOrFile(envVarName string, filePath string, v *StringValidation) string {
	val, err := StringFromEnvOrFile(envVarName, filePath, v)
	if err != nil {
		errors.Panic(err)
	}
	return val
}
