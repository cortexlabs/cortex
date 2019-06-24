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
	"fmt"
	"strings"

	cr "github.com/cortexlabs/cortex/pkg/lib/configreader"
	s "github.com/cortexlabs/cortex/pkg/lib/strings"
	"github.com/cortexlabs/cortex/pkg/operator/api/resource"
)

type RawColumn interface {
	Column
	GetColumnType() ColumnType
	GetCompute() *SparkCompute
}

type RawColumns []RawColumn

var rawColumnValidation = &cr.InterfaceStructValidation{
	TypeKey:         "type",
	TypeStructField: "Type",
	ParsedInterfaceStructTypes: map[interface{}]*cr.InterfaceStructType{
		StringColumnType: {
			Type:                   (*RawStringColumn)(nil),
			StructFieldValidations: rawStringColumnFieldValidations,
		},
		IntegerColumnType: {
			Type:                   (*RawIntColumn)(nil),
			StructFieldValidations: rawIntColumnFieldValidations,
		},
		FloatColumnType: {
			Type:                   (*RawFloatColumn)(nil),
			StructFieldValidations: rawFloatColumnFieldValidations,
		},
	},
	Parser: func(str string) (interface{}, error) {
		return ColumnTypeFromString(str), nil
	},
}

type RawIntColumn struct {
	ResourceFields
	Type     ColumnType    `json:"type" yaml:"type"`
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
		Key:                 "values",
		StructField:         "Values",
		Int64ListValidation: &cr.Int64ListValidation{},
	},
	sparkComputeFieldValidation("Compute"),
	tagsFieldValidation,
	typeFieldValidation,
}

func (column *RawIntColumn) UserConfigStr() string {
	var sb strings.Builder
	sb.WriteString(column.ResourceFields.UserConfigStr())
	sb.WriteString(fmt.Sprintf("%s: %s\n", TypeKey, column.Type.String()))
	sb.WriteString(fmt.Sprintf("%s: %s\n", RequiredKey, s.Bool(column.Required)))
	if column.Min != nil {
		sb.WriteString(fmt.Sprintf("%s: %s\n", MinKey, s.Int64(*column.Min)))
	}
	if column.Max != nil {
		sb.WriteString(fmt.Sprintf("%s: %s\n", MaxKey, s.Int64(*column.Max)))
	}
	if len(column.Values) != 0 {
		sb.WriteString(fmt.Sprintf("%s: %s\n", ValuesKey, s.Obj(column.Values)))
	}
	if column.Compute != nil {
		sb.WriteString(fmt.Sprintf("%s:\n", ComputeKey))
		sb.WriteString(s.Indent(column.Compute.UserConfigStr(), "  "))
	}
	return sb.String()
}

type RawFloatColumn struct {
	ResourceFields
	Type     ColumnType    `json:"type" yaml:"type"`
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
		Key:                   "values",
		StructField:           "Values",
		Float32ListValidation: &cr.Float32ListValidation{},
	},
	sparkComputeFieldValidation("Compute"),
	tagsFieldValidation,
	typeFieldValidation,
}

func (column *RawFloatColumn) UserConfigStr() string {
	var sb strings.Builder
	sb.WriteString(column.ResourceFields.UserConfigStr())
	sb.WriteString(fmt.Sprintf("%s: %s\n", TypeKey, column.Type.String()))
	sb.WriteString(fmt.Sprintf("%s: %s\n", RequiredKey, s.Bool(column.Required)))
	if column.Min != nil {
		sb.WriteString(fmt.Sprintf("%s: %s\n", MinKey, s.Float32(*column.Min)))
	}
	if column.Max != nil {
		sb.WriteString(fmt.Sprintf("%s: %s\n", MaxKey, s.Float32(*column.Max)))
	}
	if len(column.Values) != 0 {
		sb.WriteString(fmt.Sprintf("%s: %s\n", ValuesKey, s.Obj(column.Values)))
	}
	if column.Compute != nil {
		sb.WriteString(fmt.Sprintf("%s:\n", ComputeKey))
		sb.WriteString(s.Indent(column.Compute.UserConfigStr(), "  "))
	}
	return sb.String()
}

type RawStringColumn struct {
	ResourceFields
	Type     ColumnType    `json:"type" yaml:"type"`
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
		Key:                  "values",
		StructField:          "Values",
		StringListValidation: &cr.StringListValidation{},
	},
	sparkComputeFieldValidation("Compute"),
	tagsFieldValidation,
	typeFieldValidation,
}

func (column *RawStringColumn) UserConfigStr() string {
	var sb strings.Builder
	sb.WriteString(column.ResourceFields.UserConfigStr())
	sb.WriteString(fmt.Sprintf("%s: %s\n", TypeKey, column.Type.String()))
	sb.WriteString(fmt.Sprintf("%s: %s\n", RequiredKey, s.Bool(column.Required)))
	if len(column.Values) != 0 {
		sb.WriteString(fmt.Sprintf("%s: %s\n", ValuesKey, s.Obj(column.Values)))
	}
	if column.Compute != nil {
		sb.WriteString(fmt.Sprintf("%s:\n", ComputeKey))
		sb.WriteString(s.Indent(column.Compute.UserConfigStr(), "  "))
	}
	return sb.String()
}

type RawInferredColumn struct {
	ResourceFields
	Type    ColumnType    `json:"type" yaml:"type"`
	Compute *SparkCompute `json:"compute" yaml:"compute"`
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

func (column *RawInferredColumn) UserConfigStr() string {
	var sb strings.Builder
	sb.WriteString(column.ResourceFields.UserConfigStr())
	if column.Compute != nil {
		sb.WriteString(fmt.Sprintf("%s:\n", ComputeKey))
		sb.WriteString(s.Indent(column.Compute.UserConfigStr(), "  "))
	}
	return sb.String()
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

func (column *RawIntColumn) GetColumnType() ColumnType {
	return column.Type
}

func (column *RawFloatColumn) GetColumnType() ColumnType {
	return column.Type
}

func (column *RawStringColumn) GetColumnType() ColumnType {
	return column.Type
}

func (column *RawInferredColumn) GetColumnType() ColumnType {
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

func (column *RawInferredColumn) GetCompute() *SparkCompute {
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

func (column *RawInferredColumn) GetResourceType() resource.Type {
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

func (column *RawInferredColumn) IsRaw() bool {
	return true
}
