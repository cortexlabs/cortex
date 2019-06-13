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
	"github.com/cortexlabs/cortex/pkg/lib/aws"
	cr "github.com/cortexlabs/cortex/pkg/lib/configreader"
	"github.com/cortexlabs/cortex/pkg/lib/errors"
	"github.com/cortexlabs/cortex/pkg/operator/api/resource"
)

type APIs []*API

type API struct {
	ResourceFields
	Model         *string        `json:"model" yaml:"model"`
	ExternalModel *ExternalModel `json:"external_model" yaml:"external_model"`
	Compute       *APICompute    `json:"compute" yaml:"compute"`
	Tags          Tags           `json:"tags" yaml:"tags"`
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
			StructField: "Model",
			StringPtrValidation: &cr.StringPtrValidation{
				RequireCortexResources: true,
			},
		},
		externalModelFieldValidation,
		apiComputeFieldValidation,
		tagsFieldValidation,
		typeFieldValidation,
	},
}

type ExternalModel struct {
	Path   string `json:"path" yaml:"path"`
	Region string `json:"region" yaml:"region"`
}

var externalModelFieldValidation = &cr.StructFieldValidation{
	StructField: "ExternalModel",
	StructValidation: &cr.StructValidation{
		StructFieldValidations: []*cr.StructFieldValidation{
			{
				StructField: "Path",
				StringValidation: &cr.StringValidation{
					Validator: cr.GetS3PathValidator(),
				},
			},
			{
				StructField:      "Region",
				StringValidation: &cr.StringValidation{},
			},
		},
	},
}

func (apis APIs) Validate() error {
	for _, api := range apis {
		if err := api.Validate(); err != nil {
			return err
		}
	}

	resources := make([]Resource, len(apis))
	for i, res := range apis {
		resources[i] = res
	}

	dups := FindDuplicateResourceName(resources...)
	if len(dups) > 0 {
		return ErrorDuplicateResourceName(dups...)
	}

	return nil
}

func (api *API) Validate() error {
	if api.ExternalModel == nil && api.Model == nil {
		return errors.Wrap(ErrorSpecifyOnlyOneMissing("model", "external_model"), Identify(api))
	}

	if api.ExternalModel != nil && api.Model != nil {
		return errors.Wrap(ErrorSpecifyOnlyOne("model", "external_model"), Identify(api))
	}

	if api.ExternalModel != nil {
		bucket, key, err := aws.SplitS3Path(api.ExternalModel.Path)
		if err != nil {
			return err
		}

		if ok, err := aws.IsS3FileExternal(bucket, key, api.ExternalModel.Region); err != nil || !ok {
			return ErrorExternalModelNotFound(api.ExternalModel.Path)
		}
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
