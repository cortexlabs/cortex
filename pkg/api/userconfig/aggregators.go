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
	"github.com/cortexlabs/cortex/pkg/api/resource"
	cr "github.com/cortexlabs/cortex/pkg/utils/configreader"
)

type Aggregators []*Aggregator

type Aggregator struct {
	Name       string      `json:"name"  yaml:"name"`
	Inputs     *Inputs     `json:"inputs"  yaml:"inputs"`
	OutputType interface{} `json:"output_type"  yaml:"output_type"`
	Path       string      `json:"path"  yaml:"path"`
	FilePath   string      `json:"file_path" yaml:"-"`
	Embed      *Embed      `json:"embed"  yaml:"-"`
}

var aggregatorValidation = &cr.StructValidation{
	StructFieldValidations: []*cr.StructFieldValidation{
		&cr.StructFieldValidation{
			StructField: "Name",
			StringValidation: &cr.StringValidation{
				Required:                   true,
				AlphaNumericDashUnderscore: true,
			},
		},
		&cr.StructFieldValidation{
			StructField:      "Path",
			StringValidation: &cr.StringValidation{},
			DefaultField:     "Name",
			DefaultFieldFunc: func(name interface{}) interface{} {
				return "implementations/aggregators/" + name.(string) + ".py"
			},
		},
		&cr.StructFieldValidation{
			StructField: "OutputType",
			InterfaceValidation: &cr.InterfaceValidation{
				Required: true,
				Validator: func(outputType interface{}) (interface{}, error) {
					return outputType, ValidateValueType(outputType)
				},
			},
		},
		inputTypesFieldValidation,
		typeFieldValidation,
	},
}

func (aggregators Aggregators) Validate() error {
	resources := make([]Resource, len(aggregators))
	for i, res := range aggregators {
		resources[i] = res
	}

	dups := FindDuplicateResourceName(resources...)
	if len(dups) > 0 {
		return ErrorDuplicateResourceName(dups...)
	}
	return nil
}

func (aggregators Aggregators) Get(name string) *Aggregator {
	for _, aggregator := range aggregators {
		if aggregator.Name == name {
			return aggregator
		}
	}
	return nil
}

func (aggregator *Aggregator) GetName() string {
	return aggregator.Name
}

func (aggregator *Aggregator) GetResourceType() resource.Type {
	return resource.AggregatorType
}

func (aggregator *Aggregator) GetFilePath() string {
	return aggregator.FilePath
}

func (aggregator *Aggregator) GetEmbed() *Embed {
	return aggregator.Embed
}

func (aggregators Aggregators) Names() []string {
	names := make([]string, len(aggregators))
	for i, aggregator := range aggregators {
		names[i] = aggregator.Name
	}
	return names
}
