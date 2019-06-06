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

type Estimators []*Estimator

type Estimator struct {
	ResourceFields
	TargetColumn  ColumnType   `json:"target_column"  yaml:"target_column"`
	Input         *InputSchema `json:"input" yaml:"input"`
	TrainingInput *InputSchema `json:"training_input" yaml:"training_input"`
	Hparams       *InputSchema `json:"hparams"  yaml:"hparams"`
	PredictionKey string       `json:"prediction_key" yaml:"prediction_key"`
	Path          string       `json:"path"  yaml:"path"`
}

var estimatorValidation = &cr.StructValidation{
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
				return "implementations/estimators/" + name.(string) + ".py"
			},
		},
		{
			StructField: "TargetColumn",
			StringValidation: &cr.StringValidation{
				Required: true,
				Validator: func(col string) (string, error) {
					colType := ColumnTypeFromString(col)
					if colType != IntegerColumnType && colType != FloatColumnType {
						return "", ErrorTargetColumnIntOrFloat()
					}
					return col, nil
				},
			},
			Parser: func(str string) (interface{}, error) {
				return ColumnTypeFromString(str), nil
			},
		},
		{
			StructField: "Input",
			InterfaceValidation: &cr.InterfaceValidation{
				Required:  true,
				Validator: inputSchemaValidator,
			},
		},
		{
			StructField: "TrainingInput",
			InterfaceValidation: &cr.InterfaceValidation{
				Required:  false,
				Validator: inputSchemaValidator,
			},
		},
		{
			StructField: "Hparams",
			InterfaceValidation: &cr.InterfaceValidation{
				Required:  false,
				Validator: inputSchemaValidatorValueTypesOnly,
			},
		},
		{
			StructField: "PredictionKey",
			StringValidation: &cr.StringValidation{
				Default:    "",
				AllowEmpty: true,
			},
		},
		typeFieldValidation,
	},
}

func (estimators Estimators) Validate() error {
	resources := make([]Resource, len(estimators))
	for i, res := range estimators {
		resources[i] = res
	}

	dups := FindDuplicateResourceName(resources...)
	if len(dups) > 0 {
		return ErrorDuplicateResourceName(dups...)
	}

	return nil
}

func (estimators Estimators) Get(name string) *Estimator {
	for _, estimator := range estimators {
		if estimator.Name == name {
			return estimator
		}
	}
	return nil
}

func (estimator *Estimator) GetResourceType() resource.Type {
	return resource.EstimatorType
}

func (estimators Estimators) Names() []string {
	names := make([]string, len(estimators))
	for i, estimator := range estimators {
		names[i] = estimator.Name
	}
	return names
}
