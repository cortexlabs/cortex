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

	"github.com/cortexlabs/yaml"
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
	Tensorflow *Tensorflow            `json:"tensorflow" yaml:"tensorflow"`
	ONNX       *ONNX                  `json:"onnx" yaml:"onnx"`
	Python     *Python                `json:"python" yaml:"python"`
	Tracker    *Tracker               `json:"tracker" yaml:"tracker"`
	Compute    *APICompute            `json:"compute" yaml:"compute"`
	Tags       Tags                   `json:"tags" yaml:"tags"`
	Metadata   map[string]interface{} `json:"metadata" yaml:"metadata"`
}

type Tracker struct {
	Key       *string   `json:"key" yaml:"key"`
	ModelType ModelType `json:"model_type" yaml:"model_type"`
}

type Tensorflow struct {
	Model          string  `json:"model" yaml:"model"`
	RequestHandler *string `json:"request_handler" yaml:"request_handler"`
	SignatureKey   *string `json:"signature_key" yaml:"signature_key"`
}

type ONNX struct {
	Model          string  `json:"model" yaml:"model"`
	RequestHandler *string `json:"request_handler" yaml:"request_handler"`
}

type Python struct {
	Inference string `json:"inference" yaml:"inference"`
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
			StructField: "Tensorflow",
			StructValidation: &cr.StructValidation{
				DefaultNil: true,
				StructFieldValidations: []*cr.StructFieldValidation{
					{
						StructField: "Model",
						StringValidation: &cr.StringValidation{
							Required:  true,
							Validator: cr.S3PathValidator(),
						},
					},
					{
						StructField:         "RequestHandler",
						StringPtrValidation: &cr.StringPtrValidation{},
					},
					{
						StructField:         "SignatureKey",
						StringPtrValidation: &cr.StringPtrValidation{},
					},
				},
			},
		},
		{
			StructField: "ONNX",
			StructValidation: &cr.StructValidation{
				DefaultNil: true,
				StructFieldValidations: []*cr.StructFieldValidation{
					{
						StructField: "Model",
						StringValidation: &cr.StringValidation{
							Required:  true,
							Validator: cr.S3PathValidator(),
						},
					},
					{
						StructField:         "RequestHandler",
						StringPtrValidation: &cr.StringPtrValidation{},
					},
				},
			},
		},
		{
			StructField: "Python",
			StructValidation: &cr.StructValidation{
				DefaultNil: true,
				StructFieldValidations: []*cr.StructFieldValidation{
					{
						StructField: "Inference",
						StringValidation: &cr.StringValidation{
							Required: true,
						},
					},
				},
			},
		},
		{
			StructField: "Metadata",
			InterfaceMapValidation: &cr.InterfaceMapValidation{
				StringKeysOnly: true,
			},
		},
		apiComputeFieldValidation,
		tagsFieldValidation,
		typeFieldValidation,
	},
}

// IsValidTensorFlowS3Directory checks that the path contains a valid S3 directory for TensorFlow models
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
	if api.Tensorflow != nil {
		sb.WriteString(fmt.Sprintf("%s:\n", TensorflowKey))
		sb.WriteString(s.Indent(api.Tensorflow.UserConfigStr(), "  "))
	}
	if api.ONNX != nil {
		sb.WriteString(fmt.Sprintf("%s:\n", ONNXKey))
		sb.WriteString(s.Indent(api.ONNX.UserConfigStr(), "  "))
	}
	if api.Python != nil {
		sb.WriteString(fmt.Sprintf("%s:\n", PythonKey))
		sb.WriteString(s.Indent(api.Python.UserConfigStr(), "  "))
	}
	if api.Compute != nil {
		sb.WriteString(fmt.Sprintf("%s:\n", ComputeKey))
		sb.WriteString(s.Indent(api.Compute.UserConfigStr(), "  "))
	}
	if api.Tracker != nil {
		sb.WriteString(fmt.Sprintf("%s:\n", TrackerKey))
		sb.WriteString(s.Indent(api.Tracker.UserConfigStr(), "  "))
	}
	if len(api.Metadata) > 0 {
		sb.WriteString(fmt.Sprintf("%s:\n", MetadataKey))
		d, _ := yaml.Marshal(&api.Metadata)
		sb.WriteString(s.Indent(string(d), "  "))
	}
	return sb.String()
}

func (tracker *Tracker) UserConfigStr() string {
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("%s: %s\n", ModelTypeKey, tracker.ModelType.String()))
	if tracker.Key != nil {
		sb.WriteString(fmt.Sprintf("%s: %s\n", KeyKey, *tracker.Key))
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

func (tf *Tensorflow) Validate(projectFileMap map[string][]byte) error {
	awsClient, err := aws.NewFromS3Path(tf.Model, false)
	if err != nil {
		return err
	}
	if strings.HasSuffix(tf.Model, ".zip") {
		if ok, err := awsClient.IsS3PathFile(tf.Model); err != nil || !ok {
			return errors.Wrap(ErrorExternalNotFound(tf.Model), TensorflowKey, ModelKey)
		}
	} else {
		path, err := GetTFServingExportFromS3Path(tf.Model, awsClient)
		if path == "" || err != nil {
			return errors.Wrap(ErrorInvalidTensorFlowDir(tf.Model), TensorflowKey, ModelKey)
		}
		tf.Model = path
	}
	if tf.RequestHandler != nil {
		if _, ok := projectFileMap[*tf.RequestHandler]; !ok {
			return errors.Wrap(ErrorImplDoesNotExist(*tf.RequestHandler), TensorflowKey, RequestHandlerKey)
		}
	}
	return nil
}

func (tf *Tensorflow) UserConfigStr() string {
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("%s: %s\n", ModelKey, tf.Model))
	if tf.RequestHandler != nil {
		sb.WriteString(fmt.Sprintf("%s: %s\n", RequestHandlerKey, *tf.RequestHandler))
	}
	if tf.SignatureKey != nil {
		sb.WriteString(fmt.Sprintf("%s: %s\n", SignatureKeyKey, *tf.SignatureKey))
	}
	return sb.String()
}

func (onnx *ONNX) Validate(projectFileMap map[string][]byte) error {
	awsClient, err := aws.NewFromS3Path(onnx.Model, false)
	if err != nil {
		return err
	}
	if ok, err := awsClient.IsS3PathFile(onnx.Model); err != nil || !ok {
		return errors.Wrap(ErrorExternalNotFound(onnx.Model), ONNXKey, ModelKey)
	}
	if onnx.RequestHandler != nil {
		if _, ok := projectFileMap[*onnx.RequestHandler]; !ok {
			return errors.Wrap(ErrorImplDoesNotExist(*onnx.RequestHandler), ONNXKey, RequestHandlerKey)
		}
	}
	return nil
}

func (onnx *ONNX) UserConfigStr() string {
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("%s: %s\n", ModelKey, onnx.Model))
	if onnx.RequestHandler != nil {
		sb.WriteString(fmt.Sprintf("%s: %s\n", RequestHandlerKey, *onnx.RequestHandler))
	}
	return sb.String()
}

func (python *Python) Validate(projectFileMap map[string][]byte) error {
	if _, ok := projectFileMap[python.Inference]; !ok {
		return errors.Wrap(ErrorImplDoesNotExist(python.Inference), PythonKey, RequestHandlerKey)
	}
	return nil
}

func (python *Python) UserConfigStr() string {
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("%s: %s\n", InferenceKey, python.Inference))
	return sb.String()
}

func (api *API) Validate(projectFileMap map[string][]byte) error {
	specifiedModelFormats := []string{}
	if api.Tensorflow != nil {
		specifiedModelFormats = append(specifiedModelFormats, TensorflowKey)
	}
	if api.ONNX != nil {
		specifiedModelFormats = append(specifiedModelFormats, ONNXKey)
	}
	if api.Python != nil {
		specifiedModelFormats = append(specifiedModelFormats, PythonKey)
	}

	if len(specifiedModelFormats) == 0 {
		return ErrorSpecifyOneModelFormatFoundNone(TensorflowKey, ONNXKey, PythonKey)
	} else if len(specifiedModelFormats) > 1 {
		return ErrorSpecifyOneModelFormatFoundMultiple(specifiedModelFormats, TensorflowKey, ONNXKey, PythonKey)
	} else {
		switch specifiedModelFormats[0] {
		case TensorflowKey:
			if err := api.Tensorflow.Validate(projectFileMap); err != nil {
				return errors.Wrap(err, Identify(api))
			}
		case ONNXKey:
			if err := api.ONNX.Validate(projectFileMap); err != nil {
				return errors.Wrap(err, Identify(api))
			}
		case PythonKey:
			if err := api.Python.Validate(projectFileMap); err != nil {
				return errors.Wrap(err, Identify(api))
			}
		}
	}

	if err := api.Compute.Validate(); err != nil {
		return errors.Wrap(err, Identify(api), ComputeKey)
	}

	return nil
}

func (api *API) AreProjectFilesRequired() bool {
	switch {
	case api.Tensorflow != nil && api.Tensorflow.RequestHandler != nil:
		return true
	case api.ONNX != nil && api.ONNX.RequestHandler != nil:
		return true
	case api.Python != nil:
		return true
	}
	return false
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
