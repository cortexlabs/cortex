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
	"sort"

	"github.com/cortexlabs/cortex/pkg/api/resource"
	cr "github.com/cortexlabs/cortex/pkg/lib/configreader"
	"github.com/cortexlabs/cortex/pkg/lib/interfaces"
)

type Aggregates []*Aggregate

type Aggregate struct {
	ResourceConfigFields
	Aggregator string        `json:"aggregator" yaml:"aggregator"`
	Inputs     *Inputs       `json:"inputs" yaml:"inputs"`
	Compute    *SparkCompute `json:"compute" yaml:"compute"`
	Tags       Tags          `json:"tags" yaml:"tags"`
}

var aggregateValidation = &cr.StructValidation{
	StructFieldValidations: []*cr.StructFieldValidation{
		{
			StructField: "Name",
			StringValidation: &cr.StringValidation{
				Required:                   true,
				AlphaNumericDashUnderscore: true,
			},
		},
		{
			StructField: "Aggregator",
			StringValidation: &cr.StringValidation{
				Required:                      true,
				AlphaNumericDashDotUnderscore: true,
			},
		},
		inputValuesFieldValidation,
		sparkComputeFieldValidation,
		tagsFieldValidation,
		typeFieldValidation,
	},
}

func (aggregates Aggregates) Validate() error {
	resources := make([]Resource, len(aggregates))
	for i, res := range aggregates {
		resources[i] = res
	}

	dups := FindDuplicateResourceName(resources...)
	if len(dups) > 0 {
		return ErrorDuplicateResourceName(dups...)
	}
	return nil
}

func (aggregate *Aggregate) GetResourceType() resource.Type {
	return resource.AggregateType
}

func (aggregates Aggregates) Names() []string {
	names := make([]string, len(aggregates))
	for i, aggregate := range aggregates {
		names[i] = aggregate.GetName()
	}
	return names
}

func (aggregates Aggregates) Get(name string) *Aggregate {
	for _, aggregate := range aggregates {
		if aggregate.GetName() == name {
			return aggregate
		}
	}
	return nil
}

func (aggregate *Aggregate) InputColumnNames() []string {
	inputs, _ := interfaces.FlattenAllStrValues(aggregate.Inputs.Columns)
	sort.Strings(inputs)
	return inputs
}
