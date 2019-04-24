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

package userconfig

import (
	cr "github.com/cortexlabs/cortex/pkg/lib/configreader"
)

type Inputs struct {
	Columns map[string]interface{} `json:"columns" yaml:"columns"`
	Args    map[string]interface{} `json:"args" yaml:"args"`
}

var inputTypesFieldValidation = &cr.StructFieldValidation{
	StructField: "Inputs",
	StructValidation: &cr.StructValidation{
		Required: true,
		StructFieldValidations: []*cr.StructFieldValidation{
			{
				StructField: "Columns",
				InterfaceMapValidation: &cr.InterfaceMapValidation{
					AllowEmpty: true,
					Default:    make(map[string]interface{}),
					Validator: func(columnTypes map[string]interface{}) (map[string]interface{}, error) {
						return columnTypes, ValidateColumnInputTypes(columnTypes)
					},
				},
			},
			{
				StructField: "Args",
				InterfaceMapValidation: &cr.InterfaceMapValidation{
					AllowEmpty: true,
					Default:    make(map[string]interface{}),
					Validator: func(argTypes map[string]interface{}) (map[string]interface{}, error) {
						return argTypes, ValidateArgTypes(argTypes)
					},
				},
			},
		},
	},
}

var inputValuesFieldValidation = &cr.StructFieldValidation{
	StructField: "Inputs",
	StructValidation: &cr.StructValidation{
		Required: true,
		StructFieldValidations: []*cr.StructFieldValidation{
			{
				StructField: "Columns",
				InterfaceMapValidation: &cr.InterfaceMapValidation{
					AllowEmpty: true,
					Default:    make(map[string]interface{}),
					Validator: func(columnInputValues map[string]interface{}) (map[string]interface{}, error) {
						return columnInputValues, ValidateColumnInputValues(columnInputValues)
					},
				},
			},
			{
				StructField: "Args",
				InterfaceMapValidation: &cr.InterfaceMapValidation{
					AllowEmpty: true,
					Default:    make(map[string]interface{}),
					Validator: func(argValues map[string]interface{}) (map[string]interface{}, error) {
						return argValues, ValidateArgValues(argValues)
					},
				},
			},
		},
	},
}
