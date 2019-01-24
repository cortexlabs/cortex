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

type TransformedFeatures []*TransformedFeature

type TransformedFeature struct {
	Name        string        `json:"name" yaml:"name"`
	Transformer string        `json:"transformer" yaml:"transformer"`
	Inputs      *Inputs       `json:"inputs" yaml:"inputs"`
	Compute     *SparkCompute `json:"compute" yaml:"compute"`
	Tags        Tags          `json:"tags" yaml:"tags"`
}

var transformedFeatureValidation = &cr.StructValidation{
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

func (transformedFeatures TransformedFeatures) Validate() error {
	dups := util.FindDuplicateStrs(transformedFeatures.Names())
	if len(dups) > 0 {
		return ErrorDuplicateConfigName(dups[0], resource.TransformedFeatureType)
	}
	return nil
}

func (feature *TransformedFeature) IsRaw() bool {
	return false
}

func (transformedFeature *TransformedFeature) GetName() string {
	return transformedFeature.Name
}

func (transformedFeature *TransformedFeature) GetResourceType() resource.Type {
	return resource.TransformedFeatureType
}

func (transformedFeatures TransformedFeatures) Names() []string {
	names := make([]string, len(transformedFeatures))
	for i, transformedFeature := range transformedFeatures {
		names[i] = transformedFeature.GetName()
	}
	return names
}

func (transformedFeatures TransformedFeatures) Get(name string) *TransformedFeature {
	for _, transformedFeature := range transformedFeatures {
		if transformedFeature.GetName() == name {
			return transformedFeature
		}
	}
	return nil
}

func (transformedFeature *TransformedFeature) InputFeatureNames() map[string]bool {
	inputs, _ := util.FlattenAllStrValuesAsSet(transformedFeature.Inputs.Features)
	return inputs
}
