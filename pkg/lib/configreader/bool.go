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
	"strings"

	"github.com/cortexlabs/cortex/pkg/lib/errors"
	"github.com/cortexlabs/cortex/pkg/lib/exit"
	"github.com/cortexlabs/cortex/pkg/lib/files"
	"github.com/cortexlabs/cortex/pkg/lib/prompt"
	s "github.com/cortexlabs/cortex/pkg/lib/strings"
)

type BoolValidation struct {
	Required              bool
	Default               bool
	TreatNullAsFalse      bool // `<field>: ` and `<field>: null` will be read as `<field>: false`
	CantBeSpecifiedErrStr *string
	StrToBool             map[string]bool // lowercase
}

func Bool(inter interface{}, v *BoolValidation) (bool, error) {
	if inter == nil {
		if v.TreatNullAsFalse {
			return ValidateBoolProvided(false, v)
		}
		return false, ErrorCannotBeNull(v.Required)
	}
	casted, castOk := inter.(bool)
	if !castOk {
		return false, ErrorInvalidPrimitiveType(inter, PrimTypeBool)
	}
	return ValidateBoolProvided(casted, v)
}

func BoolFromInterfaceMap(key string, iMap map[string]interface{}, v *BoolValidation) (bool, error) {
	inter, ok := ReadInterfaceMapValue(key, iMap)
	if !ok {
		val, err := ValidateBoolMissing(v)
		if err != nil {
			return false, errors.Wrap(err, key)
		}
		return val, nil
	}
	val, err := Bool(inter, v)
	if err != nil {
		return false, errors.Wrap(err, key)
	}
	return val, nil
}

func BoolFromStrMap(key string, sMap map[string]string, v *BoolValidation) (bool, error) {
	valStr, ok := sMap[key]
	if !ok || valStr == "" {
		val, err := ValidateBoolMissing(v)
		if err != nil {
			return false, errors.Wrap(err, key)
		}
		return val, nil
	}
	val, err := BoolFromStr(valStr, v)
	if err != nil {
		return false, errors.Wrap(err, key)
	}
	return val, nil
}

func BoolFromStr(valStr string, v *BoolValidation) (bool, error) {
	if valStr == "" {
		return ValidateBoolMissing(v)
	}
	if len(v.StrToBool) > 0 {
		casted, ok := v.StrToBool[strings.ToLower(valStr)]

		if !ok {
			keys := make([]string, 0, len(v.StrToBool))
			for key := range v.StrToBool {
				keys = append(keys, key)
			}

			return false, ErrorInvalidStr(valStr, keys[0], keys[1:]...)
		}
		return ValidateBoolProvided(casted, v)
	}

	casted, castOk := s.ParseBool(valStr)
	if !castOk {
		return false, ErrorInvalidPrimitiveType(valStr, PrimTypeBool)
	}
	return ValidateBoolProvided(casted, v)
}

func BoolFromEnv(envVarName string, v *BoolValidation) (bool, error) {
	valStr := ReadEnvVar(envVarName)
	if valStr == nil || *valStr == "" {
		val, err := ValidateBoolMissing(v)
		if err != nil {
			return false, errors.Wrap(err, EnvVar(envVarName))
		}
		return val, nil
	}
	val, err := BoolFromStr(*valStr, v)
	if err != nil {
		return false, errors.Wrap(err, EnvVar(envVarName))
	}
	return val, nil
}

func BoolFromFile(filePath string, v *BoolValidation) (bool, error) {
	if !files.IsFile(filePath) {
		val, err := ValidateBoolMissing(v)
		if err != nil {
			return false, errors.Wrap(err, filePath)
		}
		return val, nil
	}

	valStr, err := files.ReadFile(filePath)
	if err != nil {
		return false, err
	}
	if len(valStr) == 0 {
		val, err := ValidateBoolMissing(v)
		if err != nil {
			return false, errors.Wrap(err, filePath)
		}
		return val, nil
	}

	val, err := BoolFromStr(valStr, v)
	if err != nil {
		return false, errors.Wrap(err, filePath)
	}
	return val, nil
}

func BoolFromEnvOrFile(envVarName string, filePath string, v *BoolValidation) (bool, error) {
	valStr := ReadEnvVar(envVarName)
	if valStr != nil && *valStr != "" {
		return BoolFromEnv(envVarName, v)
	}
	return BoolFromFile(filePath, v)
}

func BoolFromPrompt(promptOpts *prompt.Options, v *BoolValidation) (bool, error) {
	promptOpts.DefaultStr = s.Bool(v.Default)
	valStr := prompt.Prompt(promptOpts)
	if valStr == "" {
		return ValidateBoolMissing(v)
	}
	return BoolFromStr(valStr, v)
}

func ValidateBoolMissing(v *BoolValidation) (bool, error) {
	if v.Required {
		return false, ErrorMustBeDefined()
	}
	return validateBool(v.Default, v)
}

func ValidateBoolProvided(val bool, v *BoolValidation) (bool, error) {
	if v.CantBeSpecifiedErrStr != nil {
		return false, ErrorFieldCantBeSpecified(*v.CantBeSpecifiedErrStr)
	}
	return validateBool(val, v)
}

func validateBool(val bool, v *BoolValidation) (bool, error) {
	return val, nil
}

//
// Musts
//

func MustBoolFromEnv(envVarName string, v *BoolValidation) bool {
	val, err := BoolFromEnv(envVarName, v)
	if err != nil {
		exit.Panic(err)
	}
	return val
}

func MustBoolFromFile(filePath string, v *BoolValidation) bool {
	val, err := BoolFromFile(filePath, v)
	if err != nil {
		exit.Panic(err)
	}
	return val
}

func MustBoolFromEnvOrFile(envVarName string, filePath string, v *BoolValidation) bool {
	val, err := BoolFromEnvOrFile(envVarName, filePath, v)
	if err != nil {
		exit.Panic(err)
	}
	return val
}
