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
	"github.com/cortexlabs/cortex/pkg/lib/cast"
	"github.com/cortexlabs/cortex/pkg/lib/errors"
)

type StringMapValidation struct {
	Required   bool
	Default    map[string]string
	AllowNull  bool
	AllowEmpty bool
	Validator  func(map[string]string) (map[string]string, error)
}

func StringMap(inter interface{}, v *StringMapValidation) (map[string]string, error) {
	casted, castOk := cast.InterfaceToStrStrMap(inter)
	if !castOk {
		return nil, ErrorInvalidPrimitiveType(inter, PrimTypeStringToStringMap)
	}
	return ValidateStringMap(casted, v)
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
	return ValidateStringMap(v.Default, v)
}

func ValidateStringMap(val map[string]string, v *StringMapValidation) (map[string]string, error) {
	if !v.AllowNull {
		if val == nil {
			return nil, ErrorCannotBeNull()
		}
	}

	if !v.AllowEmpty {
		if val != nil && len(val) == 0 {
			return nil, ErrorCannotBeEmpty()
		}
	}

	if v.Validator != nil {
		return v.Validator(val)
	}
	return val, nil
}
