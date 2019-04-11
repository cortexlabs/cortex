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
	"github.com/cortexlabs/cortex/pkg/lib/errors"
)

type BoolPtrValidation struct {
	Required     bool
	Default      *bool
	DisallowNull bool
}

func BoolPtr(inter interface{}, v *BoolPtrValidation) (*bool, error) {
	if inter == nil {
		return ValidateBoolPtr(nil, v)
	}
	casted, castOk := inter.(bool)
	if !castOk {
		return nil, ErrorInvalidPrimitiveType(inter, s.PrimTypeBool)
	}
	return ValidateBoolPtr(&casted, v)
}

func BoolPtrFromInterfaceMap(key string, iMap map[string]interface{}, v *BoolPtrValidation) (*bool, error) {
	inter, ok := ReadInterfaceMapValue(key, iMap)
	if !ok {
		val, err := ValidateBoolPtrMissing(v)
		if err != nil {
			return nil, errors.Wrap(err, key)
		}
		return val, nil
	}
	val, err := BoolPtr(inter, v)
	if err != nil {
		return nil, errors.Wrap(err, key)
	}
	return val, nil
}

func BoolPtrFromStrMap(key string, sMap map[string]string, v *BoolPtrValidation) (*bool, error) {
	valStr, ok := sMap[key]
	if !ok || valStr == "" {
		val, err := ValidateBoolPtrMissing(v)
		if err != nil {
			return nil, errors.Wrap(err, key)
		}
		return val, nil
	}
	val, err := BoolPtrFromStr(valStr, v)
	if err != nil {
		return nil, errors.Wrap(err, key)
	}
	return val, nil
}

func BoolPtrFromStr(valStr string, v *BoolPtrValidation) (*bool, error) {
	if valStr == "" {
		return ValidateBoolPtrMissing(v)
	}
	casted, castOk := s.ParseBool(valStr)
	if !castOk {
		return nil, ErrorInvalidPrimitiveType(valStr, s.PrimTypeBool)
	}
	return ValidateBoolPtr(&casted, v)
}

func BoolPtrFromEnv(envVarName string, v *BoolPtrValidation) (*bool, error) {
	valStr := ReadEnvVar(envVarName)
	if valStr == nil || *valStr == "" {
		val, err := ValidateBoolPtrMissing(v)
		if err != nil {
			return nil, errors.Wrap(err, s.EnvVar(envVarName))
		}
		return val, nil
	}
	val, err := BoolPtrFromStr(*valStr, v)
	if err != nil {
		return nil, errors.Wrap(err, s.EnvVar(envVarName))
	}
	return val, nil
}

func BoolPtrFromFile(filePath string, v *BoolPtrValidation) (*bool, error) {
	valBytes, err := ioutil.ReadFile(filePath)
	if err != nil || len(valBytes) == 0 {
		val, err := ValidateBoolPtrMissing(v)
		if err != nil {
			return nil, errors.Wrap(err, filePath)
		}
		return val, nil
	}
	valStr := string(valBytes)
	val, err := BoolPtrFromStr(valStr, v)
	if err != nil {
		return nil, errors.Wrap(err, filePath)
	}
	return val, nil
}

func BoolPtrFromEnvOrFile(envVarName string, filePath string, v *BoolPtrValidation) (*bool, error) {
	valStr := ReadEnvVar(envVarName)
	if valStr != nil && *valStr != "" {
		return BoolPtrFromEnv(envVarName, v)
	}
	return BoolPtrFromFile(filePath, v)
}

func BoolPtrFromPrompt(promptOpts *PromptOptions, v *BoolPtrValidation) (*bool, error) {
	valStr := prompt(promptOpts)
	if valStr == "" {
		return ValidateBoolPtrMissing(v)
	}
	return BoolPtrFromStr(valStr, v)
}

func ValidateBoolPtrMissing(v *BoolPtrValidation) (*bool, error) {
	if v.Required {
		return nil, ErrorMustBeDefined()
	}
	return ValidateBoolPtr(v.Default, v)
}

func ValidateBoolPtr(val *bool, v *BoolPtrValidation) (*bool, error) {
	if v.DisallowNull {
		if val == nil {
			return nil, ErrorCannotBeNull()
		}
	}

	return val, nil
}
