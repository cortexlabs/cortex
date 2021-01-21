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

type Float32Validation struct {
	Required              bool
	Default               float32
	TreatNullAsZero       bool // `<field>: ` and `<field>: null` will be read as `<field>: 0.0`
	AllowedValues         []float32
	DisallowedValues      []float32
	CantBeSpecifiedErrStr *string
	GreaterThan           *float32
	GreaterThanOrEqualTo  *float32
	LessThan              *float32
	LessThanOrEqualTo     *float32
	Validator             func(float32) (float32, error)
}

func Float32(inter interface{}, v *Float32Validation) (float32, error) {
	if inter == nil {
		if v.TreatNullAsZero {
			return ValidateFloat32Provided(0, v)
		}
		return 0, ErrorCannotBeNull(v.Required)
	}
	casted, castOk := cast.InterfaceToFloat32(inter)
	if !castOk {
		return 0, ErrorInvalidPrimitiveType(inter, PrimTypeFloat)
	}
	return ValidateFloat32Provided(casted, v)
}

func Float32FromInterfaceMap(key string, iMap map[string]interface{}, v *Float32Validation) (float32, error) {
	inter, ok := ReadInterfaceMapValue(key, iMap)
	if !ok {
		val, err := ValidateFloat32Missing(v)
		if err != nil {
			return 0, errors.Wrap(err, key)
		}
		return val, nil
	}
	val, err := Float32(inter, v)
	if err != nil {
		return 0, errors.Wrap(err, key)
	}
	return val, nil
}

func Float32FromStrMap(key string, sMap map[string]string, v *Float32Validation) (float32, error) {
	valStr, ok := sMap[key]
	if !ok || valStr == "" {
		val, err := ValidateFloat32Missing(v)
		if err != nil {
			return 0, errors.Wrap(err, key)
		}
		return val, nil
	}
	val, err := Float32FromStr(valStr, v)
	if err != nil {
		return 0, errors.Wrap(err, key)
	}
	return val, nil
}

func Float32FromStr(valStr string, v *Float32Validation) (float32, error) {
	if valStr == "" {
		return ValidateFloat32Missing(v)
	}
	casted, castOk := s.ParseFloat32(valStr)
	if !castOk {
		return 0, ErrorInvalidPrimitiveType(valStr, PrimTypeFloat)
	}
	return ValidateFloat32Provided(casted, v)
}

func Float32FromEnv(envVarName string, v *Float32Validation) (float32, error) {
	valStr := ReadEnvVar(envVarName)
	if valStr == nil || *valStr == "" {
		val, err := ValidateFloat32Missing(v)
		if err != nil {
			return 0, errors.Wrap(err, EnvVar(envVarName))
		}
		return val, nil
	}
	val, err := Float32FromStr(*valStr, v)
	if err != nil {
		return 0, errors.Wrap(err, EnvVar(envVarName))
	}
	return val, nil
}

func Float32FromFile(filePath string, v *Float32Validation) (float32, error) {
	if !files.IsFile(filePath) {
		val, err := ValidateFloat32Missing(v)
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
		val, err := ValidateFloat32Missing(v)
		if err != nil {
			return 0, errors.Wrap(err, filePath)
		}
		return val, nil
	}

	val, err := Float32FromStr(valStr, v)
	if err != nil {
		return 0, errors.Wrap(err, filePath)
	}
	return val, nil
}

func Float32FromEnvOrFile(envVarName string, filePath string, v *Float32Validation) (float32, error) {
	valStr := ReadEnvVar(envVarName)
	if valStr != nil && *valStr != "" {
		return Float32FromEnv(envVarName, v)
	}
	return Float32FromFile(filePath, v)
}

func Float32FromPrompt(promptOpts *prompt.Options, v *Float32Validation) (float32, error) {
	promptOpts.DefaultStr = s.Float32(v.Default)
	valStr := prompt.Prompt(promptOpts)
	if valStr == "" {
		return ValidateFloat32Missing(v)
	}
	return Float32FromStr(valStr, v)
}

func ValidateFloat32Missing(v *Float32Validation) (float32, error) {
	if v.Required {
		return 0, ErrorMustBeDefined(v.AllowedValues)
	}
	return validateFloat32(v.Default, v)
}

func ValidateFloat32Provided(val float32, v *Float32Validation) (float32, error) {
	if v.CantBeSpecifiedErrStr != nil {
		return 0, ErrorFieldCantBeSpecified(*v.CantBeSpecifiedErrStr)
	}
	return validateFloat32(val, v)
}

func validateFloat32(val float32, v *Float32Validation) (float32, error) {
	err := ValidateFloat32Val(val, v)
	if err != nil {
		return 0, err
	}

	if v.Validator != nil {
		return v.Validator(val)
	}
	return val, nil
}

func ValidateFloat32Val(val float32, v *Float32Validation) error {
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
		if !slices.HasFloat32(v.AllowedValues, val) {
			return ErrorInvalidFloat32(val, v.AllowedValues[0], v.AllowedValues[1:]...)
		}
	}

	if len(v.DisallowedValues) > 0 {
		if slices.HasFloat32(v.DisallowedValues, val) {
			return ErrorDisallowedValue(val)
		}
	}

	return nil
}

//
// Musts
//

func MustFloat32FromEnv(envVarName string, v *Float32Validation) float32 {
	val, err := Float32FromEnv(envVarName, v)
	if err != nil {
		exit.Panic(err)
	}
	return val
}

func MustFloat32FromFile(filePath string, v *Float32Validation) float32 {
	val, err := Float32FromFile(filePath, v)
	if err != nil {
		exit.Panic(err)
	}
	return val
}

func MustFloat32FromEnvOrFile(envVarName string, filePath string, v *Float32Validation) float32 {
	val, err := Float32FromEnvOrFile(envVarName, filePath, v)
	if err != nil {
		exit.Panic(err)
	}
	return val
}
