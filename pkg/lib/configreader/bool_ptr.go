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
	"github.com/cortexlabs/cortex/pkg/lib/files"
	"github.com/cortexlabs/cortex/pkg/lib/prompt"
	s "github.com/cortexlabs/cortex/pkg/lib/strings"
)

type BoolPtrValidation struct {
	Required              bool
	Default               *bool
	AllowExplicitNull     bool
	CantBeSpecifiedErrStr *string
	StrToBool             map[string]bool // lowercase
}

func BoolPtr(inter interface{}, v *BoolPtrValidation) (*bool, error) {
	if inter == nil {
		return ValidateBoolPtrProvided(nil, v)
	}
	casted, castOk := inter.(bool)
	if !castOk {
		return nil, ErrorInvalidPrimitiveType(inter, PrimTypeBool)
	}
	return ValidateBoolPtrProvided(&casted, v)
}

func BoolPtrFromInterfaceMap(key string, iMap map[string]interface{}, v *BoolPtrValidation) (*bool, error) {
	inter, ok := ReadInterfaceMapValue(key, iMap)
	if !ok {
		val, err := ValidateBoolPtrMissing(v)
		if err != nil {
			return nil, errors.Wrap(err, key)
		}
		return val, nil
	}
	val, err := BoolPtr(inter, v)
	if err != nil {
		return nil, errors.Wrap(err, key)
	}
	return val, nil
}

func BoolPtrFromStrMap(key string, sMap map[string]string, v *BoolPtrValidation) (*bool, error) {
	valStr, ok := sMap[key]
	if !ok || valStr == "" {
		val, err := ValidateBoolPtrMissing(v)
		if err != nil {
			return nil, errors.Wrap(err, key)
		}
		return val, nil
	}
	val, err := BoolPtrFromStr(valStr, v)
	if err != nil {
		return nil, errors.Wrap(err, key)
	}
	return val, nil
}

func BoolPtrFromStr(valStr string, v *BoolPtrValidation) (*bool, error) {
	if valStr == "" {
		return ValidateBoolPtrMissing(v)
	}

	if len(v.StrToBool) > 0 {
		casted, ok := v.StrToBool[strings.ToLower(valStr)]

		if !ok {
			keys := make([]string, 0, len(v.StrToBool))
			for key := range v.StrToBool {
				keys = append(keys, key)
			}

			return nil, ErrorInvalidStr(valStr, keys[0], keys[1:]...)
		}
		return ValidateBoolPtrProvided(&casted, v)
	}

	casted, castOk := s.ParseBool(valStr)
	if !castOk {
		return nil, ErrorInvalidPrimitiveType(valStr, PrimTypeBool)
	}
	return ValidateBoolPtrProvided(&casted, v)
}

func BoolPtrFromEnv(envVarName string, v *BoolPtrValidation) (*bool, error) {
	valStr := ReadEnvVar(envVarName)
	if valStr == nil || *valStr == "" {
		val, err := ValidateBoolPtrMissing(v)
		if err != nil {
			return nil, errors.Wrap(err, EnvVar(envVarName))
		}
		return val, nil
	}
	val, err := BoolPtrFromStr(*valStr, v)
	if err != nil {
		return nil, errors.Wrap(err, EnvVar(envVarName))
	}
	return val, nil
}

func BoolPtrFromFile(filePath string, v *BoolPtrValidation) (*bool, error) {
	if !files.IsFile(filePath) {
		val, err := ValidateBoolPtrMissing(v)
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
		val, err := ValidateBoolPtrMissing(v)
		if err != nil {
			return nil, errors.Wrap(err, filePath)
		}
		return val, nil
	}

	val, err := BoolPtrFromStr(valStr, v)
	if err != nil {
		return nil, errors.Wrap(err, filePath)
	}
	return val, nil
}

func BoolPtrFromEnvOrFile(envVarName string, filePath string, v *BoolPtrValidation) (*bool, error) {
	valStr := ReadEnvVar(envVarName)
	if valStr != nil && *valStr != "" {
		return BoolPtrFromEnv(envVarName, v)
	}
	return BoolPtrFromFile(filePath, v)
}

func BoolPtrFromPrompt(promptOpts *prompt.Options, v *BoolPtrValidation) (*bool, error) {
	if v.Default != nil && promptOpts.DefaultStr == "" {
		promptOpts.DefaultStr = s.Bool(*v.Default)
	}
	valStr := prompt.Prompt(promptOpts)
	if valStr == "" {
		return ValidateBoolPtrMissing(v)
	}
	return BoolPtrFromStr(valStr, v)
}

func ValidateBoolPtrMissing(v *BoolPtrValidation) (*bool, error) {
	if v.Required {
		return nil, ErrorMustBeDefined()
	}
	return v.Default, nil
}

func ValidateBoolPtrProvided(val *bool, v *BoolPtrValidation) (*bool, error) {
	if v.CantBeSpecifiedErrStr != nil {
		return nil, ErrorFieldCantBeSpecified(*v.CantBeSpecifiedErrStr)
	}

	if !v.AllowExplicitNull && val == nil {
		return nil, ErrorCannotBeNull(v.Required)
	}
	return val, nil
}
