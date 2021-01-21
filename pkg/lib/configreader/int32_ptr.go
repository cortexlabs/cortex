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

type Int32PtrValidation struct {
	Required              bool
	Default               *int32
	AllowExplicitNull     bool
	AllowedValues         []int32
	DisallowedValues      []int32
	CantBeSpecifiedErrStr *string
	GreaterThan           *int32
	GreaterThanOrEqualTo  *int32
	LessThan              *int32
	LessThanOrEqualTo     *int32
	Validator             func(int32) (int32, error)
}

func makeInt32ValValidation(v *Int32PtrValidation) *Int32Validation {
	return &Int32Validation{
		AllowedValues:        v.AllowedValues,
		DisallowedValues:     v.DisallowedValues,
		GreaterThan:          v.GreaterThan,
		GreaterThanOrEqualTo: v.GreaterThanOrEqualTo,
		LessThan:             v.LessThan,
		LessThanOrEqualTo:    v.LessThanOrEqualTo,
	}
}

func Int32Ptr(inter interface{}, v *Int32PtrValidation) (*int32, error) {
	if inter == nil {
		return ValidateInt32PtrProvided(nil, v)
	}
	casted, castOk := cast.InterfaceToInt32(inter)
	if !castOk {
		return nil, ErrorInvalidPrimitiveType(inter, PrimTypeInt)
	}
	return ValidateInt32PtrProvided(&casted, v)
}

func Int32PtrFromInterfaceMap(key string, iMap map[string]interface{}, v *Int32PtrValidation) (*int32, error) {
	inter, ok := ReadInterfaceMapValue(key, iMap)
	if !ok {
		val, err := ValidateInt32PtrMissing(v)
		if err != nil {
			return nil, errors.Wrap(err, key)
		}
		return val, nil
	}
	val, err := Int32Ptr(inter, v)
	if err != nil {
		return nil, errors.Wrap(err, key)
	}
	return val, nil
}

func Int32PtrFromStrMap(key string, sMap map[string]string, v *Int32PtrValidation) (*int32, error) {
	valStr, ok := sMap[key]
	if !ok || valStr == "" {
		val, err := ValidateInt32PtrMissing(v)
		if err != nil {
			return nil, errors.Wrap(err, key)
		}
		return val, nil
	}
	val, err := Int32PtrFromStr(valStr, v)
	if err != nil {
		return nil, errors.Wrap(err, key)
	}
	return val, nil
}

func Int32PtrFromStr(valStr string, v *Int32PtrValidation) (*int32, error) {
	if valStr == "" {
		return ValidateInt32PtrMissing(v)
	}
	casted, castOk := s.ParseInt32(valStr)
	if !castOk {
		return nil, ErrorInvalidPrimitiveType(valStr, PrimTypeInt)
	}
	return ValidateInt32PtrProvided(&casted, v)
}

func Int32PtrFromEnv(envVarName string, v *Int32PtrValidation) (*int32, error) {
	valStr := ReadEnvVar(envVarName)
	if valStr == nil || *valStr == "" {
		val, err := ValidateInt32PtrMissing(v)
		if err != nil {
			return nil, errors.Wrap(err, EnvVar(envVarName))
		}
		return val, nil
	}
	val, err := Int32PtrFromStr(*valStr, v)
	if err != nil {
		return nil, errors.Wrap(err, EnvVar(envVarName))
	}
	return val, nil
}

func Int32PtrFromFile(filePath string, v *Int32PtrValidation) (*int32, error) {
	if !files.IsFile(filePath) {
		val, err := ValidateInt32PtrMissing(v)
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
		val, err := ValidateInt32PtrMissing(v)
		if err != nil {
			return nil, errors.Wrap(err, filePath)
		}
		return val, nil
	}

	val, err := Int32PtrFromStr(valStr, v)
	if err != nil {
		return nil, errors.Wrap(err, filePath)
	}
	return val, nil
}

func Int32PtrFromEnvOrFile(envVarName string, filePath string, v *Int32PtrValidation) (*int32, error) {
	valStr := ReadEnvVar(envVarName)
	if valStr != nil && *valStr != "" {
		return Int32PtrFromEnv(envVarName, v)
	}
	return Int32PtrFromFile(filePath, v)
}

func Int32PtrFromPrompt(promptOpts *prompt.Options, v *Int32PtrValidation) (*int32, error) {
	if v.Default != nil && promptOpts.DefaultStr == "" {
		promptOpts.DefaultStr = s.Int32(*v.Default)
	}
	valStr := prompt.Prompt(promptOpts)
	if valStr == "" {
		return ValidateInt32PtrMissing(v)
	}
	return Int32PtrFromStr(valStr, v)
}

func ValidateInt32PtrMissing(v *Int32PtrValidation) (*int32, error) {
	if v.Required {
		return nil, ErrorMustBeDefined(v.AllowedValues)
	}
	return validateInt32Ptr(v.Default, v)
}

func ValidateInt32PtrProvided(val *int32, v *Int32PtrValidation) (*int32, error) {
	if v.CantBeSpecifiedErrStr != nil {
		return nil, ErrorFieldCantBeSpecified(*v.CantBeSpecifiedErrStr)
	}

	if !v.AllowExplicitNull && val == nil {
		return nil, ErrorCannotBeNull(v.Required)
	}
	return validateInt32Ptr(val, v)
}

func validateInt32Ptr(val *int32, v *Int32PtrValidation) (*int32, error) {
	if val != nil {
		err := ValidateInt32Val(*val, makeInt32ValValidation(v))
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
