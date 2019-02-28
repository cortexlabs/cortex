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
	"github.com/cortexlabs/cortex/pkg/lib/slices"
)

type Int64Validation struct {
	Required             bool
	Default              int64
	AllowedValues        []int64
	GreaterThan          *int64
	GreaterThanOrEqualTo *int64
	LessThan             *int64
	LessThanOrEqualTo    *int64
	Validator            func(int64) (int64, error)
}

func Int64(inter interface{}, v *Int64Validation) (int64, error) {
	if inter == nil {
		return 0, errors.New(s.ErrCannotBeNull)
	}
	casted, castOk := cast.InterfaceToInt64(inter)
	if !castOk {
		return 0, errors.New(s.ErrInvalidPrimitiveType(inter, s.PrimTypeInt))
	}
	return ValidateInt64(casted, v)
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
		return 0, errors.New(s.ErrInvalidPrimitiveType(valStr, s.PrimTypeInt))
	}
	return ValidateInt64(casted, v)
}

func Int64FromEnv(envVarName string, v *Int64Validation) (int64, error) {
	valStr := ReadEnvVar(envVarName)
	if valStr == nil || *valStr == "" {
		val, err := ValidateInt64Missing(v)
		if err != nil {
			return 0, errors.Wrap(err, s.EnvVar(envVarName))
		}
		return val, nil
	}
	val, err := Int64FromStr(*valStr, v)
	if err != nil {
		return 0, errors.Wrap(err, s.EnvVar(envVarName))
	}
	return val, nil
}

func Int64FromFile(filePath string, v *Int64Validation) (int64, error) {
	valBytes, err := ioutil.ReadFile(filePath)
	if err != nil || len(valBytes) == 0 {
		val, err := ValidateInt64Missing(v)
		if err != nil {
			return 0, errors.Wrap(err, filePath)
		}
		return val, nil
	}
	valStr := string(valBytes)
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

func Int64FromPrompt(promptOpts *PromptOptions, v *Int64Validation) (int64, error) {
	promptOpts.defaultStr = s.Int64(v.Default)
	valStr := prompt(promptOpts)
	if valStr == "" {
		return ValidateInt64Missing(v)
	}
	return Int64FromStr(valStr, v)
}

func ValidateInt64Missing(v *Int64Validation) (int64, error) {
	if v.Required {
		return 0, errors.New(s.ErrMustBeDefined)
	}
	return ValidateInt64(v.Default, v)
}

func ValidateInt64(val int64, v *Int64Validation) (int64, error) {
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
			return errors.New(s.ErrMustBeGreaterThan(val, *v.GreaterThan))
		}
	}
	if v.GreaterThanOrEqualTo != nil {
		if val < *v.GreaterThanOrEqualTo {
			return errors.New(s.ErrMustBeGreaterThanOrEqualTo(val, *v.GreaterThanOrEqualTo))
		}
	}
	if v.LessThan != nil {
		if val >= *v.LessThan {
			return errors.New(s.ErrMustBeLessThan(val, *v.LessThan))
		}
	}
	if v.LessThanOrEqualTo != nil {
		if val > *v.LessThanOrEqualTo {
			return errors.New(s.ErrMustBeLessThanOrEqualTo(val, *v.LessThanOrEqualTo))
		}
	}

	if v.AllowedValues != nil {
		if !slices.HasInt64(v.AllowedValues, val) {
			return errors.New(s.ErrInvalidInt64(val, v.AllowedValues...))
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
		errors.Panic(err)
	}
	return val
}

func MustInt64FromFile(filePath string, v *Int64Validation) int64 {
	val, err := Int64FromFile(filePath, v)
	if err != nil {
		errors.Panic(err)
	}
	return val
}

func MustInt64FromEnvOrFile(envVarName string, filePath string, v *Int64Validation) int64 {
	val, err := Int64FromEnvOrFile(envVarName, filePath, v)
	if err != nil {
		errors.Panic(err)
	}
	return val
}
