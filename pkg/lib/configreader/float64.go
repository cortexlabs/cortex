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

type Float64Validation struct {
	Required             bool
	Default              float64
	AllowedValues        []float64
	GreaterThan          *float64
	GreaterThanOrEqualTo *float64
	LessThan             *float64
	LessThanOrEqualTo    *float64
	Validator            func(float64) (float64, error)
}

func Float64(inter interface{}, v *Float64Validation) (float64, error) {
	if inter == nil {
		return 0, ErrorCannotBeNull()
	}
	casted, castOk := cast.InterfaceToFloat64(inter)
	if !castOk {
		return 0, ErrorInvalidPrimitiveType(inter, PrimTypeFloat)
	}
	return ValidateFloat64(casted, v)
}

func Float64FromInterfaceMap(key string, iMap map[string]interface{}, v *Float64Validation) (float64, error) {
	inter, ok := ReadInterfaceMapValue(key, iMap)
	if !ok {
		val, err := ValidateFloat64Missing(v)
		if err != nil {
			return 0, errors.Wrap(err, key)
		}
		return val, nil
	}
	val, err := Float64(inter, v)
	if err != nil {
		return 0, errors.Wrap(err, key)
	}
	return val, nil
}

func Float64FromStrMap(key string, sMap map[string]string, v *Float64Validation) (float64, error) {
	valStr, ok := sMap[key]
	if !ok || valStr == "" {
		val, err := ValidateFloat64Missing(v)
		if err != nil {
			return 0, errors.Wrap(err, key)
		}
		return val, nil
	}
	val, err := Float64FromStr(valStr, v)
	if err != nil {
		return 0, errors.Wrap(err, key)
	}
	return val, nil
}

func Float64FromStr(valStr string, v *Float64Validation) (float64, error) {
	if valStr == "" {
		return ValidateFloat64Missing(v)
	}
	casted, castOk := s.ParseFloat64(valStr)
	if !castOk {
		return 0, ErrorInvalidPrimitiveType(valStr, PrimTypeFloat)
	}
	return ValidateFloat64(casted, v)
}

func Float64FromEnv(envVarName string, v *Float64Validation) (float64, error) {
	valStr := ReadEnvVar(envVarName)
	if valStr == nil || *valStr == "" {
		val, err := ValidateFloat64Missing(v)
		if err != nil {
			return 0, errors.Wrap(err, EnvVar(envVarName))
		}
		return val, nil
	}
	val, err := Float64FromStr(*valStr, v)
	if err != nil {
		return 0, errors.Wrap(err, EnvVar(envVarName))
	}
	return val, nil
}

func Float64FromFile(filePath string, v *Float64Validation) (float64, error) {
	valBytes, err := ioutil.ReadFile(filePath)
	if err != nil || len(valBytes) == 0 {
		val, err := ValidateFloat64Missing(v)
		if err != nil {
			return 0, errors.Wrap(err, filePath)
		}
		return val, nil
	}
	valStr := string(valBytes)
	val, err := Float64FromStr(valStr, v)
	if err != nil {
		return 0, errors.Wrap(err, filePath)
	}
	return val, nil
}

func Float64FromEnvOrFile(envVarName string, filePath string, v *Float64Validation) (float64, error) {
	valStr := ReadEnvVar(envVarName)
	if valStr != nil && *valStr != "" {
		return Float64FromEnv(envVarName, v)
	}
	return Float64FromFile(filePath, v)
}

func Float64FromPrompt(promptOpts *prompt.Options, v *Float64Validation) (float64, error) {
	promptOpts.DefaultStr = s.Float64(v.Default)
	valStr := prompt.Prompt(promptOpts)
	if valStr == "" {
		return ValidateFloat64Missing(v)
	}
	return Float64FromStr(valStr, v)
}

func ValidateFloat64Missing(v *Float64Validation) (float64, error) {
	if v.Required {
		return 0, ErrorMustBeDefined()
	}
	return ValidateFloat64(v.Default, v)
}

func ValidateFloat64(val float64, v *Float64Validation) (float64, error) {
	err := ValidateFloat64Val(val, v)
	if err != nil {
		return 0, err
	}

	if v.Validator != nil {
		return v.Validator(val)
	}
	return val, nil
}

func ValidateFloat64Val(val float64, v *Float64Validation) error {
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
		if !slices.HasFloat64(v.AllowedValues, val) {
			return ErrorInvalidFloat64(val, v.AllowedValues...)
		}
	}

	return nil
}

//
// Musts
//

func MustFloat64FromEnv(envVarName string, v *Float64Validation) float64 {
	val, err := Float64FromEnv(envVarName, v)
	if err != nil {
		exit.Panic(err)
	}
	return val
}

func MustFloat64FromFile(filePath string, v *Float64Validation) float64 {
	val, err := Float64FromFile(filePath, v)
	if err != nil {
		exit.Panic(err)
	}
	return val
}

func MustFloat64FromEnvOrFile(envVarName string, filePath string, v *Float64Validation) float64 {
	val, err := Float64FromEnvOrFile(envVarName, filePath, v)
	if err != nil {
		exit.Panic(err)
	}
	return val
}
