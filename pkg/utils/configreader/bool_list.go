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
	"github.com/cortexlabs/cortex/pkg/utils/cast"
	"github.com/cortexlabs/cortex/pkg/utils/errors"
)

type BoolListValidation struct {
	Required   bool
	Default    []bool
	AllowNull  bool
	AllowEmpty bool
	Validator  func([]bool) ([]bool, error)
}

func BoolList(inter interface{}, v *BoolListValidation) ([]bool, error) {
	casted, castOk := cast.InterfaceToBoolSlice(inter)
	if !castOk {
		return nil, errors.New(s.ErrInvalidPrimitiveType(inter, s.PrimTypeBoolList))
	}
	return ValidateBoolList(casted, v)
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
		return nil, errors.New(s.ErrMustBeDefined)
	}
	return ValidateBoolList(v.Default, v)
}

func ValidateBoolList(val []bool, v *BoolListValidation) ([]bool, error) {
	if !v.AllowNull {
		if val == nil {
			return nil, errors.New(s.ErrCannotBeNull)
		}
	}

	if !v.AllowEmpty {
		if val != nil && len(val) == 0 {
			return nil, errors.New(s.ErrCannotBeEmpty)
		}
	}

	if v.Validator != nil {
		return v.Validator(val)
	}
	return val, nil
}
