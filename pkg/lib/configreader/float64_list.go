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
	s "github.com/cortexlabs/cortex/pkg/operator/api/strings"
	"github.com/cortexlabs/cortex/pkg/lib/cast"
	"github.com/cortexlabs/cortex/pkg/lib/errors"
)

type Float64ListValidation struct {
	Required   bool
	Default    []float64
	AllowNull  bool
	AllowEmpty bool
	Validator  func([]float64) ([]float64, error)
}

func Float64List(inter interface{}, v *Float64ListValidation) ([]float64, error) {
	casted, castOk := cast.InterfaceToFloat64Slice(inter)
	if !castOk {
		return nil, ErrorInvalidPrimitiveType(inter, s.PrimTypeFloatList)
	}
	return ValidateFloat64List(casted, v)
}

func Float64ListFromInterfaceMap(key string, iMap map[string]interface{}, v *Float64ListValidation) ([]float64, error) {
	inter, ok := ReadInterfaceMapValue(key, iMap)
	if !ok {
		val, err := ValidateFloat64ListMissing(v)
		if err != nil {
			return nil, errors.Wrap(err, key)
		}
		return val, nil
	}
	val, err := Float64List(inter, v)
	if err != nil {
		return nil, errors.Wrap(err, key)
	}
	return val, nil
}

func ValidateFloat64ListMissing(v *Float64ListValidation) ([]float64, error) {
	if v.Required {
		return nil, ErrorMustBeDefined()
	}
	return ValidateFloat64List(v.Default, v)
}

func ValidateFloat64List(val []float64, v *Float64ListValidation) ([]float64, error) {
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
