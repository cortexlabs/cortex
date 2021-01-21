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
	"github.com/cortexlabs/cortex/pkg/lib/slices"
)

type StringListValidation struct {
	Required               bool
	Default                []string
	AllowExplicitNull      bool
	AllowEmpty             bool
	CantBeSpecifiedErrStr  *string
	CastSingleItem         bool
	DisallowDups           bool
	MinLength              int
	MaxLength              int
	InvalidLengths         []int
	AllowCortexResources   bool
	RequireCortexResources bool
	Validator              func([]string) ([]string, error)
}

func StringList(inter interface{}, v *StringListValidation) ([]string, error) {
	casted, castOk := cast.InterfaceToStrSlice(inter)
	if !castOk {
		if v.CastSingleItem {
			castedItem, castOk := inter.(string)
			if !castOk {
				return nil, ErrorInvalidPrimitiveType(inter, PrimTypeString, PrimTypeStringList)
			}
			casted = []string{castedItem}
		} else {
			return nil, ErrorInvalidPrimitiveType(inter, PrimTypeStringList)
		}
	}
	return ValidateStringListProvided(casted, v)
}

func StringListFromInterfaceMap(key string, iMap map[string]interface{}, v *StringListValidation) ([]string, error) {
	inter, ok := ReadInterfaceMapValue(key, iMap)
	if !ok {
		val, err := ValidateStringListMissing(v)
		if err != nil {
			return nil, errors.Wrap(err, key)
		}
		return val, nil
	}
	val, err := StringList(inter, v)
	if err != nil {
		return nil, errors.Wrap(err, key)
	}
	return val, nil
}

func ValidateStringListMissing(v *StringListValidation) ([]string, error) {
	if v.Required {
		return nil, ErrorMustBeDefined()
	}
	return validateStringList(v.Default, v)
}

func ValidateStringListProvided(val []string, v *StringListValidation) ([]string, error) {
	if v.CantBeSpecifiedErrStr != nil {
		return nil, ErrorFieldCantBeSpecified(*v.CantBeSpecifiedErrStr)
	}

	if !v.AllowExplicitNull && val == nil {
		return nil, ErrorCannotBeNull(v.Required)
	}
	return validateStringList(val, v)
}

func validateStringList(val []string, v *StringListValidation) ([]string, error) {
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

	if v.DisallowDups {
		if dups := slices.FindDuplicateStrs(val); len(dups) > 0 {
			return nil, ErrorDuplicatedValue(dups[0])
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
