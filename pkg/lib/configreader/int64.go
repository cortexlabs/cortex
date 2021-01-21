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

type Int64Validation struct {
	Required              bool
	Default               int64
	TreatNullAsZero       bool // `<field>: ` and `<field>: null` will be read as `<field>: 0`
	AllowedValues         []int64
	DisallowedValues      []int64
	CantBeSpecifiedErrStr *string
	GreaterThan           *int64
	GreaterThanOrEqualTo  *int64
	LessThan              *int64
	LessThanOrEqualTo     *int64
	Validator             func(int64) (int64, error)
}

func Int64(inter interface{}, v *Int64Validation) (int64, error) {
	if inter == nil {
		if v.TreatNullAsZero {
			return ValidateInt64Provided(0, v)
		}
		return 0, ErrorCannotBeNull(v.Required)
	}
	casted, castOk := cast.InterfaceToInt64(inter)
	if !castOk {
		return 0, ErrorInvalidPrimitiveType(inter, PrimTypeInt)
	}
	return ValidateInt64Provided(casted, v)
}

func Int64FromInterfaceMap(key string, iMap map[string]interface{}, v *Int64Validation) (int64, error) {
	inter, ok := ReadInterfaceMapValue(key, iMap)
	if !ok {
		val, err := ValidateInt64Missing(v)
		if err != nil {
			return 0, errors.Wrap(err, key)
		}
		return val, nil
	}
	val, err := Int64(inter, v)
	if err != nil {
		return 0, errors.Wrap(err, key)
	}
	return val, nil
}

func Int64FromStrMap(key string, sMap map[string]string, v *Int64Validation) (int64, error) {
	valStr, ok := sMap[key]
	if !ok || valStr == "" {
		val, err := ValidateInt64Missing(v)
		if err != nil {
			return 0, errors.Wrap(err, key)
		}
		return val, nil
	}
	val, err := Int64FromStr(valStr, v)
	if err != nil {
		return 0, errors.Wrap(err, key)
	}
	return val, nil
}

func Int64FromStr(valStr string, v *Int64Validation) (int64, error) {
	if valStr == "" {
		return ValidateInt64Missing(v)
	}
	casted, castOk := s.ParseInt64(valStr)
	if !castOk {
		return 0, ErrorInvalidPrimitiveType(valStr, PrimTypeInt)
	}
	return ValidateInt64Provided(casted, v)
}

func Int64FromEnv(envVarName string, v *Int64Validation) (int64, error) {
	valStr := ReadEnvVar(envVarName)
	if valStr == nil || *valStr == "" {
		val, err := ValidateInt64Missing(v)
		if err != nil {
			return 0, errors.Wrap(err, EnvVar(envVarName))
		}
		return val, nil
	}
	val, err := Int64FromStr(*valStr, v)
	if err != nil {
		return 0, errors.Wrap(err, EnvVar(envVarName))
	}
	return val, nil
}

func Int64FromFile(filePath string, v *Int64Validation) (int64, error) {
	if !files.IsFile(filePath) {
		val, err := ValidateInt64Missing(v)
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
		val, err := ValidateInt64Missing(v)
		if err != nil {
			return 0, errors.Wrap(err, filePath)
		}
		return val, nil
	}

	val, err := Int64FromStr(valStr, v)
	if err != nil {
		return 0, errors.Wrap(err, filePath)
	}
	return val, nil
}

func Int64FromEnvOrFile(envVarName string, filePath string, v *Int64Validation) (int64, error) {
	valStr := ReadEnvVar(envVarName)
	if valStr != nil && *valStr != "" {
		return Int64FromEnv(envVarName, v)
	}
	return Int64FromFile(filePath, v)
}

func Int64FromPrompt(promptOpts *prompt.Options, v *Int64Validation) (int64, error) {
	promptOpts.DefaultStr = s.Int64(v.Default)
	valStr := prompt.Prompt(promptOpts)
	if valStr == "" {
		return ValidateInt64Missing(v)
	}
	return Int64FromStr(valStr, v)
}

func ValidateInt64Missing(v *Int64Validation) (int64, error) {
	if v.Required {
		return 0, ErrorMustBeDefined(v.AllowedValues)
	}
	return validateInt64(v.Default, v)
}

func ValidateInt64Provided(val int64, v *Int64Validation) (int64, error) {
	if v.CantBeSpecifiedErrStr != nil {
		return 0, ErrorFieldCantBeSpecified(*v.CantBeSpecifiedErrStr)
	}
	return validateInt64(val, v)
}

func validateInt64(val int64, v *Int64Validation) (int64, error) {
	err := ValidateInt64Val(val, v)
	if err != nil {
		return 0, err
	}

	if v.Validator != nil {
		return v.Validator(val)
	}
	return val, nil
}

func ValidateInt64Val(val int64, v *Int64Validation) error {
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
		if !slices.HasInt64(v.AllowedValues, val) {
			return ErrorInvalidInt64(val, v.AllowedValues[0], v.AllowedValues[1:]...)
		}
	}

	if len(v.DisallowedValues) > 0 {
		if slices.HasInt64(v.DisallowedValues, val) {
			return ErrorDisallowedValue(val)
		}
	}

	return nil
}

//
// Musts
//

func MustInt64FromEnv(envVarName string, v *Int64Validation) int64 {
	val, err := Int64FromEnv(envVarName, v)
	if err != nil {
		exit.Panic(err)
	}
	return val
}

func MustInt64FromFile(filePath string, v *Int64Validation) int64 {
	val, err := Int64FromFile(filePath, v)
	if err != nil {
		exit.Panic(err)
	}
	return val
}

func MustInt64FromEnvOrFile(envVarName string, filePath string, v *Int64Validation) int64 {
	val, err := Int64FromEnvOrFile(envVarName, filePath, v)
	if err != nil {
		exit.Panic(err)
	}
	return val
}
