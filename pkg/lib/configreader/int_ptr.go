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

type IntPtrValidation struct {
	Required              bool
	Default               *int
	AllowExplicitNull     bool
	AllowedValues         []int
	DisallowedValues      []int
	CantBeSpecifiedErrStr *string
	GreaterThan           *int
	GreaterThanOrEqualTo  *int
	LessThan              *int
	LessThanOrEqualTo     *int
	Validator             func(int) (int, error)
}

func makeIntValValidation(v *IntPtrValidation) *IntValidation {
	return &IntValidation{
		AllowedValues:        v.AllowedValues,
		DisallowedValues:     v.DisallowedValues,
		GreaterThan:          v.GreaterThan,
		GreaterThanOrEqualTo: v.GreaterThanOrEqualTo,
		LessThan:             v.LessThan,
		LessThanOrEqualTo:    v.LessThanOrEqualTo,
	}
}

func IntPtr(inter interface{}, v *IntPtrValidation) (*int, error) {
	if inter == nil {
		return ValidateIntPtrProvided(nil, v)
	}
	casted, castOk := cast.InterfaceToInt(inter)
	if !castOk {
		return nil, ErrorInvalidPrimitiveType(inter, PrimTypeInt)
	}
	return ValidateIntPtrProvided(&casted, v)
}

func IntPtrFromInterfaceMap(key string, iMap map[string]interface{}, v *IntPtrValidation) (*int, error) {
	inter, ok := ReadInterfaceMapValue(key, iMap)
	if !ok {
		val, err := ValidateIntPtrMissing(v)
		if err != nil {
			return nil, errors.Wrap(err, key)
		}
		return val, nil
	}
	val, err := IntPtr(inter, v)
	if err != nil {
		return nil, errors.Wrap(err, key)
	}
	return val, nil
}

func IntPtrFromStrMap(key string, sMap map[string]string, v *IntPtrValidation) (*int, error) {
	valStr, ok := sMap[key]
	if !ok || valStr == "" {
		val, err := ValidateIntPtrMissing(v)
		if err != nil {
			return nil, errors.Wrap(err, key)
		}
		return val, nil
	}
	val, err := IntPtrFromStr(valStr, v)
	if err != nil {
		return nil, errors.Wrap(err, key)
	}
	return val, nil
}

func IntPtrFromStr(valStr string, v *IntPtrValidation) (*int, error) {
	if valStr == "" {
		return ValidateIntPtrMissing(v)
	}
	casted, castOk := s.ParseInt(valStr)
	if !castOk {
		return nil, ErrorInvalidPrimitiveType(valStr, PrimTypeInt)
	}
	return ValidateIntPtrProvided(&casted, v)
}

func IntPtrFromEnv(envVarName string, v *IntPtrValidation) (*int, error) {
	valStr := ReadEnvVar(envVarName)
	if valStr == nil || *valStr == "" {
		val, err := ValidateIntPtrMissing(v)
		if err != nil {
			return nil, errors.Wrap(err, EnvVar(envVarName))
		}
		return val, nil
	}
	val, err := IntPtrFromStr(*valStr, v)
	if err != nil {
		return nil, errors.Wrap(err, EnvVar(envVarName))
	}
	return val, nil
}

func IntPtrFromFile(filePath string, v *IntPtrValidation) (*int, error) {
	if !files.IsFile(filePath) {
		val, err := ValidateIntPtrMissing(v)
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
		val, err := ValidateIntPtrMissing(v)
		if err != nil {
			return nil, errors.Wrap(err, filePath)
		}
		return val, nil
	}

	val, err := IntPtrFromStr(valStr, v)
	if err != nil {
		return nil, errors.Wrap(err, filePath)
	}
	return val, nil
}

func IntPtrFromEnvOrFile(envVarName string, filePath string, v *IntPtrValidation) (*int, error) {
	valStr := ReadEnvVar(envVarName)
	if valStr != nil && *valStr != "" {
		return IntPtrFromEnv(envVarName, v)
	}
	return IntPtrFromFile(filePath, v)
}

func IntPtrFromPrompt(promptOpts *prompt.Options, v *IntPtrValidation) (*int, error) {
	if v.Default != nil && promptOpts.DefaultStr == "" {
		promptOpts.DefaultStr = s.Int(*v.Default)
	}
	valStr := prompt.Prompt(promptOpts)
	if valStr == "" {
		return ValidateIntPtrMissing(v)
	}
	return IntPtrFromStr(valStr, v)
}

func ValidateIntPtrMissing(v *IntPtrValidation) (*int, error) {
	if v.Required {
		return nil, ErrorMustBeDefined(v.AllowedValues)
	}
	return validateIntPtr(v.Default, v)
}

func ValidateIntPtrProvided(val *int, v *IntPtrValidation) (*int, error) {
	if v.CantBeSpecifiedErrStr != nil {
		return nil, ErrorFieldCantBeSpecified(*v.CantBeSpecifiedErrStr)
	}

	if !v.AllowExplicitNull && val == nil {
		return nil, ErrorCannotBeNull(v.Required)
	}
	return validateIntPtr(val, v)
}

func validateIntPtr(val *int, v *IntPtrValidation) (*int, error) {
	if val != nil {
		err := ValidateIntVal(*val, makeIntValValidation(v))
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
