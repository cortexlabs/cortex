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

type Int32ListValidation struct {
	Required          bool
	Default           []int32
	AllowExplicitNull bool
	AllowEmpty        bool
	Validator         func([]int32) ([]int32, error)
}

func Int32List(inter interface{}, v *Int32ListValidation) ([]int32, error) {
	casted, castOk := cast.InterfaceToInt32Slice(inter)
	if !castOk {
		return nil, ErrorInvalidPrimitiveType(inter, PrimTypeIntList)
	}
	return ValidateInt32ListProvided(casted, v)
}

func Int32ListFromInterfaceMap(key string, iMap map[string]interface{}, v *Int32ListValidation) ([]int32, error) {
	inter, ok := ReadInterfaceMapValue(key, iMap)
	if !ok {
		val, err := ValidateInt32ListMissing(v)
		if err != nil {
			return nil, errors.Wrap(err, key)
		}
		return val, nil
	}
	val, err := Int32List(inter, v)
	if err != nil {
		return nil, errors.Wrap(err, key)
	}
	return val, nil
}

func ValidateInt32ListMissing(v *Int32ListValidation) ([]int32, error) {
	if v.Required {
		return nil, ErrorMustBeDefined()
	}
	return validateInt32List(v.Default, v)
}

func ValidateInt32ListProvided(val []int32, v *Int32ListValidation) ([]int32, error) {
	if !v.AllowExplicitNull && val == nil {
		return nil, ErrorCannotBeNull()
	}
	return validateInt32List(val, v)
}

func validateInt32List(val []int32, v *Int32ListValidation) ([]int32, error) {
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
