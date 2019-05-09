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

type Int64ListValidation struct {
	Required          bool
	Default           []int64
	AllowExplicitNull bool
	AllowEmpty        bool
	Validator         func([]int64) ([]int64, error)
}

func Int64List(inter interface{}, v *Int64ListValidation) ([]int64, error) {
	casted, castOk := cast.InterfaceToInt64Slice(inter)
	if !castOk {
		return nil, ErrorInvalidPrimitiveType(inter, PrimTypeIntList)
	}
	return ValidateInt64ListProvided(casted, v)
}

func Int64ListFromInterfaceMap(key string, iMap map[string]interface{}, v *Int64ListValidation) ([]int64, error) {
	inter, ok := ReadInterfaceMapValue(key, iMap)
	if !ok {
		val, err := ValidateInt64ListMissing(v)
		if err != nil {
			return nil, errors.Wrap(err, key)
		}
		return val, nil
	}
	val, err := Int64List(inter, v)
	if err != nil {
		return nil, errors.Wrap(err, key)
	}
	return val, nil
}

func ValidateInt64ListMissing(v *Int64ListValidation) ([]int64, error) {
	if v.Required {
		return nil, ErrorMustBeDefined()
	}
	return validateInt64List(v.Default, v)
}

func ValidateInt64ListProvided(val []int64, v *Int64ListValidation) ([]int64, error) {
	if !v.AllowExplicitNull && val == nil {
		return nil, ErrorCannotBeNull()
	}
	return validateInt64List(val, v)
}

func validateInt64List(val []int64, v *Int64ListValidation) ([]int64, error) {
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
