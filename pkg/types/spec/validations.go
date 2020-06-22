/*
Copyright 2020 Cortex Labs, Inc.

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

package spec

import (
	"fmt"
	"math"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/cortexlabs/cortex/pkg/consts"
	"github.com/cortexlabs/cortex/pkg/lib/aws"
	"github.com/cortexlabs/cortex/pkg/lib/cast"
	cr "github.com/cortexlabs/cortex/pkg/lib/configreader"
	"github.com/cortexlabs/cortex/pkg/lib/docker"
	"github.com/cortexlabs/cortex/pkg/lib/errors"
	"github.com/cortexlabs/cortex/pkg/lib/files"
	"github.com/cortexlabs/cortex/pkg/lib/k8s"
	libmath "github.com/cortexlabs/cortex/pkg/lib/math"
	"github.com/cortexlabs/cortex/pkg/lib/pointer"
	"github.com/cortexlabs/cortex/pkg/lib/regex"
	"github.com/cortexlabs/cortex/pkg/lib/sets/strset"
	s "github.com/cortexlabs/cortex/pkg/lib/strings"
	libtime "github.com/cortexlabs/cortex/pkg/lib/time"
	"github.com/cortexlabs/cortex/pkg/types"
	"github.com/cortexlabs/cortex/pkg/types/userconfig"
	kresource "k8s.io/apimachinery/pkg/api/resource"
)

var AutoscalingTickInterval = 10 * time.Second

func apiValidation(provider types.ProviderType, kind userconfig.Kind) *cr.StructValidation {
	structFieldValidations := []*cr.StructFieldValidation{}
	structFieldValidations = append(structFieldValidations, resourceStructValidations...)
	structFieldValidations = append(structFieldValidations,
		&cr.StructFieldValidation{
			StructField: "LocalPort",
			IntPtrValidation: &cr.IntPtrValidation{
				GreaterThan:       pointer.Int(0),
				LessThanOrEqualTo: pointer.Int(math.MaxUint16),
			},
		},
		predictorValidation(),
		monitoringValidation(),
		networkingValidation(),
		computeValidation(provider),
		autoscalingValidation(provider),
		updateStrategyValidation(provider),
	)
	return &cr.StructValidation{
		StructFieldValidations: structFieldValidations,
	}
}

var resourceStructValidations = []*cr.StructFieldValidation{
	{
		StructField: "Name",
		StringValidation: &cr.StringValidation{
			Required:  true,
			DNS1035:   true,
			MaxLength: 42, // k8s adds 21 characters to the pod name, and 63 is the max before it starts to truncate
		},
	},
	{
		StructField: "Kind",
		StringValidation: &cr.StringValidation{
			Required:      true,
			AllowedValues: userconfig.KindStrings(),
		},
		Parser: func(str string) (interface{}, error) {
			return userconfig.KindFromString(str), nil
		},
	},
}

func predictorValidation() *cr.StructFieldValidation {
	return &cr.StructFieldValidation{
		StructField: "Predictor",
		StructValidation: &cr.StructValidation{
			Required: true,
			StructFieldValidations: []*cr.StructFieldValidation{
				{
					StructField: "Type",
					StringValidation: &cr.StringValidation{
						Required:      true,
						AllowedValues: userconfig.PredictorTypeStrings(),
					},
					Parser: func(str string) (interface{}, error) {
						return userconfig.PredictorTypeFromString(str), nil
					},
				},
				{
					StructField: "Path",
					StringValidation: &cr.StringValidation{
						Required: true,
					},
				},
				{
					StructField:         "Model",
					StringPtrValidation: &cr.StringPtrValidation{},
				},
				{
					StructField: "PythonPath",
					StringPtrValidation: &cr.StringPtrValidation{
						AllowEmpty: true,
						Validator: func(path string) (string, error) {
							return s.EnsureSuffix(path, "/"), nil
						},
					},
				},
				{
					StructField: "Image",
					StringValidation: &cr.StringValidation{
						Required:           false,
						AllowEmpty:         true,
						DockerImageOrEmpty: true,
					},
				},
				{
					StructField: "TensorFlowServingImage",
					StringValidation: &cr.StringValidation{
						Required:           false,
						AllowEmpty:         true,
						DockerImageOrEmpty: true,
					},
				},
				{
					StructField: "Config",
					InterfaceMapValidation: &cr.InterfaceMapValidation{
						StringKeysOnly:     true,
						AllowEmpty:         true,
						AllowExplicitNull:  true,
						ConvertNullToEmpty: true,
						Default:            map[string]interface{}{},
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
				multiModelValidation(),
			},
		},
	}
}

func monitoringValidation() *cr.StructFieldValidation {
	return &cr.StructFieldValidation{
		StructField: "Monitoring",
		StructValidation: &cr.StructValidation{
			DefaultNil:        true,
			AllowExplicitNull: true,
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
						AllowedValues: userconfig.ModelTypeStrings(),
					},
					Parser: func(str string) (interface{}, error) {
						return userconfig.ModelTypeFromString(str), nil
					},
				},
			},
		},
	}
}

func networkingValidation() *cr.StructFieldValidation {
	return &cr.StructFieldValidation{
		StructField: "Networking",
		StructValidation: &cr.StructValidation{
			StructFieldValidations: []*cr.StructFieldValidation{
				{
					StructField: "APIGateway",
					StringValidation: &cr.StringValidation{
						AllowedValues: userconfig.APIGatewayTypeStrings(),
						Default:       userconfig.PublicAPIGatewayType.String(),
					},
					Parser: func(str string) (interface{}, error) {
						return userconfig.APIGatewayTypeFromString(str), nil
					},
				},
			},
		},
	}
}

func computeValidation(provider types.ProviderType) *cr.StructFieldValidation {
	cpuDefault := pointer.String("200m")
	if provider == types.LocalProviderType {
		cpuDefault = nil
	}
	return &cr.StructFieldValidation{
		StructField: "Compute",
		StructValidation: &cr.StructValidation{
			StructFieldValidations: []*cr.StructFieldValidation{
				{
					StructField: "CPU",
					StringPtrValidation: &cr.StringPtrValidation{
						Default:           cpuDefault,
						AllowExplicitNull: true,
						CastNumeric:       true,
					},
					Parser: k8s.QuantityParser(&k8s.QuantityValidation{
						GreaterThanOrEqualTo: k8s.QuantityPtr(kresource.MustParse("20m")),
					}),
				},
				{
					StructField: "Mem",
					StringPtrValidation: &cr.StringPtrValidation{
						Default:           nil,
						AllowExplicitNull: true,
					},
					Parser: k8s.QuantityParser(&k8s.QuantityValidation{
						GreaterThanOrEqualTo: k8s.QuantityPtr(kresource.MustParse("20Mi")),
					}),
				},
				{
					StructField: "GPU",
					Int64Validation: &cr.Int64Validation{
						Default:              0,
						GreaterThanOrEqualTo: pointer.Int64(0),
					},
				},
				{
					StructField: "Inf",
					Int64Validation: &cr.Int64Validation{
						Default:              0,
						GreaterThanOrEqualTo: pointer.Int64(0),
					},
				},
			},
		},
	}
}

func autoscalingValidation(provider types.ProviderType) *cr.StructFieldValidation {
	defaultNil := provider == types.LocalProviderType
	allowExplicitNull := provider == types.LocalProviderType
	return &cr.StructFieldValidation{
		StructField: "Autoscaling",
		StructValidation: &cr.StructValidation{
			DefaultNil:        defaultNil,
			AllowExplicitNull: allowExplicitNull,
			StructFieldValidations: []*cr.StructFieldValidation{
				{
					StructField: "MinReplicas",
					Int32Validation: &cr.Int32Validation{
						Default:     1,
						GreaterThan: pointer.Int32(0),
					},
				},
				{
					StructField: "MaxReplicas",
					Int32Validation: &cr.Int32Validation{
						Default:     100,
						GreaterThan: pointer.Int32(0),
					},
				},
				{
					StructField:  "InitReplicas",
					DefaultField: "MinReplicas",
					Int32Validation: &cr.Int32Validation{
						GreaterThan: pointer.Int32(0),
					},
				},
				{
					StructField: "WorkersPerReplica",
					Int32Validation: &cr.Int32Validation{
						Default:              1,
						GreaterThanOrEqualTo: pointer.Int32(1),
						LessThanOrEqualTo:    pointer.Int32(20),
					},
				},
				{
					StructField: "ThreadsPerWorker",
					Int32Validation: &cr.Int32Validation{
						Default:              1,
						GreaterThanOrEqualTo: pointer.Int32(1),
					},
				},
				{
					StructField: "TargetReplicaConcurrency",
					Float64PtrValidation: &cr.Float64PtrValidation{
						GreaterThan: pointer.Float64(0),
					},
				},
				{
					StructField: "MaxReplicaConcurrency",
					Int64Validation: &cr.Int64Validation{
						Default:           1024,
						GreaterThan:       pointer.Int64(0),
						LessThanOrEqualTo: pointer.Int64(math.MaxUint16),
					},
				},
				{
					StructField: "Window",
					StringValidation: &cr.StringValidation{
						Default: "60s",
					},
					Parser: cr.DurationParser(&cr.DurationValidation{
						GreaterThanOrEqualTo: &AutoscalingTickInterval,
						MultipleOf:           &AutoscalingTickInterval,
					}),
				},
				{
					StructField: "DownscaleStabilizationPeriod",
					StringValidation: &cr.StringValidation{
						Default: "5m",
					},
					Parser: cr.DurationParser(&cr.DurationValidation{
						GreaterThanOrEqualTo: pointer.Duration(libtime.MustParseDuration("0s")),
					}),
				},
				{
					StructField: "UpscaleStabilizationPeriod",
					StringValidation: &cr.StringValidation{
						Default: "1m",
					},
					Parser: cr.DurationParser(&cr.DurationValidation{
						GreaterThanOrEqualTo: pointer.Duration(libtime.MustParseDuration("0s")),
					}),
				},
				{
					StructField: "MaxDownscaleFactor",
					Float64Validation: &cr.Float64Validation{
						Default:              0.75,
						GreaterThanOrEqualTo: pointer.Float64(0),
						LessThan:             pointer.Float64(1),
					},
				},
				{
					StructField: "MaxUpscaleFactor",
					Float64Validation: &cr.Float64Validation{
						Default:     1.5,
						GreaterThan: pointer.Float64(1),
					},
				},
				{
					StructField: "DownscaleTolerance",
					Float64Validation: &cr.Float64Validation{
						Default:              0.05,
						GreaterThanOrEqualTo: pointer.Float64(0),
						LessThan:             pointer.Float64(1),
					},
				},
				{
					StructField: "UpscaleTolerance",
					Float64Validation: &cr.Float64Validation{
						Default:              0.05,
						GreaterThanOrEqualTo: pointer.Float64(0),
					},
				},
			},
		},
	}
}

func updateStrategyValidation(provider types.ProviderType) *cr.StructFieldValidation {
	defaultNil := provider == types.LocalProviderType
	allowExplicitNull := provider == types.LocalProviderType
	return &cr.StructFieldValidation{
		StructField: "UpdateStrategy",
		StructValidation: &cr.StructValidation{
			DefaultNil:        defaultNil,
			AllowExplicitNull: allowExplicitNull,
			StructFieldValidations: []*cr.StructFieldValidation{
				{
					StructField: "MaxSurge",
					StringValidation: &cr.StringValidation{
						Default:   "25%",
						CastInt:   true,
						Validator: surgeOrUnavailableValidator,
					},
				},
				{
					StructField: "MaxUnavailable",
					StringValidation: &cr.StringValidation{
						Default:   "25%",
						CastInt:   true,
						Validator: surgeOrUnavailableValidator,
					},
				},
			},
		},
	}
}

func multiModelValidation() *cr.StructFieldValidation {
	return &cr.StructFieldValidation{
		StructField: "Models",
		StructListValidation: &cr.StructListValidation{
			Required:         false,
			TreatNullAsEmpty: true,
			StructValidation: &cr.StructValidation{
				StructFieldValidations: []*cr.StructFieldValidation{
					{
						StructField: "Name",
						StringValidation: &cr.StringValidation{
							Required:                   true,
							AllowEmpty:                 false,
							DisallowedValues:           []string{consts.SingleModelName},
							AlphaNumericDashUnderscore: true,
						},
					},
					{
						StructField: "Model",
						StringValidation: &cr.StringValidation{
							Required:   true,
							AllowEmpty: false,
						},
					},
					{
						StructField: "SignatureKey",
						StringPtrValidation: &cr.StringPtrValidation{
							Required:   false,
							AllowEmpty: false,
						},
					},
				},
			},
		},
	}
}

func surgeOrUnavailableValidator(str string) (string, error) {
	if strings.HasSuffix(str, "%") {
		parsed, ok := s.ParseInt32(strings.TrimSuffix(str, "%"))
		if !ok {
			return "", ErrorInvalidSurgeOrUnavailable(str)
		}
		if parsed < 0 || parsed > 100 {
			return "", ErrorInvalidSurgeOrUnavailable(str)
		}
	} else {
		parsed, ok := s.ParseInt32(str)
		if !ok {
			return "", ErrorInvalidSurgeOrUnavailable(str)
		}
		if parsed < 0 {
			return "", ErrorInvalidSurgeOrUnavailable(str)
		}
	}

	return str, nil
}

type kindStruct struct {
	Name string          `json:"name" yaml:"name"`
	Kind userconfig.Kind `json:"kind" yaml:"kind"`
}

var kindStructValidation = cr.StructValidation{
	AllowExtraFields:       true,
	StructFieldValidations: resourceStructValidations,
}

func ExtractAPIConfigs(configBytes []byte, provider types.ProviderType, projectFiles ProjectFiles, filePath string) ([]userconfig.API, error) {
	var err error

	configData, err := cr.ReadYAMLBytes(configBytes)
	if err != nil {
		return nil, errors.Wrap(err, filePath)
	}

	configDataSlice, ok := cast.InterfaceToStrInterfaceMapSlice(configData)
	if !ok {
		return nil, errors.Wrap(ErrorMalformedConfig(), filePath)
	}

	apis := make([]userconfig.API, len(configDataSlice))
	for i, data := range configDataSlice {
		api := userconfig.API{}
		var kindStruct kindStruct
		errs := cr.Struct(&kindStruct, data, &kindStructValidation)
		if errors.HasError(errs) {
			name, _ := data[userconfig.NameKey].(string)
			err = errors.Wrap(errors.FirstError(errs...), userconfig.IdentifyAPI(filePath, name, i))
			return nil, errors.Append(err, fmt.Sprintf("\n\napi configuration schema can be found here: https://docs.cortex.dev/v/%s/deployments/api-configuration", consts.CortexVersionMinor))
		}

		errs = cr.Struct(&api, data, apiValidation(provider, kindStruct.Kind))

		if errors.HasError(errs) {
			name, _ := data[userconfig.NameKey].(string)
			err = errors.Wrap(errors.FirstError(errs...), userconfig.IdentifyAPI(filePath, name, i))
			return nil, errors.Append(err, fmt.Sprintf("\n\napi configuration schema can be found here: https://docs.cortex.dev/v/%s/deployments/api-configuration", consts.CortexVersionMinor))
		}
		api.Index = i
		api.FilePath = filePath

		api.ApplyDefaultDockerPaths()

		apis[i] = api
	}

	return apis, nil
}

func ValidateAPI(
	api *userconfig.API,
	projectFiles ProjectFiles,
	providerType types.ProviderType,
	awsClient *aws.Client,
) error {
	if providerType == types.AWSProviderType && api.Endpoint == nil {
		api.Endpoint = pointer.String("/" + api.Name)
	}

	if err := validatePredictor(api, projectFiles, providerType, awsClient); err != nil {
		return errors.Wrap(err, api.Identify(), userconfig.PredictorKey)
	}

	if api.Autoscaling != nil { // should only be nil for local provider
		if err := validateAutoscaling(api); err != nil {
			return errors.Wrap(err, api.Identify(), userconfig.AutoscalingKey)
		}
	}

	if err := validateCompute(api, providerType); err != nil {
		return errors.Wrap(err, api.Identify(), userconfig.ComputeKey)
	}

	if api.UpdateStrategy != nil { // should only be nil for local provider
		if err := validateUpdateStrategy(api.UpdateStrategy); err != nil {
			return errors.Wrap(err, api.Identify(), userconfig.UpdateStrategyKey)
		}
	}

	return nil
}

func validatePredictor(api *userconfig.API, projectFiles ProjectFiles, providerType types.ProviderType, awsClient *aws.Client) error {
	predictor := api.Predictor

	switch predictor.Type {
	case userconfig.PythonPredictorType:
		if err := validatePythonPredictor(predictor); err != nil {
			return err
		}
	case userconfig.TensorFlowPredictorType:
		if err := validateTensorFlowPredictor(api, providerType, projectFiles, awsClient); err != nil {
			return err
		}
		if err := validateDockerImagePath(predictor.TensorFlowServingImage, providerType, awsClient); err != nil {
			return errors.Wrap(err, userconfig.TensorFlowServingImageKey)
		}
	case userconfig.ONNXPredictorType:
		if err := validateONNXPredictor(predictor, providerType, projectFiles, awsClient); err != nil {
			return err
		}
	}

	if err := validateDockerImagePath(predictor.Image, providerType, awsClient); err != nil {
		return errors.Wrap(err, userconfig.ImageKey)
	}

	for key := range predictor.Env {
		if strings.HasPrefix(key, "CORTEX_") {
			return errors.Wrap(ErrorCortexPrefixedEnvVarNotAllowed(), userconfig.EnvKey, key)
		}
	}

	if _, err := projectFiles.GetFile(predictor.Path); err != nil {
		if errors.GetKind(err) == files.ErrFileDoesNotExist {
			return errors.Wrap(files.ErrorFileDoesNotExist(predictor.Path), userconfig.PathKey)
		}
		return errors.Wrap(err, userconfig.PathKey)
	}

	if predictor.PythonPath != nil {
		if err := validatePythonPath(*predictor.PythonPath, projectFiles); err != nil {
			return errors.Wrap(err, userconfig.PythonPathKey)
		}
	}

	return nil
}

func validatePythonPredictor(predictor *userconfig.Predictor) error {
	if predictor.SignatureKey != nil {
		return ErrorFieldNotSupportedByPredictorType(userconfig.SignatureKeyKey, userconfig.PythonPredictorType)
	}

	if predictor.Model != nil {
		return ErrorFieldNotSupportedByPredictorType(userconfig.ModelKey, userconfig.PythonPredictorType)
	}

	if len(predictor.Models) > 0 {
		return ErrorFieldNotSupportedByPredictorType(userconfig.ModelsKey, userconfig.PythonPredictorType)
	}

	if predictor.TensorFlowServingImage != "" {
		return ErrorFieldNotSupportedByPredictorType(userconfig.TensorFlowServingImageKey, userconfig.PythonPredictorType)
	}

	return nil
}

func validateTensorFlowPredictor(api *userconfig.API, providerType types.ProviderType, projectFiles ProjectFiles, awsClient *aws.Client) error {
	predictor := api.Predictor

	if predictor.Model == nil && len(predictor.Models) == 0 {
		return ErrorMissingModel(userconfig.ModelKey, userconfig.ModelsKey, predictor.Type)
	} else if predictor.Model != nil && len(predictor.Models) > 0 {
		return ErrorConflictingFields(userconfig.ModelKey, userconfig.ModelsKey)
	} else if predictor.Model != nil {
		modelResource := &userconfig.ModelResource{
			Name:         consts.SingleModelName,
			Model:        *predictor.Model,
			SignatureKey: predictor.SignatureKey,
		}
		// place the predictor.Model into predictor.Models for ease of use
		predictor.Models = []*userconfig.ModelResource{modelResource}
	}

	if err := checkDuplicateModelNames(predictor.Models); err != nil {
		return errors.Wrap(err, userconfig.ModelsKey)
	}

	for i := range predictor.Models {
		if err := validateTensorFlowModel(predictor.Models[i], api, providerType, projectFiles, awsClient); err != nil {
			if predictor.Model == nil {
				return errors.Wrap(err, userconfig.ModelsKey, predictor.Models[i].Name)
			}
			return err
		}
	}

	return nil
}

func validateTensorFlowModel(modelResource *userconfig.ModelResource, api *userconfig.API, providerType types.ProviderType, projectFiles ProjectFiles, awsClient *aws.Client) error {
	model := modelResource.Model

	if strings.HasPrefix(model, "s3://") {
		awsClientForBucket, err := aws.NewFromClientS3Path(model, awsClient)
		if err != nil {
			return errors.Wrap(err, userconfig.ModelKey)
		}

		model, err := cr.S3PathValidator(model)
		if err != nil {
			return errors.Wrap(err, userconfig.ModelKey)
		}

		if strings.HasSuffix(model, ".zip") {
			if ok, err := awsClientForBucket.IsS3PathFile(model); err != nil || !ok {
				return errors.Wrap(ErrorS3FileNotFound(model), userconfig.ModelKey)
			}
		} else {

			isNeuronExport := api.Compute.Inf > 0
			path, err := getTFServingExportFromS3Path(model, isNeuronExport, awsClientForBucket)
			if err != nil {
				return errors.Wrap(err, userconfig.ModelKey)
			}
			if path == "" {
				if isNeuronExport {
					return errors.Wrap(ErrorInvalidNeuronTensorFlowDir(model), userconfig.ModelKey)
				}
				return errors.Wrap(ErrorInvalidTensorFlowDir(model), userconfig.ModelKey)
			}
			modelResource.Model = path
		}
	} else {
		if providerType == types.AWSProviderType {
			return errors.Wrap(ErrorLocalModelPathNotSupportedByAWSProvider(), model, userconfig.ModelKey)
		}

		configFileDir := filepath.Dir(projectFiles.GetConfigFilePath())

		var err error
		if strings.HasPrefix(modelResource.Model, "~/") {
			model, err = files.EscapeTilde(model)
			if err != nil {
				return err
			}
		} else {
			model = files.RelToAbsPath(modelResource.Model, configFileDir)
		}
		if strings.HasSuffix(model, ".zip") {
			if err := files.CheckFile(model); err != nil {
				return errors.Wrap(err, userconfig.ModelKey)
			}
			modelResource.Model = model
		} else if files.IsDir(model) {
			path, err := GetTFServingExportFromLocalPath(model)
			if err != nil {
				return errors.Wrap(err, userconfig.ModelKey)
			} else if path == "" {
				return errors.Wrap(ErrorInvalidTensorFlowDir(model), userconfig.ModelKey)
			}
			modelResource.Model = path
		} else {
			return errors.Wrap(ErrorInvalidTensorFlowModelPath(), userconfig.ModelKey, model)
		}
	}

	return nil
}

func validateONNXPredictor(predictor *userconfig.Predictor, providerType types.ProviderType, projectFiles ProjectFiles, awsClient *aws.Client) error {
	if predictor.SignatureKey != nil {
		return ErrorFieldNotSupportedByPredictorType(userconfig.SignatureKeyKey, predictor.Type)
	}
	if predictor.Model == nil && len(predictor.Models) == 0 {
		return ErrorMissingModel(userconfig.ModelKey, userconfig.ModelsKey, predictor.Type)
	} else if predictor.Model != nil && len(predictor.Models) > 0 {
		return ErrorConflictingFields(userconfig.ModelKey, userconfig.ModelsKey)
	} else if predictor.Model != nil {
		modelResource := &userconfig.ModelResource{
			Name:  consts.SingleModelName,
			Model: *predictor.Model,
		}
		// place the predictor.Model into predictor.Models for ease of use
		predictor.Models = []*userconfig.ModelResource{modelResource}
	}

	if err := checkDuplicateModelNames(predictor.Models); err != nil {
		return errors.Wrap(err, userconfig.ModelsKey)
	}

	for i := range predictor.Models {
		if predictor.Models[i].SignatureKey != nil {
			return errors.Wrap(ErrorFieldNotSupportedByPredictorType(userconfig.SignatureKeyKey, predictor.Type), userconfig.ModelsKey, predictor.Models[i].Name)
		}
		if err := validateONNXModel(predictor.Models[i], providerType, projectFiles, awsClient); err != nil {
			if predictor.Model == nil {
				return errors.Wrap(err, userconfig.ModelsKey, predictor.Models[i].Name)
			}
			return err
		}
	}

	return nil
}

func validateONNXModel(modelResource *userconfig.ModelResource, providerType types.ProviderType, projectFiles ProjectFiles, awsClient *aws.Client) error {
	model := modelResource.Model
	var err error
	if !strings.HasSuffix(model, ".onnx") {
		return errors.Wrap(ErrorInvalidONNXModelPath(), userconfig.ModelKey, model)
	}

	if strings.HasPrefix(model, "s3://") {
		awsClientForBucket, err := aws.NewFromClientS3Path(model, awsClient)
		if err != nil {
			return errors.Wrap(err, userconfig.ModelKey)
		}

		model, err := cr.S3PathValidator(model)
		if err != nil {
			return errors.Wrap(err, userconfig.ModelKey)
		}

		if ok, err := awsClientForBucket.IsS3PathFile(model); err != nil || !ok {
			return errors.Wrap(ErrorS3FileNotFound(model), userconfig.ModelKey)
		}
	} else {
		if providerType == types.AWSProviderType {
			return errors.Wrap(ErrorLocalModelPathNotSupportedByAWSProvider(), model, userconfig.ModelKey)
		}

		configFileDir := filepath.Dir(projectFiles.GetConfigFilePath())
		if strings.HasPrefix(modelResource.Model, "~/") {
			model, err = files.EscapeTilde(model)
			if err != nil {
				return err
			}
		} else {
			model = files.RelToAbsPath(modelResource.Model, configFileDir)
		}
		if err := files.CheckFile(model); err != nil {
			return errors.Wrap(err, userconfig.ModelKey)
		}
		modelResource.Model = model
	}
	return nil
}

func getTFServingExportFromS3Path(path string, isNeuronExport bool, awsClientForBucket *aws.Client) (string, error) {
	if isValidTensorFlowS3Directory(path, awsClientForBucket) {
		return path, nil
	}

	bucket, _, err := aws.SplitS3Path(path)
	if err != nil {
		return "", err
	}

	objects, err := awsClientForBucket.ListS3PathDir(path, false, pointer.Int64(1000))
	if err != nil {
		return "", err
	} else if len(objects) == 0 {
		return "", errors.Wrap(ErrorInvalidTensorFlowModelPath(), path)
	}

	highestVersion := int64(0)
	var highestPath string
	for _, object := range objects {
		if !strings.HasSuffix(*object.Key, "saved_model.pb") {
			continue
		}

		keyParts := strings.Split(*object.Key, "/")
		versionStr := keyParts[len(keyParts)-1]
		version, err := strconv.ParseInt(versionStr, 10, 64)
		if err != nil {
			version = 0
		}

		possiblePath := "s3://" + filepath.Join(bucket, filepath.Join(keyParts[:len(keyParts)-1]...))

		if version >= highestVersion {
			if isNeuronExport && isValidNeuronTensorFlowS3Directory(possiblePath, awsClientForBucket) {
				highestVersion = version
				highestPath = possiblePath
			}
			if !isNeuronExport && isValidTensorFlowS3Directory(possiblePath, awsClientForBucket) {
				highestVersion = version
				highestPath = possiblePath
			}
		}
	}

	return highestPath, nil
}

// isValidTensorFlowS3Directory checks that the path contains a valid S3 directory for TensorFlow models
// Must contain the following structure:
// - 1523423423/ (version prefix, usually a timestamp)
// 		- saved_model.pb
//		- variables/
//			- variables.index
//			- variables.data-00000-of-00001 (there are a variable number of these files)
func isValidTensorFlowS3Directory(path string, awsClientForBucket *aws.Client) bool {
	if valid, err := awsClientForBucket.IsS3PathFile(
		aws.JoinS3Path(path, "saved_model.pb"),
		aws.JoinS3Path(path, "variables/variables.index"),
	); err != nil || !valid {
		return false
	}

	if valid, err := awsClientForBucket.IsS3PathPrefix(
		aws.JoinS3Path(path, "variables/variables.data-00000-of"),
	); err != nil || !valid {
		return false
	}
	return true
}

// isValidNeuronTensorFlowS3Directory checks that the path contains a valid S3 directory for Neuron TensorFlow models
// Must contain the following structure:
// - 1523423423/ (version prefix, usually a timestamp)
// 		- saved_model.pb
func isValidNeuronTensorFlowS3Directory(path string, awsClient *aws.Client) bool {
	if valid, err := awsClient.IsS3PathFile(
		aws.JoinS3Path(path, "saved_model.pb"),
	); err != nil || !valid {
		return false
	}

	return true
}

func GetTFServingExportFromLocalPath(path string) (string, error) {
	if err := files.CheckDir(path); err != nil {
		return "", err
	}
	paths, err := files.ListDirRecursive(path, false, files.IgnoreHiddenFiles, files.IgnoreHiddenFolders)
	if err != nil {
		return "", err
	}

	if len(paths) == 0 {
		return "", ErrorDirIsEmpty(path)
	}

	highestVersion := int64(0)
	var highestPath string

	for _, path := range paths {
		if strings.HasSuffix(path, "saved_model.pb") {
			possiblePath := filepath.Dir(path)

			versionStr := filepath.Base(possiblePath)
			version, err := strconv.ParseInt(versionStr, 10, 64)
			if err != nil {
				version = 0
			}

			validTFDirectory, err := IsValidTensorFlowLocalDirectory(possiblePath)
			if err != nil {
				return "", err
			}
			if version > highestVersion && validTFDirectory {
				highestVersion = version
				highestPath = possiblePath
			}
		}
	}

	return highestPath, nil
}

func IsValidTensorFlowLocalDirectory(path string) (bool, error) {
	paths, err := files.ListDirRecursive(path, true, files.IgnoreHiddenFiles, files.IgnoreHiddenFolders)
	if err != nil {
		return false, err
	}
	pathSet := strset.New(paths...)

	if !(pathSet.Has("saved_model.pb") && pathSet.Has("variables/variables.index")) {
		return false, nil
	}

	for _, path := range paths {
		if strings.HasPrefix(path, "variables/variables.data-00000-of") {
			return true, nil
		}
	}

	return false, nil
}

func validatePythonPath(pythonPath string, projectFiles ProjectFiles) error {
	validPythonPath := false
	for _, fileKey := range projectFiles.GetAllPaths() {
		if strings.HasPrefix(fileKey, pythonPath) {
			validPythonPath = true
			break
		}
	}
	if !validPythonPath {
		return files.ErrorFileDoesNotExist(pythonPath)
	}
	return nil
}

func validateAutoscaling(api *userconfig.API) error {
	autoscaling := api.Autoscaling

	if autoscaling.TargetReplicaConcurrency == nil {
		autoscaling.TargetReplicaConcurrency = pointer.Float64(float64(autoscaling.WorkersPerReplica * autoscaling.ThreadsPerWorker))
	}

	if *autoscaling.TargetReplicaConcurrency > float64(autoscaling.MaxReplicaConcurrency) {
		return ErrorConfigGreaterThanOtherConfig(userconfig.TargetReplicaConcurrencyKey, *autoscaling.TargetReplicaConcurrency, userconfig.MaxReplicaConcurrencyKey, autoscaling.MaxReplicaConcurrency)
	}

	if autoscaling.MinReplicas > autoscaling.MaxReplicas {
		return ErrorMinReplicasGreaterThanMax(autoscaling.MinReplicas, autoscaling.MaxReplicas)
	}

	if autoscaling.InitReplicas > autoscaling.MaxReplicas {
		return ErrorInitReplicasGreaterThanMax(autoscaling.InitReplicas, autoscaling.MaxReplicas)
	}

	if autoscaling.InitReplicas < autoscaling.MinReplicas {
		return ErrorInitReplicasLessThanMin(autoscaling.InitReplicas, autoscaling.MinReplicas)
	}

	if api.Compute.Inf > 0 {
		numNeuronCores := api.Compute.Inf * consts.NeuronCoresPerInf
		workersPerReplica := int64(api.Autoscaling.WorkersPerReplica)
		if !libmath.IsDivisibleByInt64(numNeuronCores, workersPerReplica) {
			return ErrorInvalidNumberOfInfWorkers(workersPerReplica, api.Compute.Inf, numNeuronCores)
		}
	}

	return nil
}

func validateCompute(api *userconfig.API, providerType types.ProviderType) error {
	compute := api.Compute

	if compute.Inf > 0 && providerType == types.LocalProviderType {
		return ErrorUnsupportedLocalComputeResource(userconfig.InfKey)
	}

	if compute.Inf > 0 && api.Predictor.Type == userconfig.ONNXPredictorType {
		return ErrorFieldNotSupportedByPredictorType(userconfig.InfKey, api.Predictor.Type)
	}

	if compute.GPU > 0 && compute.Inf > 0 {
		return ErrorComputeResourceConflict(userconfig.GPUKey, userconfig.InfKey)
	}

	if compute.Inf > 1 {
		return ErrorInvalidNumberOfInfs(compute.Inf)
	}

	return nil
}

func validateUpdateStrategy(updateStrategy *userconfig.UpdateStrategy) error {
	if (updateStrategy.MaxSurge == "0" || updateStrategy.MaxSurge == "0%") && (updateStrategy.MaxUnavailable == "0" || updateStrategy.MaxUnavailable == "0%") {
		return ErrorSurgeAndUnavailableBothZero()
	}

	return nil
}

func FindDuplicateNames(apis []userconfig.API) []userconfig.API {
	names := make(map[string][]userconfig.API)

	for _, api := range apis {
		names[api.Name] = append(names[api.Name], api)
	}

	for name := range names {
		if len(names[name]) > 1 {
			return names[name]
		}
	}

	return nil
}

func checkDuplicateModelNames(modelResources []*userconfig.ModelResource) error {
	names := strset.New()

	for _, modelResource := range modelResources {
		if names.Has(modelResource.Name) {
			return ErrorDuplicateModelNames(modelResource.Name)
		}
		names.Add(modelResource.Name)
	}

	return nil
}

func validateDockerImagePath(image string, providerType types.ProviderType, awsClient *aws.Client) error {
	if consts.DefaultImagePathsSet.Has(image) {
		return nil
	}
	if _, err := cr.ValidateImageVersion(image, consts.CortexVersion); err != nil {
		return err
	}

	dockerClient, err := docker.GetDockerClient()
	if err != nil {
		return err
	}

	if providerType == types.LocalProviderType {
		// short circuit if the image is already available locally
		if err := docker.CheckLocalImageAccessible(dockerClient, image); err == nil {
			return nil
		}
	}

	dockerAuth := docker.NoAuth
	if regex.IsValidECRURL(image) {
		if awsClient.IsAnonymous {
			return errors.Wrap(ErrorCannotAccessECRWithAnonymousAWSCreds(), image)
		}

		ecrRegion := aws.GetRegionFromECRURL(image)
		if ecrRegion != awsClient.Region {
			return ErrorRegistryInDifferentRegion(ecrRegion, awsClient.Region)
		}

		operatorID, _, err := awsClient.GetCachedAccountID()
		if err != nil {
			return err
		}
		registryID := aws.GetAccountIDFromECRURL(image)

		if operatorID != registryID {
			return ErrorRegistryAccountIDMismatch(registryID, operatorID)
		}

		dockerAuth, err = docker.AWSAuthConfig(awsClient)
		if err != nil {
			if _, ok := errors.CauseOrSelf(err).(awserr.Error); ok {
				// because the operator's IAM user != instances's IAM role (which is created by eksctl and
				// has access to ECR), if the operator IAM doesn't include ECR access, then this will fail
				// even though the instance IAM role may have access; instead, ignore this error because the
				// instance will have access (this will result in missing the case where the image does not exist)
				return nil
			}

			return err
		}
	}

	if err := docker.CheckImageAccessible(dockerClient, image, dockerAuth); err != nil {
		return err
	}

	return nil
}
