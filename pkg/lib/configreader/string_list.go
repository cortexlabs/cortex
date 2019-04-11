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
	"github.com/cortexlabs/cortex/pkg/lib/slices"
)

type StringListValidation struct {
	Required     bool
	Default      []string
	AllowNull    bool
	AllowEmpty   bool
	DisallowDups bool
	Validator    func([]string) ([]string, error)
}

func StringList(inter interface{}, v *StringListValidation) ([]string, error) {
	casted, castOk := cast.InterfaceToStrSlice(inter)
	if !castOk {
		return nil, ErrorInvalidPrimitiveType(inter, s.PrimTypeStringList)
	}
	return ValidateStringList(casted, v)
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
	return ValidateStringList(v.Default, v)
}

func ValidateStringList(val []string, v *StringListValidation) ([]string, error) {
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

	if v.DisallowDups {
		if dups := slices.FindDuplicateStrs(val); len(dups) > 0 {
			return nil, ErrorDuplicatedValue(dups[0])
		}
	}

	if v.Validator != nil {
		return v.Validator(val)
	}
	return val, nil
}
