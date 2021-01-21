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

type Float64PtrValidation struct {
	Required              bool
	Default               *float64
	AllowExplicitNull     bool
	AllowedValues         []float64
	DisallowedValues      []float64
	CantBeSpecifiedErrStr *string
	GreaterThan           *float64
	GreaterThanOrEqualTo  *float64
	LessThan              *float64
	LessThanOrEqualTo     *float64
	Validator             func(float64) (float64, error)
}

func makeFloat64ValValidation(v *Float64PtrValidation) *Float64Validation {
	return &Float64Validation{
		AllowedValues:        v.AllowedValues,
		DisallowedValues:     v.DisallowedValues,
		GreaterThan:          v.GreaterThan,
		GreaterThanOrEqualTo: v.GreaterThanOrEqualTo,
		LessThan:             v.LessThan,
		LessThanOrEqualTo:    v.LessThanOrEqualTo,
	}
}

func Float64Ptr(inter interface{}, v *Float64PtrValidation) (*float64, error) {
	if inter == nil {
		return ValidateFloat64PtrProvided(nil, v)
	}
	casted, castOk := cast.InterfaceToFloat64(inter)
	if !castOk {
		return nil, ErrorInvalidPrimitiveType(inter, PrimTypeFloat)
	}
	return ValidateFloat64PtrProvided(&casted, v)
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
		return nil, ErrorInvalidPrimitiveType(valStr, PrimTypeFloat)
	}
	return ValidateFloat64PtrProvided(&casted, v)
}

func Float64PtrFromEnv(envVarName string, v *Float64PtrValidation) (*float64, error) {
	valStr := ReadEnvVar(envVarName)
	if valStr == nil || *valStr == "" {
		val, err := ValidateFloat64PtrMissing(v)
		if err != nil {
			return nil, errors.Wrap(err, EnvVar(envVarName))
		}
		return val, nil
	}
	val, err := Float64PtrFromStr(*valStr, v)
	if err != nil {
		return nil, errors.Wrap(err, EnvVar(envVarName))
	}
	return val, nil
}

func Float64PtrFromFile(filePath string, v *Float64PtrValidation) (*float64, error) {
	if !files.IsFile(filePath) {
		val, err := ValidateFloat64PtrMissing(v)
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
		val, err := ValidateFloat64PtrMissing(v)
		if err != nil {
			return nil, errors.Wrap(err, filePath)
		}
		return val, nil
	}

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

func Float64PtrFromPrompt(promptOpts *prompt.Options, v *Float64PtrValidation) (*float64, error) {
	if v.Default != nil && promptOpts.DefaultStr == "" {
		promptOpts.DefaultStr = s.Float64(*v.Default)
	}
	valStr := prompt.Prompt(promptOpts)
	if valStr == "" {
		return ValidateFloat64PtrMissing(v)
	}
	return Float64PtrFromStr(valStr, v)
}

func ValidateFloat64PtrMissing(v *Float64PtrValidation) (*float64, error) {
	if v.Required {
		return nil, ErrorMustBeDefined(v.AllowedValues)
	}
	return validateFloat64Ptr(v.Default, v)
}

func ValidateFloat64PtrProvided(val *float64, v *Float64PtrValidation) (*float64, error) {
	if v.CantBeSpecifiedErrStr != nil {
		return nil, ErrorFieldCantBeSpecified(*v.CantBeSpecifiedErrStr)
	}

	if !v.AllowExplicitNull && val == nil {
		return nil, ErrorCannotBeNull(v.Required)
	}
	return validateFloat64Ptr(val, v)
}

func validateFloat64Ptr(val *float64, v *Float64PtrValidation) (*float64, error) {
	if val != nil {
		err := ValidateFloat64Val(*val, makeFloat64ValValidation(v))
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
