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
	"github.com/cortexlabs/cortex/pkg/lib/interfaces"
	"github.com/cortexlabs/cortex/pkg/lib/slices"
	s "github.com/cortexlabs/cortex/pkg/operator/api/strings"
)

type InterfaceMapValidation struct {
	Required          bool
	AllowNull         bool
	AllowEmpty        bool
	ScalarsOnly       bool
	StringLeavesOnly  bool
	AllowedLeafValues []string
	Default           map[string]interface{}
	Validator         func(map[string]interface{}) (map[string]interface{}, error)
}

func InterfaceMap(inter interface{}, v *InterfaceMapValidation) (map[string]interface{}, error) {
	casted, castOk := cast.InterfaceToStrInterfaceMap(inter)
	if !castOk {
		return nil, ErrorInvalidPrimitiveType(inter, s.PrimTypeMap)
	}
	return ValidateInterfaceMap(casted, v)
}

func InterfaceMapFromInterfaceMap(key string, iMap map[string]interface{}, v *InterfaceMapValidation) (map[string]interface{}, error) {
	inter, ok := ReadInterfaceMapValue(key, iMap)
	if !ok {
		val, err := ValidateInterfaceMapMissing(v)
		if err != nil {
			return nil, errors.Wrap(err, key)
		}
		return val, nil
	}
	val, err := InterfaceMap(inter, v)
	if err != nil {
		return nil, errors.Wrap(err, key)
	}
	return val, nil
}

func ValidateInterfaceMapMissing(v *InterfaceMapValidation) (map[string]interface{}, error) {
	if v.Required {
		return nil, ErrorMustBeDefined()
	}
	return ValidateInterfaceMap(v.Default, v)
}

func ValidateInterfaceMap(val map[string]interface{}, v *InterfaceMapValidation) (map[string]interface{}, error) {
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

	if v.ScalarsOnly {
		for k, v := range val {
			if !cast.IsScalarType(v) {
				return nil, errors.Wrap(ErrorInvalidPrimitiveType(v, s.PrimTypeString, s.PrimTypeInt, s.PrimTypeFloat, s.PrimTypeBool), k)
			}
		}
	}

	if v.StringLeavesOnly {
		_, err := interfaces.FlattenAllStrValues(val)
		if err != nil {
			return nil, err
		}
	}

	if v.AllowedLeafValues != nil {
		leafVals, err := interfaces.FlattenAllStrValues(val)
		if err != nil {
			return nil, err
		}
		for _, leafVal := range leafVals {
			if !slices.HasString(v.AllowedLeafValues, leafVal) {
				return nil, ErrorInvalidStr(leafVal, v.AllowedLeafValues...)
			}
		}
	}

	if v.Validator != nil {
		return v.Validator(val)
	}
	return val, nil
}
