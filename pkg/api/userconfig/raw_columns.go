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
	cr "github.com/cortexlabs/cortex/pkg/lib/configreader"
)

type RawColumn interface {
	Column
	GetType() string
	GetCompute() *SparkCompute
	GetUserConfig() Resource
}

type RawColumns []RawColumn

var rawColumnValidation = &cr.InterfaceStructValidation{
	TypeKey:         "type",
	TypeStructField: "Type",
	InterfaceStructTypes: map[string]*cr.InterfaceStructType{
		"STRING_COLUMN": {
			Type:                   (*RawStringColumn)(nil),
			StructFieldValidations: rawStringColumnFieldValidations,
		},
		"INT_COLUMN": {
			Type:                   (*RawIntColumn)(nil),
			StructFieldValidations: rawIntColumnFieldValidations,
		},
		"FLOAT_COLUMN": {
			Type:                   (*RawFloatColumn)(nil),
			StructFieldValidations: rawFloatColumnFieldValidations,
		},
	},
}

type RawIntColumn struct {
	ResourceConfigFields
	Type     string        `json:"type" yaml:"type"`
	Required bool          `json:"required" yaml:"required"`
	Min      *int64        `json:"min" yaml:"min"`
	Max      *int64        `json:"max" yaml:"max"`
	Values   []int64       `json:"values" yaml:"values"`
	Compute  *SparkCompute `json:"compute" yaml:"compute"`
	Tags     Tags          `json:"tags" yaml:"tags"`
}

var rawIntColumnFieldValidations = []*cr.StructFieldValidation{
	{
		Key:         "name",
		StructField: "Name",
		StringValidation: &cr.StringValidation{
			Required:                   true,
			AlphaNumericDashUnderscore: true,
		},
	},
	{
		Key:         "required",
		StructField: "Required",
		BoolValidation: &cr.BoolValidation{
			Default: false,
		},
	},
	{
		Key:                "min",
		StructField:        "Min",
		Int64PtrValidation: &cr.Int64PtrValidation{},
	},
	{
		Key:                "max",
		StructField:        "Max",
		Int64PtrValidation: &cr.Int64PtrValidation{},
	},
	{
		Key:         "values",
		StructField: "Values",
		Int64ListValidation: &cr.Int64ListValidation{
			AllowNull: true,
		},
	},
	sparkComputeFieldValidation,
	tagsFieldValidation,
	typeFieldValidation,
}

type RawFloatColumn struct {
	ResourceConfigFields
	Type     string        `json:"type" yaml:"type"`
	Required bool          `json:"required" yaml:"required"`
	Min      *float32      `json:"min" yaml:"min"`
	Max      *float32      `json:"max" yaml:"max"`
	Values   []float32     `json:"values" yaml:"values"`
	Compute  *SparkCompute `json:"compute" yaml:"compute"`
	Tags     Tags          `json:"tags" yaml:"tags"`
}

var rawFloatColumnFieldValidations = []*cr.StructFieldValidation{
	{
		Key:         "name",
		StructField: "Name",
		StringValidation: &cr.StringValidation{
			Required:                   true,
			AlphaNumericDashUnderscore: true,
		},
	},
	{
		Key:         "required",
		StructField: "Required",
		BoolValidation: &cr.BoolValidation{
			Default: false,
		},
	},
	{
		Key:                  "min",
		StructField:          "Min",
		Float32PtrValidation: &cr.Float32PtrValidation{},
	},
	{
		Key:                  "max",
		StructField:          "Max",
		Float32PtrValidation: &cr.Float32PtrValidation{},
	},
	{
		Key:         "values",
		StructField: "Values",
		Float32ListValidation: &cr.Float32ListValidation{
			AllowNull: true,
		},
	},
	sparkComputeFieldValidation,
	tagsFieldValidation,
	typeFieldValidation,
}

type RawStringColumn struct {
	ResourceConfigFields
	Type     string        `json:"type" yaml:"type"`
	Required bool          `json:"required" yaml:"required"`
	Values   []string      `json:"values" yaml:"values"`
	Compute  *SparkCompute `json:"compute" yaml:"compute"`
	Tags     Tags          `json:"tags" yaml:"tags"`
}

var rawStringColumnFieldValidations = []*cr.StructFieldValidation{
	{
		Key:         "name",
		StructField: "Name",
		StringValidation: &cr.StringValidation{
			AlphaNumericDashUnderscore: true,
			Required:                   true,
		},
	},
	{
		Key:         "required",
		StructField: "Required",
		BoolValidation: &cr.BoolValidation{
			Default: false,
		},
	},
	{
		Key:         "values",
		StructField: "Values",
		StringListValidation: &cr.StringListValidation{
			AllowNull: true,
		},
	},
	sparkComputeFieldValidation,
	tagsFieldValidation,
	typeFieldValidation,
}

func (rawColumns RawColumns) Validate() error {
	resources := make([]Resource, len(rawColumns))
	for i, res := range rawColumns {
		resources[i] = res
	}

	dups := FindDuplicateResourceName(resources...)
	if len(dups) > 0 {
		return ErrorDuplicateResourceName(dups...)
	}

	return nil
}

func (rawColumns RawColumns) Names() []string {
	names := []string{}
	for _, column := range rawColumns {
		names = append(names, column.GetName())
	}
	return names
}

func (rawColumns RawColumns) Get(name string) RawColumn {
	for _, column := range rawColumns {
		if column.GetName() == name {
			return column
		}
	}
	return nil
}

func (column *RawIntColumn) GetType() string {
	return column.Type
}

func (column *RawFloatColumn) GetType() string {
	return column.Type
}

func (column *RawStringColumn) GetType() string {
	return column.Type
}

func (column *RawIntColumn) GetCompute() *SparkCompute {
	return column.Compute
}

func (column *RawFloatColumn) GetCompute() *SparkCompute {
	return column.Compute
}

func (column *RawStringColumn) GetCompute() *SparkCompute {
	return column.Compute
}

func (column *RawIntColumn) GetResourceType() resource.Type {
	return resource.RawColumnType
}

func (column *RawFloatColumn) GetResourceType() resource.Type {
	return resource.RawColumnType
}

func (column *RawStringColumn) GetResourceType() resource.Type {
	return resource.RawColumnType
}

func (column *RawIntColumn) IsRaw() bool {
	return true
}

func (column *RawFloatColumn) IsRaw() bool {
	return true
}

func (column *RawStringColumn) IsRaw() bool {
	return true
}

func (column *RawIntColumn) GetUserConfig() Resource {
	return column
}

func (column *RawFloatColumn) GetUserConfig() Resource {
	return column
}

func (column *RawStringColumn) GetUserConfig() Resource {
	return column
}
