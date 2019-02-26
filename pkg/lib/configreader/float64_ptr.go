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

type Float64PtrValidation struct {
	Required             bool
	Default              *float64
	DisallowNull         bool
	AllowedValues        []float64
	GreaterThan          *float64
	GreaterThanOrEqualTo *float64
	LessThan             *float64
	LessThanOrEqualTo    *float64
	Validator            func(*float64) (*float64, error)
}

func makeFloat64ValValidation(v *Float64PtrValidation) *Float64Validation {
	return &Float64Validation{
		AllowedValues:        v.AllowedValues,
		GreaterThan:          v.GreaterThan,
		GreaterThanOrEqualTo: v.GreaterThanOrEqualTo,
		LessThan:             v.LessThan,
		LessThanOrEqualTo:    v.LessThanOrEqualTo,
	}
}

func Float64Ptr(inter interface{}, v *Float64PtrValidation) (*float64, error) {
	if inter == nil {
		return ValidateFloat64Ptr(nil, v)
	}
	casted, castOk := cast.InterfaceToFloat64(inter)
	if !castOk {
		return nil, errors.New(s.ErrInvalidPrimitiveType(inter, s.PrimTypeFloat))
	}
	return ValidateFloat64Ptr(&casted, v)
}

func Float64PtrFromInterfaceMap(key string, iMap map[string]interface{}, v *Float64PtrValidation) (*float64, error) {
	inter, ok := ReadInterfaceMapValue(key, iMap)
	if !ok {
		val, err := ValidateFloat64PtrMissing(v)
		if err != nil {
			return nil, errors.Wrap(err, key)
		}
		return val, nil
	}
	val, err := Float64Ptr(inter, v)
	if err != nil {
		return nil, errors.Wrap(err, key)
	}
	return val, nil
}

func Float64PtrFromStrMap(key string, sMap map[string]string, v *Float64PtrValidation) (*float64, error) {
	valStr, ok := sMap[key]
	if !ok || valStr == "" {
		val, err := ValidateFloat64PtrMissing(v)
		if err != nil {
			return nil, errors.Wrap(err, key)
		}
		return val, nil
	}
	val, err := Float64PtrFromStr(valStr, v)
	if err != nil {
		return nil, errors.Wrap(err, key)
	}
	return val, nil
}

func Float64PtrFromStr(valStr string, v *Float64PtrValidation) (*float64, error) {
	if valStr == "" {
		return ValidateFloat64PtrMissing(v)
	}
	casted, castOk := s.ParseFloat64(valStr)
	if !castOk {
		return nil, errors.New(s.ErrInvalidPrimitiveType(valStr, s.PrimTypeFloat))
	}
	return ValidateFloat64Ptr(&casted, v)
}

func Float64PtrFromEnv(envVarName string, v *Float64PtrValidation) (*float64, error) {
	valStr := ReadEnvVar(envVarName)
	if valStr == nil || *valStr == "" {
		val, err := ValidateFloat64PtrMissing(v)
		if err != nil {
			return nil, errors.Wrap(err, s.EnvVar(envVarName))
		}
		return val, nil
	}
	val, err := Float64PtrFromStr(*valStr, v)
	if err != nil {
		return nil, errors.Wrap(err, s.EnvVar(envVarName))
	}
	return val, nil
}

func Float64PtrFromFile(filePath string, v *Float64PtrValidation) (*float64, error) {
	valBytes, err := ioutil.ReadFile(filePath)
	if err != nil || len(valBytes) == 0 {
		val, err := ValidateFloat64PtrMissing(v)
		if err != nil {
			return nil, errors.Wrap(err, filePath)
		}
		return val, nil
	}
	valStr := string(valBytes)
	val, err := Float64PtrFromStr(valStr, v)
	if err != nil {
		return nil, errors.Wrap(err, filePath)
	}
	return val, nil
}

func Float64PtrFromEnvOrFile(envVarName string, filePath string, v *Float64PtrValidation) (*float64, error) {
	valStr := ReadEnvVar(envVarName)
	if valStr != nil && *valStr != "" {
		return Float64PtrFromEnv(envVarName, v)
	}
	return Float64PtrFromFile(filePath, v)
}

func Float64PtrFromPrompt(promptOpts *PromptOptions, v *Float64PtrValidation) (*float64, error) {
	valStr := prompt(promptOpts)
	if valStr == "" {
		return ValidateFloat64PtrMissing(v)
	}
	return Float64PtrFromStr(valStr, v)
}

func ValidateFloat64PtrMissing(v *Float64PtrValidation) (*float64, error) {
	if v.Required {
		return nil, errors.New(s.ErrMustBeDefined)
	}
	return ValidateFloat64Ptr(v.Default, v)
}

func ValidateFloat64Ptr(val *float64, v *Float64PtrValidation) (*float64, error) {
	if v.DisallowNull {
		if val == nil {
			return nil, errors.New(s.ErrCannotBeNull)
		}
	}

	if val != nil {
		err := ValidateFloat64Val(*val, makeFloat64ValValidation(v))
		if err != nil {
			return nil, err
		}
	}

	if v.Validator != nil {
		return v.Validator(val)
	}
	return val, nil
}
