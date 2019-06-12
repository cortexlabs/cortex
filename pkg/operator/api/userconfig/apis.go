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
	"github.com/cortexlabs/yaml"

	cr "github.com/cortexlabs/cortex/pkg/lib/configreader"
	"github.com/cortexlabs/cortex/pkg/lib/errors"
	"github.com/cortexlabs/cortex/pkg/operator/api/resource"
)

type APIs []*API

type API struct {
	ResourceFields
	Model     string      `json:"model" yaml:"model"`
	ModelPath *string     `json:"model_path" yaml:"model_path"`
	Compute   *APICompute `json:"compute" yaml:"compute"`
	Tags      Tags        `json:"tags" yaml:"tags"`
}

var apiValidation = &cr.StructValidation{
	StructFieldValidations: []*cr.StructFieldValidation{
		{
			StructField: "Name",
			StringValidation: &cr.StringValidation{
				Required: true,
				DNS1035:  true,
			},
		},
		{
			StructField:  "Model",
			DefaultField: "Name",
			DefaultFieldFunc: func(name interface{}) interface{} {
				model := "@" + name.(string)
				escapedModel, _ := yaml.EscapeAtSymbol(model)
				return escapedModel
			},
			StringValidation: &cr.StringValidation{
				Required:               false,
				RequireCortexResources: true,
			},
		},
		{
			StructField:         "ModelPath",
			StringPtrValidation: &configreader.StringPtrValidation{},
		},
		apiComputeFieldValidation,
		tagsFieldValidation,
		typeFieldValidation,
	},
}

func (apis APIs) Validate() error {
	resources := make([]Resource, len(apis))
	for i, res := range apis {
		if err := res.Validate(); err != nil {
			return err
		}

		resources[i] = res
	}

	dups := FindDuplicateResourceName(resources...)
	if len(dups) > 0 {
		return ErrorDuplicateResourceName(dups...)
	}
	return nil
}

func (api *API) Validate() error {
	if api.ModelPath == nil && api.ModelName == "" {
		return errors.Wrap(ErrorSpecifyOnlyOneMissing("model_name", "model_path"), Identify(api))
	}

	if api.ModelPath != nil && api.ModelName != "" {
		return errors.Wrap(ErrorSpecifyOnlyOne("model_name", "model_path"), Identify(api))
	}

	return nil
}

func (api *API) GetResourceType() resource.Type {
	return resource.APIType
}

func (apis APIs) Names() []string {
	names := make([]string, len(apis))
	for i, api := range apis {
		names[i] = api.Name
	}
	return names
}
