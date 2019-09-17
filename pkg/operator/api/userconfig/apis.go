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
	"path/filepath"
	"strconv"
	"strings"

	"github.com/aws/aws-sdk-go/service/s3"
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
	TFSignatureKey *string     `json:"tf_signature_key" yaml:"tf_signature_key"`
	Compute        *APICompute `json:"compute" yaml:"compute"`
	Tags           Tags        `json:"tags" yaml:"tags"`
}

type Tracker struct {
	Key       *string   `json:"key" yaml:"key"`
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
				Required:  true,
				Validator: cr.GetS3PathValidator(),
			},
		},
		{
			StructField: "Tracker",
			StructValidation: &cr.StructValidation{
				DefaultNil: true,
				StructFieldValidations: []*cr.StructFieldValidation{
					{
						StructField:         "Key",
						StringPtrValidation: &cr.StringPtrValidation{},
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
		{
			StructField: "TFSignatureKey",
			StringPtrValidation: &cr.StringPtrValidation{
				Required: false,
			},
		},
		apiComputeFieldValidation,
		tagsFieldValidation,
		typeFieldValidation,
	},
}

// IsValidTensorFlowS3Directory checks that the path contains a valid S3 directory for Tensorflow models
// Must contain the following structure:
// - 1523423423/ (version prefix, usually a timestamp)
// 		- saved_model.pb
//		- variables/
//			- variables.index
//			- variables.data-00000-of-00001 (there are a variable number of these files)
func IsValidTensorFlowS3Directory(path string, awsClient *aws.Client) bool {
	if valid, err := awsClient.IsS3PathFile(
		aws.S3PathJoin(path, "saved_model.pb"),
		aws.S3PathJoin(path, "variables/variables.index"),
	); err != nil || !valid {
		return false
	}

	if valid, err := awsClient.IsS3PathPrefix(
		aws.S3PathJoin(path, "variables/variables.data-00000-of"),
	); err != nil || !valid {
		return false
	}
	return true
}

func GetTFServingExportFromS3Path(path string, awsClient *aws.Client) (string, error) {
	if IsValidTensorFlowS3Directory(path, awsClient) {
		return path, nil
	}

	bucket, prefix, err := aws.SplitS3Path(path)
	if err != nil {
		return "", err
	}

	resp, _ := awsClient.S3.ListObjects(&s3.ListObjectsInput{
		Bucket: &bucket,
		Prefix: &prefix,
	})

	highestVersion := int64(0)
	var highestPath string
	for _, key := range resp.Contents {
		if !strings.HasSuffix(*key.Key, "saved_model.pb") {
			continue
		}

		keyParts := strings.Split(*key.Key, "/")
		versionStr := keyParts[len(keyParts)-1]
		version, err := strconv.ParseInt(versionStr, 10, 64)
		if err != nil {
			version = 0
		}

		possiblePath := "s3://" + filepath.Join(bucket, filepath.Join(keyParts[:len(keyParts)-1]...))
		if version >= highestVersion && IsValidTensorFlowS3Directory(possiblePath, awsClient) {
			highestVersion = version
			highestPath = possiblePath
		}
	}

	return highestPath, nil
}

func (api *API) UserConfigStr() string {
	var sb strings.Builder
	sb.WriteString(api.ResourceFields.UserConfigStr())
	sb.WriteString(fmt.Sprintf("%s: %s\n", ModelKey, api.Model))
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

func (apis APIs) Validate(projectFileMap map[string][]byte) error {
	for _, api := range apis {
		if err := api.Validate(projectFileMap); err != nil {
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

func (api *API) Validate(projectFileMap map[string][]byte) error {
	awsClient, err := aws.NewFromS3Path(api.Model, false)
	if err != nil {
		return err
	}

	switch {
	case api.ModelFormat == ONNXModelFormat:
		if strings.HasSuffix(api.Model, ".zip") {
			return errors.Wrap(ErrorONNXDoesntSupportZip(), Identify(api), ModelKey)
		}
		if ok, err := awsClient.IsS3PathFile(api.Model); err != nil || !ok {
			return errors.Wrap(ErrorExternalNotFound(api.Model), Identify(api), ModelKey)
		}
	case api.ModelFormat == TensorFlowModelFormat:
		if strings.HasSuffix(api.Model, ".zip") {
			if ok, err := awsClient.IsS3PathFile(api.Model); err != nil || !ok {
				return errors.Wrap(ErrorExternalNotFound(api.Model), Identify(api), ModelKey)
			}
		} else {
			path, err := GetTFServingExportFromS3Path(api.Model, awsClient)
			if path == "" || err != nil {
				return errors.Wrap(ErrorInvalidTensorflowDir(api.Model), Identify(api), ModelKey)
			}
			api.Model = path
		}
	default:
		switch {
		case strings.HasSuffix(api.Model, ".onnx"):
			api.ModelFormat = ONNXModelFormat
			if ok, err := awsClient.IsS3PathFile(api.Model); err != nil || !ok {
				return errors.Wrap(ErrorExternalNotFound(api.Model), Identify(api), ModelKey)
			}
		case strings.HasSuffix(api.Model, ".zip"):
			api.ModelFormat = TensorFlowModelFormat
			if ok, err := awsClient.IsS3PathFile(api.Model); err != nil || !ok {
				return errors.Wrap(ErrorExternalNotFound(api.Model), Identify(api), ModelKey)
			}
		default:
			path, err := GetTFServingExportFromS3Path(api.Model, awsClient)
			if err != nil {
				return errors.Wrap(err, Identify(api), ModelKey)
			}
			if path == "" {
				return errors.Wrap(ErrorUnableToInferModelFormat(api.Model), Identify(api))
			}
			api.ModelFormat = TensorFlowModelFormat
			api.Model = path
		}
	}

	if api.ModelFormat != TensorFlowModelFormat && api.TFServing != nil {
		return errors.Wrap(ErrorTFServingOptionsForTFOnly(api.ModelFormat), Identify(api))
	}

	if api.RequestHandler != nil {
		if _, ok := projectFileMap[*api.RequestHandler]; !ok {
			return errors.Wrap(ErrorImplDoesNotExist(*api.RequestHandler), Identify(api), RequestHandlerKey)
		}
	}

	if err := api.Compute.Validate(); err != nil {
		return errors.Wrap(err, Identify(api), ComputeKey)
	}

	return nil
}

func (api *API) AreProjectFilesRequired() bool {
	return api.RequestHandler != nil
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

func (apis APIs) AreProjectFilesRequired() bool {
	for _, api := range apis {
		if api.AreProjectFilesRequired() {
			return true
		}
	}
	return false
}
