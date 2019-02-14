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
	"github.com/cortexlabs/cortex/pkg/utils/util"
)

type TransformedColumns []*TransformedColumn

type TransformedColumn struct {
	Name        string        `json:"name" yaml:"name"`
	Transformer string        `json:"transformer" yaml:"transformer"`
	Inputs      *Inputs       `json:"inputs" yaml:"inputs"`
	Compute     *SparkCompute `json:"compute" yaml:"compute"`
	Tags        Tags          `json:"tags" yaml:"tags"`
	FilePath    string        `json:"file_path"  yaml:"-"`
}

var transformedColumnValidation = &cr.StructValidation{
	StructFieldValidations: []*cr.StructFieldValidation{
		&cr.StructFieldValidation{
			StructField: "Name",
			StringValidation: &cr.StringValidation{
				Required:                   true,
				AlphaNumericDashUnderscore: true,
			},
		},
		&cr.StructFieldValidation{
			StructField: "Transformer",
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

func (transformedColumns TransformedColumns) Validate() error {
	resources := make([]Resource, len(transformedColumns))
	for i, res := range transformedColumns {
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

func (transformedColumn *TransformedColumn) GetName() string {
	return transformedColumn.Name
}

func (transformedColumn *TransformedColumn) GetResourceType() resource.Type {
	return resource.TransformedColumnType
}

func (transformedColumn *TransformedColumn) GetFilePath() string {
	return transformedColumn.FilePath
}

func (transformedColumns TransformedColumns) Names() []string {
	names := make([]string, len(transformedColumns))
	for i, transformedColumn := range transformedColumns {
		names[i] = transformedColumn.GetName()
	}
	return names
}

func (transformedColumns TransformedColumns) Get(name string) *TransformedColumn {
	for _, transformedColumn := range transformedColumns {
		if transformedColumn.GetName() == name {
			return transformedColumn
		}
	}
	return nil
}

func (transformedColumn *TransformedColumn) InputColumnNames() map[string]bool {
	inputs, _ := util.FlattenAllStrValuesAsSet(transformedColumn.Inputs.Columns)
	return inputs
}
