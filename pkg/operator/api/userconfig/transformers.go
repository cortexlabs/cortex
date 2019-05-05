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
	"github.com/cortexlabs/cortex/pkg/operator/api/resource"
)

type Transformers []*Transformer

type Transformer struct {
	ResourceFields
	Inputs     *Inputs    `json:"inputs"  yaml:"inputs"`
	OutputType ColumnType `json:"output_type"  yaml:"output_type"`
	Path       string     `json:"path"  yaml:"path"`
}

var transformerValidation = &cr.StructValidation{
	StructFieldValidations: []*cr.StructFieldValidation{
		{
			StructField: "Name",
			StringValidation: &cr.StringValidation{
				Required:                   true,
				AlphaNumericDashUnderscore: true,
			},
		},
		{
			StructField:      "Path",
			StringValidation: &cr.StringValidation{},
			DefaultField:     "Name",
			DefaultFieldFunc: func(name interface{}) interface{} {
				return "implementations/transformers/" + name.(string) + ".py"
			},
		},
		{
			StructField: "OutputType",
			StringValidation: &cr.StringValidation{
				Required:      true,
				AllowedValues: ColumnTypeStrings(),
			},
			Parser: func(str string) (interface{}, error) {
				return ColumnTypeFromString(str), nil
			},
		},
		inputTypesFieldValidation,
		typeFieldValidation,
	},
}

func (transformers Transformers) Validate() error {
	resources := make([]Resource, len(transformers))
	for i, res := range transformers {
		resources[i] = res
	}

	dups := FindDuplicateResourceName(resources...)
	if len(dups) > 0 {
		return ErrorDuplicateResourceName(dups...)
	}

	return nil
}

func (transformers Transformers) Get(name string) *Transformer {
	for _, transformer := range transformers {
		if transformer.Name == name {
			return transformer
		}
	}
	return nil
}

func (transformer *Transformer) GetResourceType() resource.Type {
	return resource.TransformerType
}

func (transformers Transformers) Names() []string {
	names := make([]string, len(transformers))
	for i, transformer := range transformers {
		names[i] = transformer.Name
	}
	return names
}
