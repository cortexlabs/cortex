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
	Python     *Python     `json:"python" yaml:"python"`
	Tracker    *Tracker    `json:"tracker" yaml:"tracker"`
	Compute    *APICompute `json:"compute" yaml:"compute"`
}

type Tracker struct {
	Key       *string   `json:"key" yaml:"key"`
	ModelType ModelType `json:"model_type" yaml:"model_type"`
}

var configValidation = &cr.StructFieldValidation{
	StructField: "Config",
	InterfaceMapValidation: &cr.InterfaceMapValidation{
		StringKeysOnly: true,
		AllowEmpty:     true,
		Default:        map[string]interface{}{},
	},
}

var envValidation = &cr.StructFieldValidation{
	StructField: "Env",
	StringMapValidation: &cr.StringMapValidation{
		Default: map[string]string{},
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
	Model        string                 `json:"model" yaml:"model"`
	Predictor    string                 `json:"predictor" yaml:"predictor"`
	SignatureKey *string                `json:"signature_key" yaml:"signature_key"`
	PythonPath   *string                `json:"python_path" yaml:"python_path"`
	Config       map[string]interface{} `json:"config" yaml:"config"`
	Env          map[string]string      `json:"env" yaml:"env"`
}

type ONNX struct {
	Model      string                 `json:"model" yaml:"model"`
	Predictor  string                 `json:"predictor" yaml:"predictor"`
	PythonPath *string                `json:"python_path" yaml:"python_path"`
	Config     map[string]interface{} `json:"config" yaml:"config"`
	Env        map[string]string      `json:"env" yaml:"env"`
}

type Python struct {
	Predictor  string                 `json:"predictor" yaml:"predictor"`
	PythonPath *string                `json:"python_path" yaml:"python_path"`
	Config     map[string]interface{} `json:"config" yaml:"config"`
	Env        map[string]string      `json:"env" yaml:"env"`
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
						StructField: "Predictor",
						StringValidation: &cr.StringValidation{
							Required: true,
						},
					},
					{
						StructField:         "SignatureKey",
						StringPtrValidation: &cr.StringPtrValidation{},
					},
					pythonPathValidation,
					configValidation,
					envValidation,
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
						StructField: "Predictor",
						StringValidation: &cr.StringValidation{
							Required: true,
						},
					},
					pythonPathValidation,
					configValidation,
					envValidation,
				},
			},
		},
		{
			StructField: "Python",
			StructValidation: &cr.StructValidation{
				DefaultNil: true,
				StructFieldValidations: []*cr.StructFieldValidation{
					{
						StructField: "Predictor",
						StringValidation: &cr.StringValidation{
							Required: true,
						},
					},
					pythonPathValidation,
					configValidation,
					envValidation,
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
	if _, ok := projectFileMap[tf.Predictor]; !ok {
		return errors.Wrap(ErrorImplDoesNotExist(tf.Predictor), TensorFlowKey, PredictorKey)
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
	sb.WriteString(fmt.Sprintf("%s: %s\n", PredictorKey, tf.Predictor))
	if tf.PythonPath != nil {
		sb.WriteString(fmt.Sprintf("%s: %s\n", PythonPathKey, *tf.PythonPath))
	}
	if tf.SignatureKey != nil {
		sb.WriteString(fmt.Sprintf("%s: %s\n", SignatureKeyKey, *tf.SignatureKey))
	}
	if len(tf.Config) > 0 {
		sb.WriteString(fmt.Sprintf("%s:\n", ConfigKey))
		d, _ := yaml.Marshal(&tf.Config)
		sb.WriteString(s.Indent(string(d), "  "))
	}
	if len(tf.Env) > 0 {
		sb.WriteString(fmt.Sprintf("%s:\n", EnvKey))
		d, _ := yaml.Marshal(&tf.Env)
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
	if _, ok := projectFileMap[onnx.Predictor]; !ok {
		return errors.Wrap(ErrorImplDoesNotExist(onnx.Predictor), ONNXKey, PredictorKey)
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
	sb.WriteString(fmt.Sprintf("%s: %s\n", PredictorKey, onnx.Predictor))

	if onnx.PythonPath != nil {
		sb.WriteString(fmt.Sprintf("%s: %s\n", PythonPathKey, *onnx.PythonPath))
	}
	if len(onnx.Config) > 0 {
		sb.WriteString(fmt.Sprintf("%s:\n", ConfigKey))
		d, _ := yaml.Marshal(&onnx.Config)
		sb.WriteString(s.Indent(string(d), "  "))
	}
	if len(onnx.Env) > 0 {
		sb.WriteString(fmt.Sprintf("%s:\n", EnvKey))
		d, _ := yaml.Marshal(&onnx.Env)
		sb.WriteString(s.Indent(string(d), "  "))
	}
	return sb.String()
}

func (python *Python) Validate(projectFileMap map[string][]byte) error {
	if _, ok := projectFileMap[python.Predictor]; !ok {
		return errors.Wrap(ErrorImplDoesNotExist(python.Predictor), PythonKey, PredictorKey)
	}
	if python.PythonPath != nil {
		if err := ValidatePythonPath(*python.PythonPath, projectFileMap); err != nil {
			return errors.Wrap(err, PythonKey)
		}
	}
	return nil
}

func (python *Python) UserConfigStr() string {
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("%s: %s\n", PredictorKey, python.Predictor))
	if python.PythonPath != nil {
		sb.WriteString(fmt.Sprintf("%s: %s\n", PythonPathKey, *python.PythonPath))
	}
	if len(python.Config) > 0 {
		sb.WriteString(fmt.Sprintf("%s:\n", ConfigKey))
		d, _ := yaml.Marshal(&python.Config)
		sb.WriteString(s.Indent(string(d), "  "))
	}
	if len(python.Env) > 0 {
		sb.WriteString(fmt.Sprintf("%s:\n", EnvKey))
		d, _ := yaml.Marshal(&python.Env)
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
	if api.Python != nil {
		specifiedModelFormats = append(specifiedModelFormats, PythonKey)
	}

	if len(specifiedModelFormats) == 0 {
		return ErrorSpecifyOneModelFormatFoundNone(TensorFlowKey, ONNXKey, PythonKey)
	} else if len(specifiedModelFormats) > 1 {
		return ErrorSpecifyOneModelFormatFoundMultiple(specifiedModelFormats, TensorFlowKey, ONNXKey, PythonKey)
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
