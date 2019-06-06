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
	"github.com/cortexlabs/cortex/pkg/lib/configreader"
	"github.com/cortexlabs/cortex/pkg/lib/errors"
	"github.com/cortexlabs/cortex/pkg/operator/api/resource"
)

type Aggregates []*Aggregate

type Aggregate struct {
	ResourceFields
	Aggregator     string        `json:"aggregator" yaml:"aggregator"`
	AggregatorPath *string       `json:"aggregator_path" yaml:"aggregator_path"`
	Input          interface{}   `json:"input" yaml:"input"`
	Compute        *SparkCompute `json:"compute" yaml:"compute"`
	Tags           Tags          `json:"tags" yaml:"tags"`
}

var aggregateValidation = &configreader.StructValidation{
	StructFieldValidations: []*configreader.StructFieldValidation{
		{
			StructField: "Name",
			StringValidation: &configreader.StringValidation{
				Required:                   true,
				AlphaNumericDashUnderscore: true,
			},
		},
		{
			StructField: "Aggregator",
			StringValidation: &configreader.StringValidation{
				AllowEmpty:                           true,
				AlphaNumericDashDotUnderscoreOrEmpty: true,
			},
		},
		{
			StructField:         "AggregatorPath",
			StringPtrValidation: &configreader.StringPtrValidation{},
		},
		{
			StructField: "Input",
			InterfaceValidation: &configreader.InterfaceValidation{
				Required:             true,
				AllowCortexResources: true,
			},
		},
		sparkComputeFieldValidation("Compute"),
		tagsFieldValidation,
		typeFieldValidation,
	},
}

func (aggregates Aggregates) Validate() error {
	for _, aggregate := range aggregates {
		if err := aggregate.Validate(); err != nil {
			return err
		}
	}

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

func (aggregate *Aggregate) Validate() error {
	if aggregate.AggregatorPath == nil && aggregate.Aggregator == "" {
		return errors.Wrap(ErrorSpecifyOnlyOneMissing("aggregator", "aggregator_path"), Identify(aggregate))
	}

	if aggregate.AggregatorPath != nil && aggregate.Aggregator != "" {
		return errors.Wrap(ErrorSpecifyOnlyOne("aggregator", "aggregator_path"), Identify(aggregate))
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
