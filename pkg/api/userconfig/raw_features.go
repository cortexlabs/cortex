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

type RawFeature interface {
	Feature
	GetType() string
	GetCompute() *SparkCompute
	GetUserConfig() Resource
	GetResourceType() resource.Type
}

type RawFeatures []RawFeature

var rawFeatureValidation = &cr.InterfaceStructValidation{
	TypeKey:         "type",
	TypeStructField: "Type",
	InterfaceStructTypes: map[string]*cr.InterfaceStructType{
		"STRING_FEATURE": &cr.InterfaceStructType{
			Type:                   (*RawStringFeature)(nil),
			StructFieldValidations: rawStringFeatureFieldValidations,
		},
		"INT_FEATURE": &cr.InterfaceStructType{
			Type:                   (*RawIntFeature)(nil),
			StructFieldValidations: rawIntFeatureFieldValidations,
		},
		"FLOAT_FEATURE": &cr.InterfaceStructType{
			Type:                   (*RawFloatFeature)(nil),
			StructFieldValidations: rawFloatFeatureFieldValidations,
		},
	},
}

type RawIntFeature struct {
	Name     string        `json:"name" yaml:"name"`
	Type     string        `json:"type" yaml:"type"`
	Required bool          `json:"required" yaml:"required"`
	Min      *int64        `json:"min" yaml:"min"`
	Max      *int64        `json:"max" yaml:"max"`
	Values   []int64       `json:"values" yaml:"values"`
	Compute  *SparkCompute `json:"compute" yaml:"compute"`
	Tags     Tags          `json:"tags" yaml:"tags"`
}

var rawIntFeatureFieldValidations = []*cr.StructFieldValidation{
	&cr.StructFieldValidation{
		Key:         "name",
		StructField: "Name",
		StringValidation: &cr.StringValidation{
			Required:                   true,
			AlphaNumericDashUnderscore: true,
		},
	},
	&cr.StructFieldValidation{
		Key:         "required",
		StructField: "Required",
		BoolValidation: &cr.BoolValidation{
			Default: false,
		},
	},
	&cr.StructFieldValidation{
		Key:                "min",
		StructField:        "Min",
		Int64PtrValidation: &cr.Int64PtrValidation{},
	},
	&cr.StructFieldValidation{
		Key:                "max",
		StructField:        "Max",
		Int64PtrValidation: &cr.Int64PtrValidation{},
	},
	&cr.StructFieldValidation{
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

type RawFloatFeature struct {
	Name     string        `json:"name" yaml:"name"`
	Type     string        `json:"type" yaml:"type"`
	Required bool          `json:"required" yaml:"required"`
	Min      *float32      `json:"min" yaml:"min"`
	Max      *float32      `json:"max" yaml:"max"`
	Values   []float32     `json:"values" yaml:"values"`
	Compute  *SparkCompute `json:"compute" yaml:"compute"`
	Tags     Tags          `json:"tags" yaml:"tags"`
}

var rawFloatFeatureFieldValidations = []*cr.StructFieldValidation{
	&cr.StructFieldValidation{
		Key:         "name",
		StructField: "Name",
		StringValidation: &cr.StringValidation{
			Required:                   true,
			AlphaNumericDashUnderscore: true,
		},
	},
	&cr.StructFieldValidation{
		Key:         "required",
		StructField: "Required",
		BoolValidation: &cr.BoolValidation{
			Default: false,
		},
	},
	&cr.StructFieldValidation{
		Key:                  "min",
		StructField:          "Min",
		Float32PtrValidation: &cr.Float32PtrValidation{},
	},
	&cr.StructFieldValidation{
		Key:                  "max",
		StructField:          "Max",
		Float32PtrValidation: &cr.Float32PtrValidation{},
	},
	&cr.StructFieldValidation{
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

type RawStringFeature struct {
	Name     string        `json:"name" yaml:"name"`
	Type     string        `json:"type" yaml:"type"`
	Required bool          `json:"required" yaml:"required"`
	Values   []string      `json:"values" yaml:"values"`
	Compute  *SparkCompute `json:"compute" yaml:"compute"`
	Tags     Tags          `json:"tags" yaml:"tags"`
}

var rawStringFeatureFieldValidations = []*cr.StructFieldValidation{
	&cr.StructFieldValidation{
		Key:         "name",
		StructField: "Name",
		StringValidation: &cr.StringValidation{
			AlphaNumericDashUnderscore: true,
			Required:                   true,
		},
	},
	&cr.StructFieldValidation{
		Key:         "required",
		StructField: "Required",
		BoolValidation: &cr.BoolValidation{
			Default: false,
		},
	},
	&cr.StructFieldValidation{
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

func (rawFeatures *RawFeatures) Validate() error {
	dups := util.FindDuplicateStrs(rawFeatures.Names())
	if len(dups) > 0 {
		return ErrorDuplicateConfigName(dups[0], resource.RawFeatureType)
	}
	return nil
}

func (rawFeatures RawFeatures) Names() []string {
	names := []string{}
	for _, feature := range rawFeatures {
		names = append(names, feature.GetName())
	}
	return names
}

func (rawFeatures RawFeatures) Get(name string) RawFeature {
	for _, feature := range rawFeatures {
		if feature.GetName() == name {
			return feature
		}
	}
	return nil
}

func (feature *RawIntFeature) GetName() string {
	return feature.Name
}

func (feature *RawFloatFeature) GetName() string {
	return feature.Name
}

func (feature *RawStringFeature) GetName() string {
	return feature.Name
}

func (feature *RawIntFeature) GetType() string {
	return feature.Type
}

func (feature *RawFloatFeature) GetType() string {
	return feature.Type
}

func (feature *RawStringFeature) GetType() string {
	return feature.Type
}

func (feature *RawIntFeature) GetCompute() *SparkCompute {
	return feature.Compute
}

func (feature *RawFloatFeature) GetCompute() *SparkCompute {
	return feature.Compute
}

func (feature *RawStringFeature) GetCompute() *SparkCompute {
	return feature.Compute
}

func (feature *RawIntFeature) GetResourceType() resource.Type {
	return resource.RawFeatureType
}

func (feature *RawFloatFeature) GetResourceType() resource.Type {
	return resource.RawFeatureType
}

func (feature *RawStringFeature) GetResourceType() resource.Type {
	return resource.RawFeatureType
}

func (feature *RawIntFeature) IsRaw() bool {
	return true
}

func (feature *RawFloatFeature) IsRaw() bool {
	return true
}

func (feature *RawStringFeature) IsRaw() bool {
	return true
}

func (feature *RawIntFeature) GetUserConfig() Resource {
	return feature
}

func (feature *RawFloatFeature) GetUserConfig() Resource {
	return feature
}

func (feature *RawStringFeature) GetUserConfig() Resource {
	return feature
}
