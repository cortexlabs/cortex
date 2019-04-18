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
	s "github.com/cortexlabs/cortex/pkg/api/strings"
	"github.com/cortexlabs/cortex/pkg/lib/cast"
	"github.com/cortexlabs/cortex/pkg/lib/errors"
)

type InterfaceMapListValidation struct {
	Required   bool
	Default    []map[string]interface{}
	AllowNull  bool
	AllowEmpty bool
	Validator  func([]map[string]interface{}) ([]map[string]interface{}, error)
}

func InterfaceMapList(inter interface{}, v *InterfaceMapListValidation) ([]map[string]interface{}, error) {
	casted, castOk := cast.InterfaceToStrInterfaceMapSlice(inter)
	if !castOk {
		return nil, ErrorInvalidPrimitiveType(inter, s.PrimTypeMapList)
	}
	return ValidateInterfaceMapList(casted, v)
}

func InterfaceMapListFromInterfaceMap(key string, iMap map[string]interface{}, v *InterfaceMapListValidation) ([]map[string]interface{}, error) {
	inter, ok := ReadInterfaceMapValue(key, iMap)
	if !ok {
		val, err := ValidateInterfaceMapListMissing(v)
		if err != nil {
			return nil, errors.Wrap(err, key)
		}
		return val, nil
	}
	val, err := InterfaceMapList(inter, v)
	if err != nil {
		return nil, errors.Wrap(err, key)
	}
	return val, nil
}

func ValidateInterfaceMapListMissing(v *InterfaceMapListValidation) ([]map[string]interface{}, error) {
	if v.Required {
		return nil, ErrorMustBeDefined()
	}
	return ValidateInterfaceMapList(v.Default, v)
}

func ValidateInterfaceMapList(val []map[string]interface{}, v *InterfaceMapListValidation) ([]map[string]interface{}, error) {
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
