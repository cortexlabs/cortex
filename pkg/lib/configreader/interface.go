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
	"github.com/cortexlabs/cortex/pkg/lib/maps"
	"github.com/cortexlabs/cortex/pkg/lib/pointer"
	"github.com/cortexlabs/cortex/pkg/lib/sets/strset"
	s "github.com/cortexlabs/cortex/pkg/lib/strings"
	"github.com/cortexlabs/yaml"
)

type InterfaceValidation struct {
	Required               bool
	Default                interface{}
	AllowExplicitNull      bool
	CantBeSpecifiedErrStr  *string
	AllowCortexResources   bool
	RequireCortexResources bool
	Validator              func(interface{}) (interface{}, error)
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
	if v.CantBeSpecifiedErrStr != nil {
		return nil, ErrorFieldCantBeSpecified(*v.CantBeSpecifiedErrStr)
	}

	if !v.AllowExplicitNull && val == nil {
		return nil, ErrorCannotBeNull(v.Required)
	}
	return validateInterface(val, v)
}

func validateInterface(val interface{}, v *InterfaceValidation) (interface{}, error) {
	if v.RequireCortexResources {
		if err := checkOnlyCortexResources(val); err != nil {
			return nil, err
		}
	} else if !v.AllowCortexResources {
		if err := checkNoCortexResources(val); err != nil {
			return nil, err
		}
	}

	if v.Validator != nil {
		return v.Validator(val)
	}
	return val, nil
}

func checkNoCortexResources(obj interface{}) error {
	if objStr, ok := obj.(string); ok {
		if resourceName, ok := yaml.ExtractAtSymbolText(objStr); ok {
			return ErrorCortexResourceNotAllowed(resourceName)
		}
	}

	if objSlice, ok := cast.InterfaceToInterfaceSlice(obj); ok {
		for i, objItem := range objSlice {
			if err := checkNoCortexResources(objItem); err != nil {
				return errors.Wrap(err, s.Index(i))
			}
		}
	}

	if objMap, ok := cast.InterfaceToInterfaceInterfaceMap(obj); ok {
		for k, v := range objMap {
			if err := checkNoCortexResources(k); err != nil {
				return err
			}
			if err := checkNoCortexResources(v); err != nil {
				return errors.Wrap(err, s.UserStrStripped(k))
			}
		}
	}

	return nil
}

func checkOnlyCortexResources(obj interface{}) error {
	if objStr, ok := obj.(string); ok {
		if _, ok := yaml.ExtractAtSymbolText(objStr); !ok {
			return ErrorCortexResourceOnlyAllowed(objStr)
		}
	}

	if objSlice, ok := cast.InterfaceToInterfaceSlice(obj); ok {
		for i, objItem := range objSlice {
			if err := checkOnlyCortexResources(objItem); err != nil {
				return errors.Wrap(err, s.Index(i))
			}
		}
	}

	if objMap, ok := cast.InterfaceToInterfaceInterfaceMap(obj); ok {
		for k, v := range objMap {
			if err := checkOnlyCortexResources(k); err != nil {
				return err
			}
			if err := checkOnlyCortexResources(v); err != nil {
				return errors.Wrap(err, s.UserStrStripped(k))
			}
		}
	}

	return nil
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
