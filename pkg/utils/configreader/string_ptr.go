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
	"io/ioutil"

	s "github.com/cortexlabs/cortex/pkg/api/strings"
	"github.com/cortexlabs/cortex/pkg/utils/errors"
)

type StringPtrValidation struct {
	Required                      bool
	Default                       *string
	DisallowNull                  bool
	AllowEmpty                    bool
	AllowedValues                 []string
	Prefix                        string
	AlphaNumericDashDotUnderscore bool
	AlphaNumericDashUnderscore    bool
	Dns1035                       bool
	Validator                     func(*string) (*string, error)
}

func makeStringValValidation(v *StringPtrValidation) *StringValidation {
	return &StringValidation{
		AllowEmpty:                    v.AllowEmpty,
		AllowedValues:                 v.AllowedValues,
		Prefix:                        v.Prefix,
		AlphaNumericDashDotUnderscore: v.AlphaNumericDashDotUnderscore,
		AlphaNumericDashUnderscore:    v.AlphaNumericDashUnderscore,
		Dns1035:                       v.Dns1035,
	}
}

func StringPtr(inter interface{}, v *StringPtrValidation) (*string, error) {
	if inter == nil {
		return ValidateStringPtr(nil, v)
	}
	casted, castOk := inter.(string)
	if !castOk {
		return nil, errors.New(s.ErrInvalidPrimitiveType(inter, s.PrimTypeString))
	}
	return ValidateStringPtr(&casted, v)
}

func StringPtrFromInterfaceMap(key string, iMap map[string]interface{}, v *StringPtrValidation) (*string, error) {
	inter, ok := ReadInterfaceMapValue(key, iMap)
	if !ok {
		val, err := ValidateStringPtrMissing(v)
		if err != nil {
			return nil, errors.Wrap(err, key)
		}
		return val, nil
	}
	val, err := StringPtr(inter, v)
	if err != nil {
		return nil, errors.Wrap(err, key)
	}
	return val, nil
}

func StringPtrFromStrMap(key string, sMap map[string]string, v *StringPtrValidation) (*string, error) {
	valStr, ok := sMap[key]
	if !ok {
		val, err := ValidateStringPtrMissing(v)
		if err != nil {
			return nil, errors.Wrap(err, key)
		}
		return val, nil
	}
	val, err := StringPtrFromStr(valStr, v)
	if err != nil {
		return nil, errors.Wrap(err, key)
	}
	return val, nil
}

func StringPtrFromStr(str string, v *StringPtrValidation) (*string, error) {
	return ValidateStringPtr(&str, v)
}

func StringPtrFromEnv(envVarName string, v *StringPtrValidation) (*string, error) {
	valStr := ReadEnvVar(envVarName)
	if valStr == nil {
		val, err := ValidateStringPtrMissing(v)
		if err != nil {
			return nil, errors.Wrap(err, s.EnvVar(envVarName))
		}
		return val, nil
	}
	val, err := StringPtrFromStr(*valStr, v)
	if err != nil {
		return nil, errors.Wrap(err, s.EnvVar(envVarName))
	}
	return val, nil
}

func StringPtrFromFile(filePath string, v *StringPtrValidation) (*string, error) {
	valBytes, err := ioutil.ReadFile(filePath)
	if err != nil {
		val, err := ValidateStringPtrMissing(v)
		if err != nil {
			return nil, errors.Wrap(err, filePath)
		}
		return val, nil
	}
	valStr := string(valBytes)
	val, err := StringPtrFromStr(valStr, v)
	if err != nil {
		return nil, errors.Wrap(err, filePath)
	}
	return val, nil
}

func StringPtrFromEnvOrFile(envVarName string, filePath string, v *StringPtrValidation) (*string, error) {
	valStr := ReadEnvVar(envVarName)
	if valStr != nil {
		return StringPtrFromEnv(envVarName, v)
	}
	return StringPtrFromFile(filePath, v)
}

func StringPtrFromPrompt(promptOpts *PromptOptions, v *StringPtrValidation) (*string, error) {
	valStr := prompt(promptOpts)
	if valStr == "" { // Treat empty prompt value as missing
		ValidateStringPtrMissing(v)
	}
	return StringPtrFromStr(valStr, v)
}

func ValidateStringPtrMissing(v *StringPtrValidation) (*string, error) {
	if v.Required {
		return nil, errors.New(s.ErrMustBeDefined)
	}
	return ValidateStringPtr(v.Default, v)
}

func ValidateStringPtr(val *string, v *StringPtrValidation) (*string, error) {
	if v.DisallowNull {
		if val == nil {
			return nil, errors.New(s.ErrCannotBeNull)
		}
	}

	if val != nil {
		err := ValidateStringVal(*val, makeStringValValidation(v))
		if err != nil {
			return nil, err
		}
	}

	if v.Validator != nil {
		return v.Validator(val)
	}
	return val, nil
}
