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
)

type StringMapValidation struct {
	Required              bool
	Default               map[string]string
	AllowExplicitNull     bool
	AllowEmpty            bool
	ConvertNullToEmpty    bool
	CantBeSpecifiedErrStr *string
	KeyStringValidator    *StringValidation
	ValueStringValidator  *StringValidation
	Validator             func(map[string]string) (map[string]string, error)
}

func StringMap(inter interface{}, v *StringMapValidation) (map[string]string, error) {
	casted, castOk := cast.InterfaceToStrStrMap(inter)
	if !castOk {
		return nil, ErrorInvalidPrimitiveType(inter, PrimTypeStringToStringMap)
	}
	return ValidateStringMapProvided(casted, v)
}

func StringMapFromInterfaceMap(key string, iMap map[string]interface{}, v *StringMapValidation) (map[string]string, error) {
	inter, ok := ReadInterfaceMapValue(key, iMap)
	if !ok {
		val, err := ValidateStringMapMissing(v)
		if err != nil {
			return nil, errors.Wrap(err, key)
		}
		return val, nil
	}
	val, err := StringMap(inter, v)
	if err != nil {
		return nil, errors.Wrap(err, key)
	}
	return val, nil
}

func ValidateStringMapMissing(v *StringMapValidation) (map[string]string, error) {
	if v.Required {
		return nil, ErrorMustBeDefined()
	}
	return validateStringMap(v.Default, v)
}

func ValidateStringMapProvided(val map[string]string, v *StringMapValidation) (map[string]string, error) {
	if v.CantBeSpecifiedErrStr != nil {
		return nil, ErrorFieldCantBeSpecified(*v.CantBeSpecifiedErrStr)
	}

	if !v.AllowExplicitNull && val == nil {
		return nil, ErrorCannotBeNull(v.Required)
	}
	return validateStringMap(val, v)
}

func validateStringMap(val map[string]string, v *StringMapValidation) (map[string]string, error) {
	if !v.AllowEmpty {
		if val != nil && len(val) == 0 {
			return nil, ErrorCannotBeEmpty()
		}
	}

	if v.KeyStringValidator != nil {
		for mapKey := range val {
			err := ValidateStringVal(mapKey, v.KeyStringValidator)
			if err != nil {
				return nil, err
			}
		}
	}

	if v.ValueStringValidator != nil {
		for mapKey, mapVal := range val {
			err := ValidateStringVal(mapVal, v.ValueStringValidator)
			if err != nil {
				return nil, errors.Wrap(err, mapKey)
			}
		}
	}

	if v.Validator != nil {
		return v.Validator(val)
	}

	if val == nil && v.ConvertNullToEmpty {
		val = make(map[string]string)
	}

	return val, nil
}
