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
	"github.com/cortexlabs/cortex/pkg/lib/cast"
	"github.com/cortexlabs/cortex/pkg/lib/errors"
)

type Int64PtrValidation struct {
	Required             bool
	Default              *int64
	DisallowNull         bool
	AllowedValues        []int64
	GreaterThan          *int64
	GreaterThanOrEqualTo *int64
	LessThan             *int64
	LessThanOrEqualTo    *int64
	Validator            func(*int64) (*int64, error)
}

func makeInt64ValValidation(v *Int64PtrValidation) *Int64Validation {
	return &Int64Validation{
		AllowedValues:        v.AllowedValues,
		GreaterThan:          v.GreaterThan,
		GreaterThanOrEqualTo: v.GreaterThanOrEqualTo,
		LessThan:             v.LessThan,
		LessThanOrEqualTo:    v.LessThanOrEqualTo,
	}
}

func Int64Ptr(inter interface{}, v *Int64PtrValidation) (*int64, error) {
	if inter == nil {
		return ValidateInt64Ptr(nil, v)
	}
	casted, castOk := cast.InterfaceToInt64(inter)
	if !castOk {
		return nil, ErrorInvalidPrimitiveType(inter, s.PrimTypeInt)
	}
	return ValidateInt64Ptr(&casted, v)
}

func Int64PtrFromInterfaceMap(key string, iMap map[string]interface{}, v *Int64PtrValidation) (*int64, error) {
	inter, ok := ReadInterfaceMapValue(key, iMap)
	if !ok {
		val, err := ValidateInt64PtrMissing(v)
		if err != nil {
			return nil, errors.Wrap(err, key)
		}
		return val, nil
	}
	val, err := Int64Ptr(inter, v)
	if err != nil {
		return nil, errors.Wrap(err, key)
	}
	return val, nil
}

func Int64PtrFromStrMap(key string, sMap map[string]string, v *Int64PtrValidation) (*int64, error) {
	valStr, ok := sMap[key]
	if !ok || valStr == "" {
		val, err := ValidateInt64PtrMissing(v)
		if err != nil {
			return nil, errors.Wrap(err, key)
		}
		return val, nil
	}
	val, err := Int64PtrFromStr(valStr, v)
	if err != nil {
		return nil, errors.Wrap(err, key)
	}
	return val, nil
}

func Int64PtrFromStr(valStr string, v *Int64PtrValidation) (*int64, error) {
	if valStr == "" {
		return ValidateInt64PtrMissing(v)
	}
	casted, castOk := s.ParseInt64(valStr)
	if !castOk {
		return nil, ErrorInvalidPrimitiveType(valStr, s.PrimTypeInt)
	}
	return ValidateInt64Ptr(&casted, v)
}

func Int64PtrFromEnv(envVarName string, v *Int64PtrValidation) (*int64, error) {
	valStr := ReadEnvVar(envVarName)
	if valStr == nil || *valStr == "" {
		val, err := ValidateInt64PtrMissing(v)
		if err != nil {
			return nil, errors.Wrap(err, s.EnvVar(envVarName))
		}
		return val, nil
	}
	val, err := Int64PtrFromStr(*valStr, v)
	if err != nil {
		return nil, errors.Wrap(err, s.EnvVar(envVarName))
	}
	return val, nil
}

func Int64PtrFromFile(filePath string, v *Int64PtrValidation) (*int64, error) {
	valBytes, err := ioutil.ReadFile(filePath)
	if err != nil || len(valBytes) == 0 {
		val, err := ValidateInt64PtrMissing(v)
		if err != nil {
			return nil, errors.Wrap(err, filePath)
		}
		return val, nil
	}
	valStr := string(valBytes)
	val, err := Int64PtrFromStr(valStr, v)
	if err != nil {
		return nil, errors.Wrap(err, filePath)
	}
	return val, nil
}

func Int64PtrFromEnvOrFile(envVarName string, filePath string, v *Int64PtrValidation) (*int64, error) {
	valStr := ReadEnvVar(envVarName)
	if valStr != nil && *valStr != "" {
		return Int64PtrFromEnv(envVarName, v)
	}
	return Int64PtrFromFile(filePath, v)
}

func Int64PtrFromPrompt(promptOpts *PromptOptions, v *Int64PtrValidation) (*int64, error) {
	valStr := prompt(promptOpts)
	if valStr == "" {
		return ValidateInt64PtrMissing(v)
	}
	return Int64PtrFromStr(valStr, v)
}

func ValidateInt64PtrMissing(v *Int64PtrValidation) (*int64, error) {
	if v.Required {
		return nil, ErrorMustBeDefined()
	}
	return ValidateInt64Ptr(v.Default, v)
}

func ValidateInt64Ptr(val *int64, v *Int64PtrValidation) (*int64, error) {
	if v.DisallowNull {
		if val == nil {
			return nil, ErrorCannotBeNull()
		}
	}

	if val != nil {
		err := ValidateInt64Val(*val, makeInt64ValValidation(v))
		if err != nil {
			return nil, err
		}
	}

	if v.Validator != nil {
		return v.Validator(val)
	}
	return val, nil
}
