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

type Float32PtrValidation struct {
	Required             bool
	Default              *float32
	DisallowNull         bool
	AllowedValues        []float32
	GreaterThan          *float32
	GreaterThanOrEqualTo *float32
	LessThan             *float32
	LessThanOrEqualTo    *float32
	Validator            func(*float32) (*float32, error)
}

func makeFloat32ValValidation(v *Float32PtrValidation) *Float32Validation {
	return &Float32Validation{
		AllowedValues:        v.AllowedValues,
		GreaterThan:          v.GreaterThan,
		GreaterThanOrEqualTo: v.GreaterThanOrEqualTo,
		LessThan:             v.LessThan,
		LessThanOrEqualTo:    v.LessThanOrEqualTo,
	}
}

func Float32Ptr(inter interface{}, v *Float32PtrValidation) (*float32, error) {
	if inter == nil {
		return ValidateFloat32Ptr(nil, v)
	}
	casted, castOk := cast.InterfaceToFloat32(inter)
	if !castOk {
		return nil, errors.New(s.ErrInvalidPrimitiveType(inter, s.PrimTypeFloat))
	}
	return ValidateFloat32Ptr(&casted, v)
}

func Float32PtrFromInterfaceMap(key string, iMap map[string]interface{}, v *Float32PtrValidation) (*float32, error) {
	inter, ok := ReadInterfaceMapValue(key, iMap)
	if !ok {
		val, err := ValidateFloat32PtrMissing(v)
		if err != nil {
			return nil, errors.Wrap(err, key)
		}
		return val, nil
	}
	val, err := Float32Ptr(inter, v)
	if err != nil {
		return nil, errors.Wrap(err, key)
	}
	return val, nil
}

func Float32PtrFromStrMap(key string, sMap map[string]string, v *Float32PtrValidation) (*float32, error) {
	valStr, ok := sMap[key]
	if !ok || valStr == "" {
		val, err := ValidateFloat32PtrMissing(v)
		if err != nil {
			return nil, errors.Wrap(err, key)
		}
		return val, nil
	}
	val, err := Float32PtrFromStr(valStr, v)
	if err != nil {
		return nil, errors.Wrap(err, key)
	}
	return val, nil
}

func Float32PtrFromStr(valStr string, v *Float32PtrValidation) (*float32, error) {
	if valStr == "" {
		return ValidateFloat32PtrMissing(v)
	}
	casted, castOk := s.ParseFloat32(valStr)
	if !castOk {
		return nil, errors.New(s.ErrInvalidPrimitiveType(valStr, s.PrimTypeFloat))
	}
	return ValidateFloat32Ptr(&casted, v)
}

func Float32PtrFromEnv(envVarName string, v *Float32PtrValidation) (*float32, error) {
	valStr := ReadEnvVar(envVarName)
	if valStr == nil || *valStr == "" {
		val, err := ValidateFloat32PtrMissing(v)
		if err != nil {
			return nil, errors.Wrap(err, s.EnvVar(envVarName))
		}
		return val, nil
	}
	val, err := Float32PtrFromStr(*valStr, v)
	if err != nil {
		return nil, errors.Wrap(err, s.EnvVar(envVarName))
	}
	return val, nil
}

func Float32PtrFromFile(filePath string, v *Float32PtrValidation) (*float32, error) {
	valBytes, err := ioutil.ReadFile(filePath)
	if err != nil || len(valBytes) == 0 {
		val, err := ValidateFloat32PtrMissing(v)
		if err != nil {
			return nil, errors.Wrap(err, filePath)
		}
		return val, nil
	}
	valStr := string(valBytes)
	val, err := Float32PtrFromStr(valStr, v)
	if err != nil {
		return nil, errors.Wrap(err, filePath)
	}
	return val, nil
}

func Float32PtrFromEnvOrFile(envVarName string, filePath string, v *Float32PtrValidation) (*float32, error) {
	valStr := ReadEnvVar(envVarName)
	if valStr != nil && *valStr != "" {
		return Float32PtrFromEnv(envVarName, v)
	}
	return Float32PtrFromFile(filePath, v)
}

func Float32PtrFromPrompt(promptOpts *PromptOptions, v *Float32PtrValidation) (*float32, error) {
	valStr := prompt(promptOpts)
	if valStr == "" {
		return ValidateFloat32PtrMissing(v)
	}
	return Float32PtrFromStr(valStr, v)
}

func ValidateFloat32PtrMissing(v *Float32PtrValidation) (*float32, error) {
	if v.Required {
		return nil, errors.New(s.ErrMustBeDefined)
	}
	return ValidateFloat32Ptr(v.Default, v)
}

func ValidateFloat32Ptr(val *float32, v *Float32PtrValidation) (*float32, error) {
	if v.DisallowNull {
		if val == nil {
			return nil, errors.New(s.ErrCannotBeNull)
		}
	}

	if val != nil {
		err := ValidateFloat32Val(*val, makeFloat32ValValidation(v))
		if err != nil {
			return nil, err
		}
	}

	if v.Validator != nil {
		return v.Validator(val)
	}
	return val, nil
}
