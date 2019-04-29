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

type Float32ListValidation struct {
	Required   bool
	Default    []float32
	AllowNull  bool
	AllowEmpty bool
	Validator  func([]float32) ([]float32, error)
}

func Float32List(inter interface{}, v *Float32ListValidation) ([]float32, error) {
	casted, castOk := cast.InterfaceToFloat32Slice(inter)
	if !castOk {
		return nil, ErrorInvalidPrimitiveType(inter, PrimTypeFloatList)
	}
	return ValidateFloat32List(casted, v)
}

func Float32ListFromInterfaceMap(key string, iMap map[string]interface{}, v *Float32ListValidation) ([]float32, error) {
	inter, ok := ReadInterfaceMapValue(key, iMap)
	if !ok {
		val, err := ValidateFloat32ListMissing(v)
		if err != nil {
			return nil, errors.Wrap(err, key)
		}
		return val, nil
	}
	val, err := Float32List(inter, v)
	if err != nil {
		return nil, errors.Wrap(err, key)
	}
	return val, nil
}

func ValidateFloat32ListMissing(v *Float32ListValidation) ([]float32, error) {
	if v.Required {
		return nil, ErrorMustBeDefined()
	}
	return ValidateFloat32List(v.Default, v)
}

func ValidateFloat32List(val []float32, v *Float32ListValidation) ([]float32, error) {
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
