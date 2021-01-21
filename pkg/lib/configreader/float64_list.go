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

type Float64ListValidation struct {
	Required              bool
	Default               []float64
	AllowExplicitNull     bool
	AllowEmpty            bool
	CantBeSpecifiedErrStr *string
	CastSingleItem        bool
	MinLength             int
	MaxLength             int
	InvalidLengths        []int
	Validator             func([]float64) ([]float64, error)
}

func Float64List(inter interface{}, v *Float64ListValidation) ([]float64, error) {
	casted, castOk := cast.InterfaceToFloat64Slice(inter)
	if !castOk {
		if v.CastSingleItem {
			castedItem, castOk := cast.InterfaceToFloat64(inter)
			if !castOk {
				return nil, ErrorInvalidPrimitiveType(inter, PrimTypeFloat, PrimTypeFloatList)
			}
			casted = []float64{castedItem}
		} else {
			return nil, ErrorInvalidPrimitiveType(inter, PrimTypeFloatList)
		}
	}
	return ValidateFloat64ListProvided(casted, v)
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
	return validateFloat64List(v.Default, v)
}

func ValidateFloat64ListProvided(val []float64, v *Float64ListValidation) ([]float64, error) {
	if v.CantBeSpecifiedErrStr != nil {
		return nil, ErrorFieldCantBeSpecified(*v.CantBeSpecifiedErrStr)
	}

	if !v.AllowExplicitNull && val == nil {
		return nil, ErrorCannotBeNull(v.Required)
	}
	return validateFloat64List(val, v)
}

func validateFloat64List(val []float64, v *Float64ListValidation) ([]float64, error) {
	if !v.AllowEmpty {
		if val != nil && len(val) == 0 {
			return nil, ErrorCannotBeEmpty()
		}
	}

	if v.MinLength != 0 {
		if len(val) < v.MinLength {
			return nil, ErrorTooFewElements(v.MinLength)
		}
	}

	if v.MaxLength != 0 {
		if len(val) > v.MaxLength {
			return nil, ErrorTooManyElements(v.MaxLength)
		}
	}

	for _, invalidLength := range v.InvalidLengths {
		if len(val) == invalidLength {
			return nil, ErrorWrongNumberOfElements(v.InvalidLengths)
		}
	}

	if v.Validator != nil {
		return v.Validator(val)
	}
	return val, nil
}
