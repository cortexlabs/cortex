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
	Model              string      `json:"model" yaml:"model"`
	ModelType          ModelType   `json:"model_type" yaml:"model_type"`
	RequestHandlerPath *string     `json:"request_handler_path" yaml:"request_handler_path"`
	Compute            *APICompute `json:"compute" yaml:"compute"`
	Tags               Tags        `json:"tags" yaml:"tags"`
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
			StringValidation: &cr.StringValidation{
				Required:             true,
				AllowCortexResources: true,
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
		apiComputeFieldValidation,
		tagsFieldValidation,
		typeFieldValidation,
	},
}

func (api *API) UserConfigStr() string {
	var sb strings.Builder
	sb.WriteString(api.ResourceFields.UserConfigStr())
	sb.WriteString(fmt.Sprintf("%s: %s\n", ModelKey, yaml.UnescapeAtSymbol(api.Model)))

	if api.ModelType != UnknownModelType {
		sb.WriteString(fmt.Sprintf("%s: %s\n", ModelTypeKey, api.ModelType.String()))
	}
	if api.RequestHandlerPath != nil {
		sb.WriteString(fmt.Sprintf("%s: %s\n", RequestHandlerPathKey, *api.RequestHandlerPath))
	}
	if api.Compute != nil {
		sb.WriteString(fmt.Sprintf("%s:\n", ComputeKey))
		sb.WriteString(s.Indent(api.Compute.UserConfigStr(), "  "))
	}
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
	if !yaml.StartsWithEscapedAtSymbol(api.Model) {
		if !aws.IsValidS3Path(api.Model) {
			return errors.Wrap(ErrorInvalidS3PathOrResourceReference(api.Model), Identify(api), ModelKey)
		}

		if ok, err := aws.IsS3PathFileExternal(api.Model); err != nil || !ok {
			return errors.Wrap(ErrorExternalNotFound(api.Model), Identify(api), ModelKey)
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
