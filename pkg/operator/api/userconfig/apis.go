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

	"github.com/cortexlabs/yaml"

	"github.com/cortexlabs/cortex/pkg/lib/aws"
	cr "github.com/cortexlabs/cortex/pkg/lib/configreader"
	"github.com/cortexlabs/cortex/pkg/lib/errors"
	s "github.com/cortexlabs/cortex/pkg/lib/strings"
	"github.com/cortexlabs/cortex/pkg/operator/api/resource"
)

type APIs []*API

type API struct {
	ResourceFields
	ModelType          ModelType      `json:"model_type" yaml:"model_type"`
	Model              *string        `json:"model" yaml:"model"`
	RequestHandlerPath *string        `json:"request_handler_path" yaml:"request_handler_path"`
	ExternalModel      *ExternalModel `json:"external_model" yaml:"external_model"`
	Compute            *APICompute    `json:"compute" yaml:"compute"`
	Tags               Tags           `json:"tags" yaml:"tags"`
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
		{
			StructField:         "RequestHandlerPath",
			StringPtrValidation: &cr.StringPtrValidation{},
		},
		{
			StructField: "ModelType",
			StringValidation: &cr.StringValidation{
				AllowedValues: ModelTypeStrings(),
			},
			Parser: func(str string) (interface{}, error) {
				return ModelTypeFromString(str), nil
			},
		},
		{
			StructField:      "ExternalModel",
			StructValidation: externalModelFieldValidation,
		},
		apiComputeFieldValidation,
		tagsFieldValidation,
		typeFieldValidation,
	},
}

func (api *API) UserConfigStr() string {
	var sb strings.Builder
	sb.WriteString(api.ResourceFields.UserConfigStr())
	if api.Model != nil {
		sb.WriteString(fmt.Sprintf("%s: %s\n", ModelKey, yaml.UnescapeAtSymbol(*api.Model)))
	}
	if api.ModelType != UnknownModelType {
		sb.WriteString(fmt.Sprintf("%s: %s\n", ModelTypeKey, api.ModelType.String()))
	}
	if api.ServingFunction != nil {
		sb.WriteString(fmt.Sprintf("%s: %s\n", RequestHandlerPathKey, *api.RequestHandlerPath))
	}
	if api.ExternalModel != nil {
		sb.WriteString(fmt.Sprintf("%s:\n", ExternalModelKey))
		sb.WriteString(s.Indent(api.ExternalModel.UserConfigStr(), "  "))
	}
	if api.Compute != nil {
		sb.WriteString(fmt.Sprintf("%s:\n", ComputeKey))
		sb.WriteString(s.Indent(api.Compute.UserConfigStr(), "  "))
	}
	return sb.String()
}

type ExternalModel struct {
	Path   string `json:"path" yaml:"path"`
	Region string `json:"region" yaml:"region"`
}

var externalModelFieldValidation = &cr.StructValidation{
	DefaultNil: true,
	StructFieldValidations: []*cr.StructFieldValidation{
		{
			StructField: "Path",
			StringValidation: &cr.StringValidation{
				Validator: cr.GetS3PathValidator(),
				Required:  true,
			},
		},
		{
			StructField: "Region",
			StringValidation: &cr.StringValidation{
				Default:       aws.DefaultS3Region,
				AllowedValues: aws.S3Regions.Slice(),
			},
		},
	},
}

func (extModel *ExternalModel) UserConfigStr() string {
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("%s: %s\n", PathKey, extModel.Path))
	sb.WriteString(fmt.Sprintf("%s: %s\n", RegionKey, extModel.Region))
	return sb.String()
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
		return errors.Wrap(ErrorSpecifyOnlyOneMissing(ModelKey, ExternalModelKey), Identify(api))
	}

	if api.ExternalModel != nil && api.Model != nil {
		return errors.Wrap(ErrorSpecifyOnlyOne(ModelKey, ExternalModelKey), Identify(api))
	}

	if api.ExternalModel != nil {
		bucket, key, err := aws.SplitS3Path(api.ExternalModel.Path)
		if err != nil {
			return errors.Wrap(err, Identify(api), ExternalModelKey, PathKey)
		}

		if ok, err := aws.IsS3FileExternal(bucket, key, api.ExternalModel.Region); err != nil || !ok {
			return errors.Wrap(ErrorExternalNotFound(api.ExternalModel.Path), Identify(api), ExternalModelKey, PathKey)
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
