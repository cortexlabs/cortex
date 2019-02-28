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

type Int32Validation struct {
	Required             bool
	Default              int32
	AllowedValues        []int32
	GreaterThan          *int32
	GreaterThanOrEqualTo *int32
	LessThan             *int32
	LessThanOrEqualTo    *int32
	Validator            func(int32) (int32, error)
}

func Int32(inter interface{}, v *Int32Validation) (int32, error) {
	if inter == nil {
		return 0, errors.New(s.ErrCannotBeNull)
	}
	casted, castOk := cast.InterfaceToInt32(inter)
	if !castOk {
		return 0, errors.New(s.ErrInvalidPrimitiveType(inter, s.PrimTypeInt))
	}
	return ValidateInt32(casted, v)
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
		return 0, errors.New(s.ErrInvalidPrimitiveType(valStr, s.PrimTypeInt))
	}
	return ValidateInt32(casted, v)
}

func Int32FromEnv(envVarName string, v *Int32Validation) (int32, error) {
	valStr := ReadEnvVar(envVarName)
	if valStr == nil || *valStr == "" {
		val, err := ValidateInt32Missing(v)
		if err != nil {
			return 0, errors.Wrap(err, s.EnvVar(envVarName))
		}
		return val, nil
	}
	val, err := Int32FromStr(*valStr, v)
	if err != nil {
		return 0, errors.Wrap(err, s.EnvVar(envVarName))
	}
	return val, nil
}

func Int32FromFile(filePath string, v *Int32Validation) (int32, error) {
	valBytes, err := ioutil.ReadFile(filePath)
	if err != nil || len(valBytes) == 0 {
		val, err := ValidateInt32Missing(v)
		if err != nil {
			return 0, errors.Wrap(err, filePath)
		}
		return val, nil
	}
	valStr := string(valBytes)
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

func Int32FromPrompt(promptOpts *PromptOptions, v *Int32Validation) (int32, error) {
	promptOpts.defaultStr = s.Int32(v.Default)
	valStr := prompt(promptOpts)
	if valStr == "" {
		return ValidateInt32Missing(v)
	}
	return Int32FromStr(valStr, v)
}

func ValidateInt32Missing(v *Int32Validation) (int32, error) {
	if v.Required {
		return 0, errors.New(s.ErrMustBeDefined)
	}
	return ValidateInt32(v.Default, v)
}

func ValidateInt32(val int32, v *Int32Validation) (int32, error) {
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
		if !slices.HasInt32(v.AllowedValues, val) {
			return errors.New(s.ErrInvalidInt32(val, v.AllowedValues...))
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
		errors.Panic(err)
	}
	return val
}

func MustInt32FromFile(filePath string, v *Int32Validation) int32 {
	val, err := Int32FromFile(filePath, v)
	if err != nil {
		errors.Panic(err)
	}
	return val
}

func MustInt32FromEnvOrFile(envVarName string, filePath string, v *Int32Validation) int32 {
	val, err := Int32FromEnvOrFile(envVarName, filePath, v)
	if err != nil {
		errors.Panic(err)
	}
	return val
}
