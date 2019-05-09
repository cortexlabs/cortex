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
	"github.com/cortexlabs/cortex/pkg/lib/maps"
	"github.com/cortexlabs/cortex/pkg/lib/pointer"
	"github.com/cortexlabs/cortex/pkg/lib/sets/strset"
	s "github.com/cortexlabs/cortex/pkg/lib/strings"
)

type InterfaceValidation struct {
	Required          bool
	Default           interface{}
	AllowExplicitNull bool
	Validator         func(interface{}) (interface{}, error)
}

func Interface(inter interface{}, v *InterfaceValidation) (interface{}, error) {
	return ValidateInterfaceProvided(inter, v)
}

func InterfaceFromInterfaceMap(key string, iMap map[string]interface{}, v *InterfaceValidation) (interface{}, error) {
	inter, ok := ReadInterfaceMapValue(key, iMap)
	if !ok {
		val, err := ValidateInterfaceMissing(v)
		if err != nil {
			return nil, errors.Wrap(err, key)
		}
		return val, nil
	}
	val, err := ValidateInterfaceProvided(inter, v)
	if err != nil {
		return nil, errors.Wrap(err, key)
	}
	return val, nil
}

func ValidateInterfaceMissing(v *InterfaceValidation) (interface{}, error) {
	if v.Required {
		return nil, ErrorMustBeDefined()
	}
	return validateInterface(v.Default, v)
}

func ValidateInterfaceProvided(val interface{}, v *InterfaceValidation) (interface{}, error) {
	if !v.AllowExplicitNull && val == nil {
		return nil, ErrorCannotBeNull()
	}
	return validateInterface(val, v)
}

func validateInterface(val interface{}, v *InterfaceValidation) (interface{}, error) {
	if v.Validator != nil {
		return v.Validator(val)
	}
	return val, nil
}

// FlattenAllStrValues assumes that the order for maps is deterministic
func FlattenAllStrValues(obj interface{}) ([]string, error) {
	obj = pointer.IndirectSafe(obj)
	flattened := []string{}

	if objStr, ok := obj.(string); ok {
		return append(flattened, objStr), nil
	}

	if objSlice, ok := cast.InterfaceToInterfaceSlice(obj); ok {
		for i, elem := range objSlice {
			subFlattened, err := FlattenAllStrValues(elem)
			if err != nil {
				return nil, errors.Wrap(err, s.Index(i))
			}
			flattened = append(flattened, subFlattened...)
		}
		return flattened, nil
	}

	if objMap, ok := cast.InterfaceToStrInterfaceMap(obj); ok {
		for _, key := range maps.InterfaceMapSortedKeys(objMap) {
			subFlattened, err := FlattenAllStrValues(objMap[key])
			if err != nil {
				return nil, errors.Wrap(err, s.UserStrStripped(key))
			}
			flattened = append(flattened, subFlattened...)
		}
		return flattened, nil
	}

	return nil, ErrorInvalidPrimitiveType(obj, PrimTypeString, PrimTypeList, PrimTypeMap)
}

func FlattenAllStrValuesAsSet(obj interface{}) (strset.Set, error) {
	strs, err := FlattenAllStrValues(obj)
	if err != nil {
		return nil, err
	}

	set := strset.New()
	for _, str := range strs {
		set.Add(str)
	}
	return set, nil
}
