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

	"github.com/cortexlabs/cortex/pkg/lib/configreader"
	"github.com/cortexlabs/cortex/pkg/operator/api/resource"
)

type TransformedColumns []*TransformedColumn

type TransformedColumn struct {
	ResourceFields
	Transformer     string        `json:"transformer" yaml:"transformer"`
	TransformerPath *string       `json:"transformer_path" yaml:"transformer_path"`
	Inputs          *Inputs       `json:"inputs" yaml:"inputs"`
	Compute         *SparkCompute `json:"compute" yaml:"compute"`
	Tags            Tags          `json:"tags" yaml:"tags"`
}

var transformedColumnValidation = &configreader.StructValidation{
	StructFieldValidations: []*configreader.StructFieldValidation{
		{
			StructField: "Name",
			StringValidation: &configreader.StringValidation{
				Required:                   true,
				AlphaNumericDashUnderscore: true,
			},
		},
		{
			StructField: "Transformer",
			StringValidation: &configreader.StringValidation{
				AllowEmpty:                           true,
				AlphaNumericDashDotUnderscoreOrEmpty: true,
			},
		},
		{
			StructField:         "TransformerPath",
			StringPtrValidation: &configreader.StringPtrValidation{},
		},
		inputValuesFieldValidation,
		sparkComputeFieldValidation("Compute"),
		tagsFieldValidation,
		typeFieldValidation,
	},
}

func (columns TransformedColumns) Validate() error {
	resources := make([]Resource, len(columns))
	for i, res := range columns {
		resources[i] = res
	}

	dups := FindDuplicateResourceName(resources...)
	if len(dups) > 0 {
		return ErrorDuplicateResourceName(dups...)
	}

	return nil
}

func (column *TransformedColumn) IsRaw() bool {
	return false
}

func (column *TransformedColumn) GetResourceType() resource.Type {
	return resource.TransformedColumnType
}

func (columns TransformedColumns) Names() []string {
	names := make([]string, len(columns))
	for i, transformedColumn := range columns {
		names[i] = transformedColumn.GetName()
	}
	return names
}

func (columns TransformedColumns) Get(name string) *TransformedColumn {
	for _, transformedColumn := range columns {
		if transformedColumn.GetName() == name {
			return transformedColumn
		}
	}
	return nil
}

func (column *TransformedColumn) InputColumnNames() []string {
	inputs, _ := configreader.FlattenAllStrValues(column.Inputs.Columns)
	sort.Strings(inputs)
	return inputs
}
