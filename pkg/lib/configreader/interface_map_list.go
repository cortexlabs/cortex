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

type InterfaceMapListValidation struct {
	Required               bool
	Default                []map[string]interface{}
	AllowExplicitNull      bool
	AllowEmpty             bool
	CantBeSpecifiedErrStr  *string
	CastSingleItem         bool
	MinLength              int
	MaxLength              int
	InvalidLengths         []int
	AllowCortexResources   bool
	RequireCortexResources bool
	Validator              func([]map[string]interface{}) ([]map[string]interface{}, error)
}

func InterfaceMapList(inter interface{}, v *InterfaceMapListValidation) ([]map[string]interface{}, error) {
	casted, castOk := cast.InterfaceToStrInterfaceMapSlice(inter)
	if !castOk {
		if v.CastSingleItem {
			castedItem, castOk := cast.InterfaceToStrInterfaceMap(inter)
			if !castOk {
				return nil, ErrorInvalidPrimitiveType(inter, PrimTypeMap, PrimTypeMapList)
			}
			casted = []map[string]interface{}{castedItem}
		} else {
			return nil, ErrorInvalidPrimitiveType(inter, PrimTypeMapList)
		}
	}
	return ValidateInterfaceMapListProvided(casted, v)
}

func InterfaceMapListFromInterfaceMap(key string, iMap map[string]interface{}, v *InterfaceMapListValidation) ([]map[string]interface{}, error) {
	inter, ok := ReadInterfaceMapValue(key, iMap)
	if !ok {
		val, err := ValidateInterfaceMapListMissing(v)
		if err != nil {
			return nil, errors.Wrap(err, key)
		}
		return val, nil
	}
	val, err := InterfaceMapList(inter, v)
	if err != nil {
		return nil, errors.Wrap(err, key)
	}
	return val, nil
}

func ValidateInterfaceMapListMissing(v *InterfaceMapListValidation) ([]map[string]interface{}, error) {
	if v.Required {
		return nil, ErrorMustBeDefined()
	}
	return validateInterfaceMapList(v.Default, v)
}

func ValidateInterfaceMapListProvided(val []map[string]interface{}, v *InterfaceMapListValidation) ([]map[string]interface{}, error) {
	if v.CantBeSpecifiedErrStr != nil {
		return nil, ErrorFieldCantBeSpecified(*v.CantBeSpecifiedErrStr)
	}

	if !v.AllowExplicitNull && val == nil {
		return nil, ErrorCannotBeNull(v.Required)
	}
	return validateInterfaceMapList(val, v)
}

func validateInterfaceMapList(val []map[string]interface{}, v *InterfaceMapListValidation) ([]map[string]interface{}, error) {
	if v.RequireCortexResources {
		if err := checkOnlyCortexResources(val); err != nil {
			return nil, err
		}
	} else if !v.AllowCortexResources {
		if err := checkNoCortexResources(val); err != nil {
			return nil, err
		}
	}

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
