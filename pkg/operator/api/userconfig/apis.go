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
	Endpoint  *string     `json:"endpoint" yaml:"endpoint"`
	Predictor *Predictor  `json:"predictor" yaml:"predictor"`
	Tracker   *Tracker    `json:"tracker" yaml:"tracker"`
	Compute   *APICompute `json:"compute" yaml:"compute"`
}

type Tracker struct {
	Key       *string   `json:"key" yaml:"key"`
	ModelType ModelType `json:"model_type" yaml:"model_type"`
}

type Predictor struct {
	Type         PredictorType          `json:"type" yaml:"type"`
	Path         string                 `json:"path" yaml:"path"`
	Model        *string                `json:"model" yaml:"model"`
	PythonPath   *string                `json:"python_path" yaml:"python_path"`
	Config       map[string]interface{} `json:"config" yaml:"config"`
	Env          map[string]string      `json:"env" yaml:"env"`
	SignatureKey *string                `json:"signature_key" yaml:"signature_key"`
}

var predictorValidation = &cr.StructFieldValidation{
	StructField: "Predictor",
	StructValidation: &cr.StructValidation{
		Required: true,
		StructFieldValidations: []*cr.StructFieldValidation{
			{
				StructField: "Type",
				StringValidation: &cr.StringValidation{
					Required:      true,
					AllowedValues: PredictorTypeStrings(),
				},
				Parser: func(str string) (interface{}, error) {
					return PredictorTypeFromString(str), nil
				},
			},
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
			{
				StructField: "PythonPath",
				StringPtrValidation: &cr.StringPtrValidation{
					AllowEmpty: true,
					Validator:  ensurePythonPathSuffix,
				},
			},
			{
				StructField: "Config",
				InterfaceMapValidation: &cr.InterfaceMapValidation{
					StringKeysOnly: true,
					AllowEmpty:     true,
					Default:        map[string]interface{}{},
				},
			},
			{
				StructField: "Env",
				StringMapValidation: &cr.StringMapValidation{
					Default:    map[string]string{},
					AllowEmpty: true,
				},
			},
			{
				StructField:         "SignatureKey",
				StringPtrValidation: &cr.StringPtrValidation{},
			},
		},
	},
}

func ensurePythonPathSuffix(path string) (string, error) {
	return s.EnsureSuffix(path, "/"), nil
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
		predictorValidation,
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

	sb.WriteString(fmt.Sprintf("%s:\n", PredictorKey))
	sb.WriteString(s.Indent(api.Predictor.UserConfigStr(), "  "))

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

func (predictor *Predictor) UserConfigStr() string {
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("%s: %s\n", TypeKey, predictor.Type))
	sb.WriteString(fmt.Sprintf("%s: %s\n", PathKey, predictor.Path))
	if predictor.Model != nil {
		sb.WriteString(fmt.Sprintf("%s: %s\n", ModelKey, *predictor.Model))
	}
	if predictor.SignatureKey != nil {
		sb.WriteString(fmt.Sprintf("%s: %s\n", SignatureKeyKey, *predictor.SignatureKey))
	}
	if predictor.PythonPath != nil {
		sb.WriteString(fmt.Sprintf("%s: %s\n", PythonPathKey, *predictor.PythonPath))
	}
	if len(predictor.Config) > 0 {
		sb.WriteString(fmt.Sprintf("%s:\n", ConfigKey))
		d, _ := yaml.Marshal(&predictor.Config)
		sb.WriteString(s.Indent(string(d), "  "))
	}
	if len(predictor.Env) > 0 {
		sb.WriteString(fmt.Sprintf("%s:\n", EnvKey))
		d, _ := yaml.Marshal(&predictor.Env)
		sb.WriteString(s.Indent(string(d), "  "))
	}
	return sb.String()
}

func (predictor *Predictor) Validate(projectFileMap map[string][]byte) error {
	switch predictor.Type {
	case PythonPredictorType:
		if err := predictor.PythonValidate(); err != nil {
			return err
		}
	case TensorFlowPredictorType:
		if err := predictor.TensorFlowValidate(); err != nil {
			return err
		}
	case ONNXPredictorType:
		if err := predictor.ONNXValidate(); err != nil {
			return err
		}
	}

	if _, ok := projectFileMap[predictor.Path]; !ok {
		return errors.Wrap(ErrorImplDoesNotExist(predictor.Path), PathKey)
	}

	if predictor.PythonPath != nil {
		if err := ValidatePythonPath(*predictor.PythonPath, projectFileMap); err != nil {
			return err
		}
	}

	return nil
}

func (predictor *Predictor) TensorFlowValidate() error {
	if predictor.Model == nil {
		return ErrorFieldMustBeDefinedForPredictorType(ModelKey, TensorFlowPredictorType)
	}

	model := *predictor.Model

	awsClient, err := aws.NewFromS3Path(model, false)
	if err != nil {
		return err
	}
	if strings.HasSuffix(model, ".zip") {
		if ok, err := awsClient.IsS3PathFile(model); err != nil || !ok {
			return errors.Wrap(ErrorExternalNotFound(model), ModelKey)
		}
	} else {
		path, err := GetTFServingExportFromS3Path(model, awsClient)
		if path == "" || err != nil {
			return errors.Wrap(ErrorInvalidTensorFlowDir(model), ModelKey)
		}
		predictor.Model = pointer.String(path)
	}

	return nil
}

func (predictor *Predictor) ONNXValidate() error {
	if predictor.Model == nil {
		return ErrorFieldMustBeDefinedForPredictorType(ModelKey, ONNXPredictorType)
	}

	model := *predictor.Model

	awsClient, err := aws.NewFromS3Path(model, false)
	if err != nil {
		return err
	}
	if ok, err := awsClient.IsS3PathFile(model); err != nil || !ok {
		return errors.Wrap(ErrorExternalNotFound(model), ModelKey)
	}

	if predictor.SignatureKey != nil {
		return ErrorFieldNotSupportedByPredictorType(SignatureKeyKey, ONNXPredictorType)
	}

	return nil
}

func (predictor *Predictor) PythonValidate() error {
	if predictor.SignatureKey != nil {
		return ErrorFieldNotSupportedByPredictorType(SignatureKeyKey, PythonPredictorType)
	}

	if predictor.Model != nil {
		return ErrorFieldNotSupportedByPredictorType(ModelKey, PythonPredictorType)
	}

	return nil
}

func (api *API) Validate(deploymentName string, projectFileMap map[string][]byte) error {
	if api.Endpoint == nil {
		api.Endpoint = pointer.String("/" + deploymentName + "/" + api.Name)
	}

	if err := api.Predictor.Validate(projectFileMap); err != nil {
		return errors.Wrap(err, Identify(api), PredictorKey)
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
