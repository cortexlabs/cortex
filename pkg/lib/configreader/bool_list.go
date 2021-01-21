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

type BoolListValidation struct {
	Required              bool
	Default               []bool
	AllowExplicitNull     bool
	AllowEmpty            bool
	CantBeSpecifiedErrStr *string
	CastSingleItem        bool
	MinLength             int
	MaxLength             int
	InvalidLengths        []int
	Validator             func([]bool) ([]bool, error)
}

func BoolList(inter interface{}, v *BoolListValidation) ([]bool, error) {
	casted, castOk := cast.InterfaceToBoolSlice(inter)
	if !castOk {
		if v.CastSingleItem {
			castedItem, castOk := inter.(bool)
			if !castOk {
				return nil, ErrorInvalidPrimitiveType(inter, PrimTypeBool, PrimTypeBoolList)
			}
			casted = []bool{castedItem}
		} else {
			return nil, ErrorInvalidPrimitiveType(inter, PrimTypeBoolList)
		}
	}
	return ValidateBoolListProvided(casted, v)
}

func BoolListFromInterfaceMap(key string, iMap map[string]interface{}, v *BoolListValidation) ([]bool, error) {
	inter, ok := ReadInterfaceMapValue(key, iMap)
	if !ok {
		val, err := ValidateBoolListMissing(v)
		if err != nil {
			return nil, errors.Wrap(err, key)
		}
		return val, nil
	}
	val, err := BoolList(inter, v)
	if err != nil {
		return nil, errors.Wrap(err, key)
	}
	return val, nil
}

func ValidateBoolListMissing(v *BoolListValidation) ([]bool, error) {
	if v.Required {
		return nil, ErrorMustBeDefined()
	}
	return validateBoolList(v.Default, v)
}

func ValidateBoolListProvided(val []bool, v *BoolListValidation) ([]bool, error) {
	if v.CantBeSpecifiedErrStr != nil {
		return nil, ErrorFieldCantBeSpecified(*v.CantBeSpecifiedErrStr)
	}

	if !v.AllowExplicitNull && val == nil {
		return nil, ErrorCannotBeNull(v.Required)
	}
	return validateBoolList(val, v)
}

func validateBoolList(val []bool, v *BoolListValidation) ([]bool, error) {
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
