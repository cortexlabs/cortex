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
	"github.com/cortexlabs/cortex/pkg/lib/exit"
	"github.com/cortexlabs/cortex/pkg/lib/files"
	"github.com/cortexlabs/cortex/pkg/lib/prompt"
	"github.com/cortexlabs/cortex/pkg/lib/slices"
	s "github.com/cortexlabs/cortex/pkg/lib/strings"
)

type Int32Validation struct {
	Required              bool
	Default               int32
	TreatNullAsZero       bool // `<field>: ` and `<field>: null` will be read as `<field>: 0`
	AllowedValues         []int32
	DisallowedValues      []int32
	CantBeSpecifiedErrStr *string
	GreaterThan           *int32
	GreaterThanOrEqualTo  *int32
	LessThan              *int32
	LessThanOrEqualTo     *int32
	Validator             func(int32) (int32, error)
}

func Int32(inter interface{}, v *Int32Validation) (int32, error) {
	if inter == nil {
		if v.TreatNullAsZero {
			return ValidateInt32Provided(0, v)
		}
		return 0, ErrorCannotBeNull(v.Required)
	}
	casted, castOk := cast.InterfaceToInt32(inter)
	if !castOk {
		return 0, ErrorInvalidPrimitiveType(inter, PrimTypeInt)
	}
	return ValidateInt32Provided(casted, v)
}

func Int32FromInterfaceMap(key string, iMap map[string]interface{}, v *Int32Validation) (int32, error) {
	inter, ok := ReadInterfaceMapValue(key, iMap)
	if !ok {
		val, err := ValidateInt32Missing(v)
		if err != nil {
			return 0, errors.Wrap(err, key)
		}
		return val, nil
	}
	val, err := Int32(inter, v)
	if err != nil {
		return 0, errors.Wrap(err, key)
	}
	return val, nil
}

func Int32FromStrMap(key string, sMap map[string]string, v *Int32Validation) (int32, error) {
	valStr, ok := sMap[key]
	if !ok || valStr == "" {
		val, err := ValidateInt32Missing(v)
		if err != nil {
			return 0, errors.Wrap(err, key)
		}
		return val, nil
	}
	val, err := Int32FromStr(valStr, v)
	if err != nil {
		return 0, errors.Wrap(err, key)
	}
	return val, nil
}

func Int32FromStr(valStr string, v *Int32Validation) (int32, error) {
	if valStr == "" {
		return ValidateInt32Missing(v)
	}
	casted, castOk := s.ParseInt32(valStr)
	if !castOk {
		return 0, ErrorInvalidPrimitiveType(valStr, PrimTypeInt)
	}
	return ValidateInt32Provided(casted, v)
}

func Int32FromEnv(envVarName string, v *Int32Validation) (int32, error) {
	valStr := ReadEnvVar(envVarName)
	if valStr == nil || *valStr == "" {
		val, err := ValidateInt32Missing(v)
		if err != nil {
			return 0, errors.Wrap(err, EnvVar(envVarName))
		}
		return val, nil
	}
	val, err := Int32FromStr(*valStr, v)
	if err != nil {
		return 0, errors.Wrap(err, EnvVar(envVarName))
	}
	return val, nil
}

func Int32FromFile(filePath string, v *Int32Validation) (int32, error) {
	if !files.IsFile(filePath) {
		val, err := ValidateInt32Missing(v)
		if err != nil {
			return 0, errors.Wrap(err, filePath)
		}
		return val, nil
	}

	valStr, err := files.ReadFile(filePath)
	if err != nil {
		return 0, err
	}
	if len(valStr) == 0 {
		val, err := ValidateInt32Missing(v)
		if err != nil {
			return 0, errors.Wrap(err, filePath)
		}
		return val, nil
	}

	val, err := Int32FromStr(valStr, v)
	if err != nil {
		return 0, errors.Wrap(err, filePath)
	}
	return val, nil
}

func Int32FromEnvOrFile(envVarName string, filePath string, v *Int32Validation) (int32, error) {
	valStr := ReadEnvVar(envVarName)
	if valStr != nil && *valStr != "" {
		return Int32FromEnv(envVarName, v)
	}
	return Int32FromFile(filePath, v)
}

func Int32FromPrompt(promptOpts *prompt.Options, v *Int32Validation) (int32, error) {
	promptOpts.DefaultStr = s.Int32(v.Default)
	valStr := prompt.Prompt(promptOpts)
	if valStr == "" {
		return ValidateInt32Missing(v)
	}
	return Int32FromStr(valStr, v)
}

func ValidateInt32Missing(v *Int32Validation) (int32, error) {
	if v.Required {
		return 0, ErrorMustBeDefined(v.AllowedValues)
	}
	return validateInt32(v.Default, v)
}

func ValidateInt32Provided(val int32, v *Int32Validation) (int32, error) {
	if v.CantBeSpecifiedErrStr != nil {
		return 0, ErrorFieldCantBeSpecified(*v.CantBeSpecifiedErrStr)
	}
	return validateInt32(val, v)
}

func validateInt32(val int32, v *Int32Validation) (int32, error) {
	err := ValidateInt32Val(val, v)
	if err != nil {
		return 0, err
	}

	if v.Validator != nil {
		return v.Validator(val)
	}
	return val, nil
}

func ValidateInt32Val(val int32, v *Int32Validation) error {
	if v.GreaterThan != nil {
		if val <= *v.GreaterThan {
			return ErrorMustBeGreaterThan(val, *v.GreaterThan)
		}
	}
	if v.GreaterThanOrEqualTo != nil {
		if val < *v.GreaterThanOrEqualTo {
			return ErrorMustBeGreaterThanOrEqualTo(val, *v.GreaterThanOrEqualTo)
		}
	}
	if v.LessThan != nil {
		if val >= *v.LessThan {
			return ErrorMustBeLessThan(val, *v.LessThan)
		}
	}
	if v.LessThanOrEqualTo != nil {
		if val > *v.LessThanOrEqualTo {
			return ErrorMustBeLessThanOrEqualTo(val, *v.LessThanOrEqualTo)
		}
	}

	if len(v.AllowedValues) > 0 {
		if !slices.HasInt32(v.AllowedValues, val) {
			return ErrorInvalidInt32(val, v.AllowedValues[0], v.AllowedValues[1:]...)
		}
	}

	if len(v.DisallowedValues) > 0 {
		if slices.HasInt32(v.DisallowedValues, val) {
			return ErrorDisallowedValue(val)
		}
	}

	return nil
}

//
// Musts
//

func MustInt32FromEnv(envVarName string, v *Int32Validation) int32 {
	val, err := Int32FromEnv(envVarName, v)
	if err != nil {
		exit.Panic(err)
	}
	return val
}

func MustInt32FromFile(filePath string, v *Int32Validation) int32 {
	val, err := Int32FromFile(filePath, v)
	if err != nil {
		exit.Panic(err)
	}
	return val
}

func MustInt32FromEnvOrFile(envVarName string, filePath string, v *Int32Validation) int32 {
	val, err := Int32FromEnvOrFile(envVarName, filePath, v)
	if err != nil {
		exit.Panic(err)
	}
	return val
}
