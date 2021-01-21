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

type IntValidation struct {
	Required              bool
	Default               int
	TreatNullAsZero       bool // `<field>: ` and `<field>: null` will be read as `<field>: 0`
	AllowedValues         []int
	DisallowedValues      []int
	CantBeSpecifiedErrStr *string
	GreaterThan           *int
	GreaterThanOrEqualTo  *int
	LessThan              *int
	LessThanOrEqualTo     *int
	Validator             func(int) (int, error)
}

func Int(inter interface{}, v *IntValidation) (int, error) {
	if inter == nil {
		if v.TreatNullAsZero {
			return ValidateIntProvided(0, v)
		}
		return 0, ErrorCannotBeNull(v.Required)
	}
	casted, castOk := cast.InterfaceToInt(inter)
	if !castOk {
		return 0, ErrorInvalidPrimitiveType(inter, PrimTypeInt)
	}
	return ValidateIntProvided(casted, v)
}

func IntFromInterfaceMap(key string, iMap map[string]interface{}, v *IntValidation) (int, error) {
	inter, ok := ReadInterfaceMapValue(key, iMap)
	if !ok {
		val, err := ValidateIntMissing(v)
		if err != nil {
			return 0, errors.Wrap(err, key)
		}
		return val, nil
	}
	val, err := Int(inter, v)
	if err != nil {
		return 0, errors.Wrap(err, key)
	}
	return val, nil
}

func IntFromStrMap(key string, sMap map[string]string, v *IntValidation) (int, error) {
	valStr, ok := sMap[key]
	if !ok || valStr == "" {
		val, err := ValidateIntMissing(v)
		if err != nil {
			return 0, errors.Wrap(err, key)
		}
		return val, nil
	}
	val, err := IntFromStr(valStr, v)
	if err != nil {
		return 0, errors.Wrap(err, key)
	}
	return val, nil
}

func IntFromStr(valStr string, v *IntValidation) (int, error) {
	if valStr == "" {
		return ValidateIntMissing(v)
	}
	casted, castOk := s.ParseInt(valStr)
	if !castOk {
		return 0, ErrorInvalidPrimitiveType(valStr, PrimTypeInt)
	}
	return ValidateIntProvided(casted, v)
}

func IntFromEnv(envVarName string, v *IntValidation) (int, error) {
	valStr := ReadEnvVar(envVarName)
	if valStr == nil || *valStr == "" {
		val, err := ValidateIntMissing(v)
		if err != nil {
			return 0, errors.Wrap(err, EnvVar(envVarName))
		}
		return val, nil
	}
	val, err := IntFromStr(*valStr, v)
	if err != nil {
		return 0, errors.Wrap(err, EnvVar(envVarName))
	}
	return val, nil
}

func IntFromFile(filePath string, v *IntValidation) (int, error) {
	if !files.IsFile(filePath) {
		val, err := ValidateIntMissing(v)
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
		val, err := ValidateIntMissing(v)
		if err != nil {
			return 0, errors.Wrap(err, filePath)
		}
		return val, nil
	}

	val, err := IntFromStr(valStr, v)
	if err != nil {
		return 0, errors.Wrap(err, filePath)
	}
	return val, nil
}

func IntFromEnvOrFile(envVarName string, filePath string, v *IntValidation) (int, error) {
	valStr := ReadEnvVar(envVarName)
	if valStr != nil && *valStr != "" {
		return IntFromEnv(envVarName, v)
	}
	return IntFromFile(filePath, v)
}

func IntFromPrompt(promptOpts *prompt.Options, v *IntValidation) (int, error) {
	promptOpts.DefaultStr = s.Int(v.Default)
	valStr := prompt.Prompt(promptOpts)
	if valStr == "" {
		return ValidateIntMissing(v)
	}
	return IntFromStr(valStr, v)
}

func ValidateIntMissing(v *IntValidation) (int, error) {
	if v.Required {
		return 0, ErrorMustBeDefined(v.AllowedValues)
	}
	return validateInt(v.Default, v)
}

func ValidateIntProvided(val int, v *IntValidation) (int, error) {
	if v.CantBeSpecifiedErrStr != nil {
		return 0, ErrorFieldCantBeSpecified(*v.CantBeSpecifiedErrStr)
	}
	return validateInt(val, v)
}

func validateInt(val int, v *IntValidation) (int, error) {
	err := ValidateIntVal(val, v)
	if err != nil {
		return 0, err
	}

	if v.Validator != nil {
		return v.Validator(val)
	}
	return val, nil
}

func ValidateIntVal(val int, v *IntValidation) error {
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
		if !slices.HasInt(v.AllowedValues, val) {
			return ErrorInvalidInt(val, v.AllowedValues[0], v.AllowedValues[1:]...)
		}
	}

	if len(v.DisallowedValues) > 0 {
		if slices.HasInt(v.DisallowedValues, val) {
			return ErrorDisallowedValue(val)
		}
	}

	return nil
}

//
// Musts
//

func MustIntFromEnv(envVarName string, v *IntValidation) int {
	val, err := IntFromEnv(envVarName, v)
	if err != nil {
		exit.Panic(err)
	}
	return val
}

func MustIntFromFile(filePath string, v *IntValidation) int {
	val, err := IntFromFile(filePath, v)
	if err != nil {
		exit.Panic(err)
	}
	return val
}

func MustIntFromEnvOrFile(envVarName string, filePath string, v *IntValidation) int {
	val, err := IntFromEnvOrFile(envVarName, filePath, v)
	if err != nil {
		exit.Panic(err)
	}
	return val
}
