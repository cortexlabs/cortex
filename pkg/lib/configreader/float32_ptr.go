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
	"github.com/cortexlabs/cortex/pkg/lib/cast"
	"github.com/cortexlabs/cortex/pkg/lib/errors"
	"github.com/cortexlabs/cortex/pkg/lib/files"
	"github.com/cortexlabs/cortex/pkg/lib/prompt"
	s "github.com/cortexlabs/cortex/pkg/lib/strings"
)

type Float32PtrValidation struct {
	Required              bool
	Default               *float32
	AllowExplicitNull     bool
	AllowedValues         []float32
	DisallowedValues      []float32
	CantBeSpecifiedErrStr *string
	GreaterThan           *float32
	GreaterThanOrEqualTo  *float32
	LessThan              *float32
	LessThanOrEqualTo     *float32
	Validator             func(float32) (float32, error)
}

func makeFloat32ValValidation(v *Float32PtrValidation) *Float32Validation {
	return &Float32Validation{
		AllowedValues:        v.AllowedValues,
		DisallowedValues:     v.DisallowedValues,
		GreaterThan:          v.GreaterThan,
		GreaterThanOrEqualTo: v.GreaterThanOrEqualTo,
		LessThan:             v.LessThan,
		LessThanOrEqualTo:    v.LessThanOrEqualTo,
	}
}

func Float32Ptr(inter interface{}, v *Float32PtrValidation) (*float32, error) {
	if inter == nil {
		return ValidateFloat32PtrProvided(nil, v)
	}
	casted, castOk := cast.InterfaceToFloat32(inter)
	if !castOk {
		return nil, ErrorInvalidPrimitiveType(inter, PrimTypeFloat)
	}
	return ValidateFloat32PtrProvided(&casted, v)
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
		return nil, ErrorInvalidPrimitiveType(valStr, PrimTypeFloat)
	}
	return ValidateFloat32PtrProvided(&casted, v)
}

func Float32PtrFromEnv(envVarName string, v *Float32PtrValidation) (*float32, error) {
	valStr := ReadEnvVar(envVarName)
	if valStr == nil || *valStr == "" {
		val, err := ValidateFloat32PtrMissing(v)
		if err != nil {
			return nil, errors.Wrap(err, EnvVar(envVarName))
		}
		return val, nil
	}
	val, err := Float32PtrFromStr(*valStr, v)
	if err != nil {
		return nil, errors.Wrap(err, EnvVar(envVarName))
	}
	return val, nil
}

func Float32PtrFromFile(filePath string, v *Float32PtrValidation) (*float32, error) {
	if !files.IsFile(filePath) {
		val, err := ValidateFloat32PtrMissing(v)
		if err != nil {
			return nil, errors.Wrap(err, filePath)
		}
		return val, nil
	}

	valStr, err := files.ReadFile(filePath)
	if err != nil {
		return nil, err
	}
	if len(valStr) == 0 {
		val, err := ValidateFloat32PtrMissing(v)
		if err != nil {
			return nil, errors.Wrap(err, filePath)
		}
		return val, nil
	}

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

func Float32PtrFromPrompt(promptOpts *prompt.Options, v *Float32PtrValidation) (*float32, error) {
	if v.Default != nil && promptOpts.DefaultStr == "" {
		promptOpts.DefaultStr = s.Float32(*v.Default)
	}
	valStr := prompt.Prompt(promptOpts)
	if valStr == "" {
		return ValidateFloat32PtrMissing(v)
	}
	return Float32PtrFromStr(valStr, v)
}

func ValidateFloat32PtrMissing(v *Float32PtrValidation) (*float32, error) {
	if v.Required {
		return nil, ErrorMustBeDefined(v.AllowedValues)
	}
	return validateFloat32Ptr(v.Default, v)
}

func ValidateFloat32PtrProvided(val *float32, v *Float32PtrValidation) (*float32, error) {
	if v.CantBeSpecifiedErrStr != nil {
		return nil, ErrorFieldCantBeSpecified(*v.CantBeSpecifiedErrStr)
	}

	if !v.AllowExplicitNull && val == nil {
		return nil, ErrorCannotBeNull(v.Required)
	}
	return validateFloat32Ptr(val, v)
}

func validateFloat32Ptr(val *float32, v *Float32PtrValidation) (*float32, error) {
	if val != nil {
		err := ValidateFloat32Val(*val, makeFloat32ValValidation(v))
		if err != nil {
			return nil, err
		}
	}

	if val == nil {
		return val, nil
	}

	if v.Validator != nil {
		validated, err := v.Validator(*val)
		if err != nil {
			return nil, err
		}
		return &validated, nil
	}

	return val, nil
}
