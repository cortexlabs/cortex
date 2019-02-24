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

type IntListValidation struct {
	Required   bool
	Default    []int
	AllowNull  bool
	AllowEmpty bool
	Validator  func([]int) ([]int, error)
}

func IntList(inter interface{}, v *IntListValidation) ([]int, error) {
	casted, castOk := cast.InterfaceToIntSlice(inter)
	if !castOk {
		return nil, errors.New(s.ErrInvalidPrimitiveType(inter, s.PrimTypeIntList))
	}
	return ValidateIntList(casted, v)
}

func IntListFromInterfaceMap(key string, iMap map[string]interface{}, v *IntListValidation) ([]int, error) {
	inter, ok := ReadInterfaceMapValue(key, iMap)
	if !ok {
		val, err := ValidateIntListMissing(v)
		if err != nil {
			return nil, errors.Wrap(err, key)
		}
		return val, nil
	}
	val, err := IntList(inter, v)
	if err != nil {
		return nil, errors.Wrap(err, key)
	}
	return val, nil
}

func ValidateIntListMissing(v *IntListValidation) ([]int, error) {
	if v.Required {
		return nil, errors.New(s.ErrMustBeDefined)
	}
	return ValidateIntList(v.Default, v)
}

func ValidateIntList(val []int, v *IntListValidation) ([]int, error) {
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
