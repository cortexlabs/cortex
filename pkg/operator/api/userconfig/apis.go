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
	Model          string      `json:"model" yaml:"model"`
	ModelFormat    ModelFormat `json:"model_format" yaml:"model_format"`
	Tracker        *Tracker    `json:"tracker" yaml:"tracker"`
	RequestHandler *string     `json:"request_handler" yaml:"request_handler"`
	Compute        *APICompute `json:"compute" yaml:"compute"`
	Tags           Tags        `json:"tags" yaml:"tags"`
}

type Tracker struct {
	Key       string    `json:"key" yaml:"key"`
	ModelType ModelType `json:"model_type" yaml:"model_type"`
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
			StructField: "Tracker",
			StructValidation: &cr.StructValidation{
				DefaultNil: true,
				StructFieldValidations: []*cr.StructFieldValidation{
					{
						StructField: "Key",
						StringValidation: &cr.StringValidation{
							Required: true,
						},
					},
					{
						StructField: "ModelType",
						StringValidation: &cr.StringValidation{
							Required:      false,
							AllowEmpty:    true,
							AllowedValues: ModelTypeStrings(),
						},
						Parser: func(str string) (interface{}, error) {
							return ModelTypeFromString(str), nil
						},
					},
				},
			},
		},
		{
			StructField:         "RequestHandler",
			StringPtrValidation: &cr.StringPtrValidation{},
		},
		{
			StructField: "ModelFormat",
			StringValidation: &cr.StringValidation{
				Required:      false,
				AllowEmpty:    true,
				AllowedValues: append(ModelFormatStrings(), ""),
			},
			Parser: func(str string) (interface{}, error) {
				return ModelFormatFromString(str), nil
			},
		},
		apiComputeFieldValidation,
		tagsFieldValidation,
		typeFieldValidation,
	},
}

// IsValidS3Directory checks that the path contains a valid S3 directory for Tensorflow models
// Must contain the following structure:
// - 1523423423/ (version prefix, usually a timestamp)
// 		- saved_model.pb
//		- variables/
//			- variables.index
//			- variables.data-00000-of-00001 (there are a variable number of these files)
func IsValidS3Directory(path string) bool {
	if valid, err := aws.IsS3PathFileExternal(
		fmt.Sprintf("%s/saved_model.pb", path),
		fmt.Sprintf("%s/variables/variables.index", path),
	); err != nil || !valid {
		return false
	}

	if valid, err := aws.IsS3PathPrefixExternal(
		fmt.Sprintf("%s/variables/variables.data-00000-of", path),
	); err != nil || !valid {
		return false
	}
	return true
}

func (api *API) UserConfigStr() string {
	var sb strings.Builder
	sb.WriteString(api.ResourceFields.UserConfigStr())
	sb.WriteString(fmt.Sprintf("%s: %s\n", ModelKey, yaml.UnescapeAtSymbol(api.Model)))

	if api.ModelFormat != UnknownModelFormat {
		sb.WriteString(fmt.Sprintf("%s: %s\n", ModelFormatKey, api.ModelFormat.String()))
	}
	if api.RequestHandler != nil {
		sb.WriteString(fmt.Sprintf("%s: %s\n", RequestHandlerKey, *api.RequestHandler))
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
	if yaml.StartsWithEscapedAtSymbol(api.Model) {
		api.ModelFormat = TensorFlowModelFormat
		if err := api.Compute.Validate(); err != nil {
			return errors.Wrap(err, Identify(api), ComputeKey)
		}

		return nil
	}

	if !aws.IsValidS3Path(api.Model) {
		return errors.Wrap(ErrorInvalidS3PathOrResourceReference(api.Model), Identify(api), ModelKey)
	}

	switch api.ModelFormat {
	case ONNXModelFormat:
		if ok, err := aws.IsS3PathFileExternal(api.Model); err != nil || !ok {
			return errors.Wrap(ErrorExternalNotFound(api.Model), Identify(api), ModelKey)
		}
	case TensorFlowModelFormat:
		if !IsValidS3Directory(api.Model) {
			return errors.Wrap(ErrorInvalidTensorflowDir(api.Model), Identify(api), ModelKey)
		}
	default:
		switch {
		case strings.HasSuffix(api.Model, ".onnx"):
			api.ModelFormat = ONNXModelFormat
			if ok, err := aws.IsS3PathFileExternal(api.Model); err != nil || !ok {
				return errors.Wrap(ErrorExternalNotFound(api.Model), Identify(api), ModelKey)
			}
		case IsValidS3Directory(api.Model):
			api.ModelFormat = TensorFlowModelFormat
		default:
			return errors.Wrap(ErrorUnableToInferModelFormat(), Identify(api))
		}

	}

	if err := api.Compute.Validate(); err != nil {
		return errors.Wrap(err, Identify(api), ComputeKey)
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
