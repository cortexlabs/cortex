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
	"github.com/cortexlabs/cortex/pkg/lib/pointer"
	s "github.com/cortexlabs/cortex/pkg/lib/strings"
	"github.com/cortexlabs/cortex/pkg/lib/urls"
	"github.com/cortexlabs/cortex/pkg/operator/api/resource"
	"github.com/cortexlabs/yaml"
)

type APIs []*API

type API struct {
	ResourceFields
	Endpoint   *string     `json:"endpoint" yaml:"endpoint"`
	TensorFlow *TensorFlow `json:"tensorflow" yaml:"tensorflow"`
	ONNX       *ONNX       `json:"onnx" yaml:"onnx"`
	Predictor  *Predictor  `json:"predictor" yaml:"predictor"`
	Tracker    *Tracker    `json:"tracker" yaml:"tracker"`
	Compute    *APICompute `json:"compute" yaml:"compute"`
}

type Tracker struct {
	Key       *string   `json:"key" yaml:"key"`
	ModelType ModelType `json:"model_type" yaml:"model_type"`
}

var metadataValidation = &cr.StructFieldValidation{
	StructField: "Metadata",
	InterfaceMapValidation: &cr.InterfaceMapValidation{
		StringKeysOnly: true,
		AllowEmpty:     true,
		Default:        map[string]interface{}{},
	},
}

func ensurePythonPathSuffix(path string) (string, error) {
	return s.EnsureSuffix(path, "/"), nil
}

var pythonPathValidation = &cr.StructFieldValidation{
	StructField: "PythonPath",
	StringPtrValidation: &cr.StringPtrValidation{
		AllowEmpty: true,
		Validator:  ensurePythonPathSuffix,
	},
}

type TensorFlow struct {
	Model          string                 `json:"model" yaml:"model"`
	RequestHandler *string                `json:"request_handler" yaml:"request_handler"`
	SignatureKey   *string                `json:"signature_key" yaml:"signature_key"`
	PythonPath     *string                `json:"python_path" yaml:"python_path"`
	Metadata       map[string]interface{} `json:"metadata" yaml:"metadata"`
}

type ONNX struct {
	Model          string                 `json:"model" yaml:"model"`
	RequestHandler *string                `json:"request_handler" yaml:"request_handler"`
	PythonPath     *string                `json:"python_path" yaml:"python_path"`
	Metadata       map[string]interface{} `json:"metadata" yaml:"metadata"`
}

type Predictor struct {
	Path       string                 `json:"path" yaml:"path"`
	Model      *string                `json:"model" yaml:"model"`
	PythonPath *string                `json:"python_path" yaml:"python_path"`
	Metadata   map[string]interface{} `json:"metadata" yaml:"metadata"`
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
			StructField: "Endpoint",
			StringPtrValidation: &cr.StringPtrValidation{
				Validator: urls.ValidateEndpoint,
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
			StructField: "TensorFlow",
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
					pythonPathValidation,
					metadataValidation,
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
					pythonPathValidation,
					metadataValidation,
				},
			},
		},
		{
			StructField: "Predictor",
			StructValidation: &cr.StructValidation{
				DefaultNil: true,
				StructFieldValidations: []*cr.StructFieldValidation{
					{
						StructField: "Path",
						StringValidation: &cr.StringValidation{
							Required: true,
						},
					},
					{
						StructField: "Model",
						StringPtrValidation: &cr.StringPtrValidation{
							Validator: cr.S3PathValidator(),
						},
					},
					pythonPathValidation,
					metadataValidation,
				},
			},
		},
		apiComputeFieldValidation,
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
	prefix = s.EnsureSuffix(prefix, "/")

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

func ValidatePythonPath(path string, projectFileMap map[string][]byte) error {
	validPythonPath := false
	for fileKey := range projectFileMap {
		if strings.HasPrefix(fileKey, path) {
			validPythonPath = true
			break
		}
	}
	if !validPythonPath {
		return errors.Wrap(ErrorImplDoesNotExist(path), PythonPathKey)
	}
	return nil
}

func (api *API) UserConfigStr() string {
	var sb strings.Builder
	sb.WriteString(api.ResourceFields.UserConfigStr())
	sb.WriteString(fmt.Sprintf("%s: %s\n", EndpointKey, *api.Endpoint))

	if api.TensorFlow != nil {
		sb.WriteString(fmt.Sprintf("%s:\n", TensorFlowKey))
		sb.WriteString(s.Indent(api.TensorFlow.UserConfigStr(), "  "))
	}
	if api.ONNX != nil {
		sb.WriteString(fmt.Sprintf("%s:\n", ONNXKey))
		sb.WriteString(s.Indent(api.ONNX.UserConfigStr(), "  "))
	}
	if api.Predictor != nil {
		sb.WriteString(fmt.Sprintf("%s:\n", PredictorKey))
		sb.WriteString(s.Indent(api.Predictor.UserConfigStr(), "  "))
	}
	if api.Compute != nil {
		sb.WriteString(fmt.Sprintf("%s:\n", ComputeKey))
		sb.WriteString(s.Indent(api.Compute.UserConfigStr(), "  "))
	}
	if api.Tracker != nil {
		sb.WriteString(fmt.Sprintf("%s:\n", TrackerKey))
		sb.WriteString(s.Indent(api.Tracker.UserConfigStr(), "  "))
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

func (apis APIs) Validate(deploymentName string, projectFileMap map[string][]byte) error {
	for _, api := range apis {
		if err := api.Validate(deploymentName, projectFileMap); err != nil {
			return err
		}
	}

	endpoints := map[string]string{} // endpoint -> API name
	for _, api := range apis {
		if dupAPIName, ok := endpoints[*api.Endpoint]; ok {
			return ErrorDuplicateEndpoints(*api.Endpoint, dupAPIName, api.Name)
		}
		endpoints[*api.Endpoint] = api.Name
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

func (tf *TensorFlow) Validate(projectFileMap map[string][]byte) error {
	awsClient, err := aws.NewFromS3Path(tf.Model, false)
	if err != nil {
		return err
	}
	if strings.HasSuffix(tf.Model, ".zip") {
		if ok, err := awsClient.IsS3PathFile(tf.Model); err != nil || !ok {
			return errors.Wrap(ErrorExternalNotFound(tf.Model), TensorFlowKey, ModelKey)
		}
	} else {
		path, err := GetTFServingExportFromS3Path(tf.Model, awsClient)
		if path == "" || err != nil {
			return errors.Wrap(ErrorInvalidTensorFlowDir(tf.Model), TensorFlowKey, ModelKey)
		}
		tf.Model = path
	}
	if tf.RequestHandler != nil {
		if _, ok := projectFileMap[*tf.RequestHandler]; !ok {
			return errors.Wrap(ErrorImplDoesNotExist(*tf.RequestHandler), TensorFlowKey, RequestHandlerKey)
		}
	}
	if tf.PythonPath != nil {
		if err := ValidatePythonPath(*tf.PythonPath, projectFileMap); err != nil {
			return errors.Wrap(err, TensorFlowKey)
		}
	}
	return nil
}

func (tf *TensorFlow) UserConfigStr() string {
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("%s: %s\n", ModelKey, tf.Model))
	if tf.PythonPath != nil {
		sb.WriteString(fmt.Sprintf("%s: %s\n", PythonPathKey, *tf.PythonPath))
	}
	if tf.RequestHandler != nil {
		sb.WriteString(fmt.Sprintf("%s: %s\n", RequestHandlerKey, *tf.RequestHandler))
	}
	if tf.SignatureKey != nil {
		sb.WriteString(fmt.Sprintf("%s: %s\n", SignatureKeyKey, *tf.SignatureKey))
	}
	if len(tf.Metadata) > 0 {
		sb.WriteString(fmt.Sprintf("%s:\n", MetadataKey))
		d, _ := yaml.Marshal(&tf.Metadata)
		sb.WriteString(s.Indent(string(d), "  "))
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
	if onnx.PythonPath != nil {
		if err := ValidatePythonPath(*onnx.PythonPath, projectFileMap); err != nil {
			return errors.Wrap(err, ONNXKey)
		}
	}
	return nil
}

func (onnx *ONNX) UserConfigStr() string {
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("%s: %s\n", ModelKey, onnx.Model))
	if onnx.PythonPath != nil {
		sb.WriteString(fmt.Sprintf("%s: %s\n", PythonPathKey, *onnx.PythonPath))
	}
	if onnx.RequestHandler != nil {
		sb.WriteString(fmt.Sprintf("%s: %s\n", RequestHandlerKey, *onnx.RequestHandler))
	}
	if len(onnx.Metadata) > 0 {
		sb.WriteString(fmt.Sprintf("%s:\n", MetadataKey))
		d, _ := yaml.Marshal(&onnx.Metadata)
		sb.WriteString(s.Indent(string(d), "  "))
	}
	return sb.String()
}

func (predictor *Predictor) Validate(projectFileMap map[string][]byte) error {
	if _, ok := projectFileMap[predictor.Path]; !ok {
		return errors.Wrap(ErrorImplDoesNotExist(predictor.Path), PredictorKey, PathKey)
	}
	if predictor.Model != nil {
		var err error
		awsClient, err := aws.NewFromS3Path(*predictor.Model, false)
		if err != nil {
			return err
		}

		isFile := false
		if isFile, err = awsClient.IsS3PathFile(*predictor.Model); err != nil {
			return errors.Wrap(err, PredictorKey, ModelKey)
		}

		if !isFile {
			isDir := false
			modelPath := s.EnsureSuffix(*predictor.Model, "/")

			if isDir, err = awsClient.IsS3PathDir(modelPath); err != nil {
				return errors.Wrap(err, PredictorKey, ModelKey)
			}

			if !isDir {
				return errors.Wrap(ErrorExternalNotFound(*predictor.Model), PredictorKey, ModelKey)
			}

			predictor.Model = pointer.String(modelPath)
		}
	}
	if predictor.PythonPath != nil {
		if err := ValidatePythonPath(*predictor.PythonPath, projectFileMap); err != nil {
			return errors.Wrap(err, PredictorKey)
		}
	}
	return nil
}

func (predictor *Predictor) UserConfigStr() string {
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("%s: %s\n", PathKey, predictor.Path))
	if predictor.PythonPath != nil {
		sb.WriteString(fmt.Sprintf("%s: %s\n", PythonPathKey, *predictor.PythonPath))
	}
	if len(predictor.Metadata) > 0 {
		sb.WriteString(fmt.Sprintf("%s:\n", MetadataKey))
		d, _ := yaml.Marshal(&predictor.Metadata)
		sb.WriteString(s.Indent(string(d), "  "))
	}
	return sb.String()
}

func (api *API) Validate(deploymentName string, projectFileMap map[string][]byte) error {
	if api.Endpoint == nil {
		api.Endpoint = pointer.String("/" + deploymentName + "/" + api.Name)
	}

	specifiedModelFormats := []string{}
	if api.TensorFlow != nil {
		specifiedModelFormats = append(specifiedModelFormats, TensorFlowKey)
	}
	if api.ONNX != nil {
		specifiedModelFormats = append(specifiedModelFormats, ONNXKey)
	}
	if api.Predictor != nil {
		specifiedModelFormats = append(specifiedModelFormats, PredictorKey)
	}

	if len(specifiedModelFormats) == 0 {
		return ErrorSpecifyOneModelFormatFoundNone(TensorFlowKey, ONNXKey, PredictorKey)
	} else if len(specifiedModelFormats) > 1 {
		return ErrorSpecifyOneModelFormatFoundMultiple(specifiedModelFormats, TensorFlowKey, ONNXKey, PredictorKey)
	} else {
		switch specifiedModelFormats[0] {
		case TensorFlowKey:
			if err := api.TensorFlow.Validate(projectFileMap); err != nil {
				return errors.Wrap(err, Identify(api))
			}
		case ONNXKey:
			if err := api.ONNX.Validate(projectFileMap); err != nil {
				return errors.Wrap(err, Identify(api))
			}
		case PredictorKey:
			if err := api.Predictor.Validate(projectFileMap); err != nil {
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
	case api.TensorFlow != nil && api.TensorFlow.RequestHandler != nil:
		return true
	case api.ONNX != nil && api.ONNX.RequestHandler != nil:
		return true
	case api.Predictor != nil:
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
