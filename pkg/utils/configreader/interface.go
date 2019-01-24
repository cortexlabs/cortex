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
	"github.com/cortexlabs/cortex/pkg/utils/errors"
)

type InterfaceValidation struct {
	Required  bool
	Default   interface{}
	AllowNull bool
	Validator func(interface{}) (interface{}, error)
}

func Interface(inter interface{}, v *InterfaceValidation) (interface{}, error) {
	return ValidateInterface(inter, v)
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
	val, err := ValidateInterface(inter, v)
	if err != nil {
		return nil, errors.Wrap(err, key)
	}
	return val, nil
}

func ValidateInterfaceMissing(v *InterfaceValidation) (interface{}, error) {
	if v.Required {
		return nil, errors.New(s.ErrMustBeDefined)
	}
	return ValidateInterface(v.Default, v)
}

func ValidateInterface(val interface{}, v *InterfaceValidation) (interface{}, error) {
	if !v.AllowNull {
		if val == nil {
			return nil, errors.New(s.ErrCannotBeNull)
		}
	}

	if v.Validator != nil {
		return v.Validator(val)
	}
	return val, nil
}
