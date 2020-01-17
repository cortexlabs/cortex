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

	"github.com/cortexlabs/cortex/pkg/lib/cast"
	"github.com/cortexlabs/cortex/pkg/lib/errors"
	"github.com/cortexlabs/cortex/pkg/lib/exit"
	"github.com/cortexlabs/cortex/pkg/lib/prompt"
	"github.com/cortexlabs/cortex/pkg/lib/slices"
	s "github.com/cortexlabs/cortex/pkg/lib/strings"
)

type Float32Validation struct {
	Required             bool
	Default              float32
	AllowedValues        []float32
	GreaterThan          *float32
	GreaterThanOrEqualTo *float32
	LessThan             *float32
	LessThanOrEqualTo    *float32
	Validator            func(float32) (float32, error)
}

func Float32(inter interface{}, v *Float32Validation) (float32, error) {
	if inter == nil {
		return 0, ErrorCannotBeNull()
	}
	casted, castOk := cast.InterfaceToFloat32(inter)
	if !castOk {
		return 0, ErrorInvalidPrimitiveType(inter, PrimTypeFloat)
	}
	return ValidateFloat32(casted, v)
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
	return ValidateFloat32(casted, v)
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
	valBytes, err := ioutil.ReadFile(filePath)
	if err != nil || len(valBytes) == 0 {
		val, err := ValidateFloat32Missing(v)
		if err != nil {
			return 0, errors.Wrap(err, filePath)
		}
		return val, nil
	}
	valStr := string(valBytes)
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
		return 0, ErrorMustBeDefined()
	}
	return ValidateFloat32(v.Default, v)
}

func ValidateFloat32(val float32, v *Float32Validation) (float32, error) {
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

	if v.AllowedValues != nil {
		if !slices.HasFloat32(v.AllowedValues, val) {
			return ErrorInvalidFloat32(val, v.AllowedValues[0], v.AllowedValues[1:]...)
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
