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
	"log"
	"math"
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
	s "github.com/cortexlabs/cortex/pkg/lib/strings"
	libtime "github.com/cortexlabs/cortex/pkg/lib/time"
	"github.com/cortexlabs/cortex/pkg/lib/urls"
	"github.com/cortexlabs/cortex/pkg/types"
	"github.com/cortexlabs/cortex/pkg/types/userconfig"
	"gopkg.in/yaml.v2"
	kresource "k8s.io/apimachinery/pkg/api/resource"
)

var AutoscalingTickInterval = 10 * time.Second

func apiValidation(provider types.ProviderType, kind userconfig.Kind) *cr.StructValidation {
	structFieldValidations := []*cr.StructFieldValidation{}
	structFieldValidations = append(resourceStructValidations,
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
					StructField: "ModelPath",
					StringPtrValidation: &cr.StringPtrValidation{
						Required: false,
					},
				},
				{
					StructField: "PythonPath",
					StringPtrValidation: &cr.StringPtrValidation{
						AllowEmpty:       false,
						DisallowedValues: []string{".", "./", "./."},
						Validator: func(path string) (string, error) {
							if files.IsAbsOrTildePrefixed(path) {
								return "", ErrorMustBeRelativeProjectPath(path)
							}
							path = strings.TrimPrefix(path, "./")
							path = s.EnsureSuffix(path, "/")
							return path, nil
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
					StructField: "ProcessesPerReplica",
					Int32Validation: &cr.Int32Validation{
						Default:              1,
						GreaterThanOrEqualTo: pointer.Int32(1),
						LessThanOrEqualTo:    pointer.Int32(100),
					},
				},
				{
					StructField: "ThreadsPerProcess",
					Int32Validation: &cr.Int32Validation{
						Default:              1,
						GreaterThanOrEqualTo: pointer.Int32(1),
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
					StructField: "Endpoint",
					StringPtrValidation: &cr.StringPtrValidation{
						Validator: urls.ValidateEndpoint,
						MaxLength: 1000, // no particular reason other than it works
					},
				},
				{
					StructField: "LocalPort",
					IntPtrValidation: &cr.IntPtrValidation{
						GreaterThan:       pointer.Int(0),
						LessThanOrEqualTo: pointer.Int(math.MaxUint16),
					},
				},
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
					StructField: "TargetReplicaConcurrency",
					Float64PtrValidation: &cr.Float64PtrValidation{
						GreaterThan: pointer.Float64(0),
					},
				},
				{
					StructField: "MaxReplicaConcurrency",
					Int64Validation: &cr.Int64Validation{
						Default:           consts.DefaultMaxReplicaConcurrency,
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
		StructValidation: &cr.StructValidation{
			Required:   false,
			DefaultNil: true,
			StructFieldValidations: []*cr.StructFieldValidation{
				multiModelPathsValidation(),
				{
					StructField: "Dir",
					StringPtrValidation: &cr.StringPtrValidation{
						Required: false,
					},
				},
				{
					StructField: "SignatureKey",
					StringPtrValidation: &cr.StringPtrValidation{
						Required: false,
					},
				},
				{
					StructField: "CacheSize",
					Int32PtrValidation: &cr.Int32PtrValidation{
						Required:    false,
						GreaterThan: pointer.Int32(0),
					},
				},
				{
					StructField: "DiskCacheSize",
					Int32PtrValidation: &cr.Int32PtrValidation{
						Required:    false,
						GreaterThan: pointer.Int32(0),
					},
				},
			},
		},
	}
}

func multiModelPathsValidation() *cr.StructFieldValidation {
	return &cr.StructFieldValidation{
		StructField: "Paths",
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
						StructField: "ModelPath",
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

var resourceStructValidation = cr.StructValidation{
	AllowExtraFields:       true,
	StructFieldValidations: resourceStructValidations,
}

func ExtractAPIConfigs(configBytes []byte, provider types.ProviderType, configFileName string) ([]userconfig.API, error) {
	var err error

	configData, err := cr.ReadYAMLBytes(configBytes)
	if err != nil {
		return nil, errors.Wrap(err, configFileName)
	}

	configDataSlice, ok := cast.InterfaceToStrInterfaceMapSlice(configData)
	if !ok {
		return nil, errors.Wrap(ErrorMalformedConfig(), configFileName)
	}

	apis := make([]userconfig.API, len(configDataSlice))
	for i, data := range configDataSlice {
		api := userconfig.API{}
		var resourceStruct userconfig.Resource
		errs := cr.Struct(&resourceStruct, data, &resourceStructValidation)
		if errors.HasError(errs) {
			name, _ := data[userconfig.NameKey].(string)
			kindString, _ := data[userconfig.KindKey].(string)
			kind := userconfig.KindFromString(kindString)
			err = errors.Wrap(errors.FirstError(errs...), userconfig.IdentifyAPI(configFileName, name, kind, i))
			return nil, errors.Append(err, fmt.Sprintf("\n\napi configuration schema can be found here: https://docs.cortex.dev/v/%s/deployments/api-configuration", consts.CortexVersionMinor))
		}

		errs = cr.Struct(&api, data, apiValidation(provider, resourceStruct.Kind))

		if errors.HasError(errs) {
			name, _ := data[userconfig.NameKey].(string)
			kindString, _ := data[userconfig.KindKey].(string)
			kind := userconfig.KindFromString(kindString)
			err = errors.Wrap(errors.FirstError(errs...), userconfig.IdentifyAPI(configFileName, name, kind, i))
			return nil, errors.Append(err, fmt.Sprintf("\n\napi configuration schema can be found here: https://docs.cortex.dev/v/%s/deployments/api-configuration", consts.CortexVersionMinor))
		}
		api.Index = i
		api.FileName = configFileName

		api.ApplyDefaultDockerPaths()

		apis[i] = api
	}

	return apis, nil
}

func ValidateAPI(
	api *userconfig.API,
	models *[]CuratedModelResource,
	projectFiles ProjectFiles,
	providerType types.ProviderType,
	awsClient *aws.Client,
) error {
	if providerType == types.AWSProviderType && api.Networking.Endpoint == nil {
		api.Networking.Endpoint = pointer.String("/" + api.Name)
	}

	if err := validatePredictor(api, models, projectFiles, providerType, awsClient); err != nil {
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

	// TODO to remove when no longer required
	apiDump, err := yaml.Marshal(&api)
	if err != nil {
		log.Fatalf("error while dumping api spec: %v", err)
	}
	fmt.Printf("--- api dump:\n%s\n\n", string(apiDump))

	fmt.Println("--- model dump")
	for _, model := range *models {
		fmt.Println("model.Name", model.Name, "model.ModelPath", model.ModelPath, "model.Versions", model.Versions, "model.S3Path", model.S3Path)
	}
	fmt.Print("\n\n")

	return &errors.Error{}

	return nil
}

func validatePredictor(
	api *userconfig.API,
	models *[]CuratedModelResource,
	projectFiles ProjectFiles,
	providerType types.ProviderType,
	awsClient *aws.Client,
) error {
	predictor := api.Predictor

	if predictor.Models != nil && predictor.ModelPath != nil {
		return ErrorConflictingFields(userconfig.ModelPathKey, userconfig.ModelsKey)
	}
	if err := validateMultiModelsFields(predictor); err != nil {
		return err
	}

	switch predictor.Type {
	case userconfig.PythonPredictorType:
		if err := validatePythonPredictor(predictor, models, providerType, projectFiles, awsClient); err != nil {
			return err
		}
	case userconfig.TensorFlowPredictorType:
		if err := validateTensorFlowPredictor(api, models, providerType, projectFiles, awsClient); err != nil {
			return err
		}
		if err := validateDockerImagePath(predictor.TensorFlowServingImage, providerType, awsClient); err != nil {
			return errors.Wrap(err, userconfig.TensorFlowServingImageKey)
		}
	case userconfig.ONNXPredictorType:
		if err := validateONNXPredictor(predictor, models, providerType, projectFiles, awsClient); err != nil {
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

	if !projectFiles.HasFile(predictor.Path) {
		return errors.Wrap(files.ErrorFileDoesNotExist(predictor.Path), userconfig.PathKey)
	}

	if predictor.PythonPath != nil {
		if err := validatePythonPath(predictor, projectFiles); err != nil {
			return errors.Wrap(err, userconfig.PythonPathKey)
		}
	}

	return nil
}

func validateMultiModelsFields(predictor *userconfig.Predictor) error {
	if predictor.Models == nil {
		return nil
	}

	if len(predictor.Models.Paths) == 0 && predictor.Models.Dir == nil {
		return errors.Wrap(ErrorSpecifyOneOrTheOther(userconfig.ModelsPathsKey, userconfig.ModelsDirKey), userconfig.ModelsKey)
	}
	if len(predictor.Models.Paths) > 0 && predictor.Models.Dir != nil {
		return errors.Wrap(ErrorConflictingFields(userconfig.ModelsPathsKey, userconfig.ModelsDirKey), userconfig.ModelsKey)
	}
	if (predictor.Models.CacheSize == nil && predictor.Models.DiskCacheSize != nil) ||
		(predictor.Models.CacheSize != nil && predictor.Models.DiskCacheSize == nil) {
		return errors.Wrap(ErrorSpecifyAllOrNone(userconfig.ModelsCacheSizeKey, userconfig.ModelsDiskCacheSizeKey), userconfig.ModelsKey)
	}

	if predictor.Models.CacheSize != nil && predictor.Models.DiskCacheSize != nil {
		if *predictor.Models.CacheSize > *predictor.Models.DiskCacheSize {
			return errors.Wrap(ErrorConfigGreaterThanOtherConfig(userconfig.ModelsCacheSizeKey, *predictor.Models.CacheSize, userconfig.ModelsDiskCacheSizeKey, *predictor.Models.DiskCacheSize), userconfig.ModelsKey)
		}

		if predictor.ProcessesPerReplica > 1 {
			return ErrorInvalidNumberOfProcessesWhenCaching(predictor.ProcessesPerReplica)
		}
	}

	return nil
}

func validatePythonPredictor(predictor *userconfig.Predictor, models *[]CuratedModelResource, providerType types.ProviderType, projectFiles ProjectFiles, awsClient *aws.Client) error {
	if predictor.SignatureKey != nil {
		return ErrorFieldNotSupportedByPredictorType(userconfig.SignatureKeyKey, predictor.Type)
	}
	if predictor.TensorFlowServingImage != "" {
		return ErrorFieldNotSupportedByPredictorType(userconfig.TensorFlowServingImageKey, predictor.Type)
	}

	hasSingleModel := predictor.ModelPath != nil
	hasMultiModels := predictor.Models != nil

	var modelWrapError func(error) error
	var modelResources []userconfig.ModelResource

	if hasSingleModel {
		modelResources = []userconfig.ModelResource{
			{
				Name:      consts.SingleModelName,
				ModelPath: *predictor.ModelPath,
			},
		}
		modelWrapError = func(err error) error {
			return errors.Wrap(err, userconfig.ModelPathKey)
		}
	}
	if hasMultiModels {
		if predictor.Models.SignatureKey != nil {
			return errors.Wrap(ErrorFieldNotSupportedByPredictorType(userconfig.SignatureKeyKey, predictor.Type), userconfig.ModelsKey)
		}

		if len(predictor.Models.Paths) > 0 {
			modelWrapError = func(err error) error {
				return errors.Wrap(err, userconfig.ModelsKey, userconfig.ModelsPathsKey)
			}

			for _, path := range predictor.Models.Paths {
				if path.SignatureKey != nil {
					return errors.Wrap(
						ErrorFieldNotSupportedByPredictorType(userconfig.SignatureKeyKey, predictor.Type),
						userconfig.ModelsKey,
						userconfig.ModelsPathsKey,
						path.Name,
					)
				}
				modelResources = append(modelResources, *path)
			}
		}

		if predictor.Models.Dir != nil {
			modelWrapError = func(err error) error {
				return errors.Wrap(err, userconfig.ModelsKey, userconfig.ModelsDirKey)
			}

			var err error
			modelResources, err = retrieveModelsResourcesFromPath(*predictor.Models.Dir, projectFiles, awsClient)
			if err != nil {
				return modelWrapError(err)
			}
		}
	}
	var err error
	*models, err = modelResourceToCurated(modelResources, predictor.Type, projectFiles)
	if err != nil {
		return modelWrapError(err)
	}

	if hasMultiModels {
		for _, model := range *models {
			if model.Name == consts.SingleModelName {
				return modelWrapError(ErrorIllegalModelName(model.Name))
			}
		}
	}

	if err := checkDuplicateModelNames(*models); err != nil {
		return modelWrapError(err)
	}

	for i := range *models {
		if err := validatePythonModel(&(*models)[i], providerType, projectFiles, awsClient); err != nil {
			return modelWrapError(err)
		}
	}

	return nil
}

func validatePythonModel(modelResource *CuratedModelResource, providerType types.ProviderType, projectFiles ProjectFiles, awsClient *aws.Client) error {
	modelName := modelResource.Name
	if modelName == consts.SingleModelName {
		modelName = ""
	}

	if modelResource.S3Path {
		awsClientForBucket, err := aws.NewFromClientS3Path(modelResource.ModelPath, awsClient)
		if err != nil {
			return errors.Wrap(err, modelName)
		}

		_, err = cr.S3PathValidator(modelResource.ModelPath)
		if err != nil {
			return errors.Wrap(err, modelName)
		}

		versions, err := getPythonVersionsFromS3Paths(modelResource.ModelPath, awsClientForBucket)
		if err != nil {
			if errors.GetKind(err) == errors.ErrNotCortexError || errors.GetKind(err) == ErrModelPathNotDirectory {
				return errors.Wrap(err, modelName)
			}

			modelSubPaths, err := awsClientForBucket.GetPrefixesFromS3Path(modelResource.ModelPath, false, pointer.Int64(1000))
			if err != nil {
				return errors.Wrap(err, modelName)
			}

			if err = validatePythonS3ModelDir(modelResource.ModelPath, awsClientForBucket); err != nil {
				if errors.GetKind(err) == errors.ErrNotCortexError {
					return errors.Wrap(err, modelName)
				}
				return errors.Wrap(ErrorInvalidPythonModelPath(modelResource.ModelPath, modelResource.S3Path, true, true, modelSubPaths), modelName)
			}
		}
		modelResource.Versions = versions
	} else {
		if providerType == types.AWSProviderType {
			return ErrorLocalModelPathNotSupportedByAWSProvider()
		}

		versions, err := getPythonVersionsFromLocalPaths(modelResource.ModelPath)
		if err != nil {
			if errors.GetKind(err) == errors.ErrNotCortexError || errors.GetKind(err) == ErrModelPathNotDirectory {
				return errors.Wrap(err, modelName)
			}

			modelSubPaths, err := files.ListDirRecursive(modelResource.ModelPath, false)
			if err != nil {
				return errors.Wrap(err, modelName)
			}

			if err = validatePythonLocalModelDir(modelResource.ModelPath); err != nil {
				if errors.GetKind(err) == errors.ErrNotCortexError {
					return errors.Wrap(err, modelName)
				}
				return errors.Wrap(ErrorInvalidPythonModelPath(modelResource.ModelPath, modelResource.S3Path, true, true, modelSubPaths), modelName)
			}
		}
		modelResource.Versions = versions
	}

	return nil
}

func validateTensorFlowPredictor(api *userconfig.API, models *[]CuratedModelResource, providerType types.ProviderType, projectFiles ProjectFiles, awsClient *aws.Client) error {
	predictor := api.Predictor

	hasSingleModel := predictor.ModelPath != nil
	hasMultiModels := predictor.Models != nil

	if !hasSingleModel && !hasMultiModels {
		return ErrorMissingModel(predictor.Type)
	}

	var modelWrapError func(error) error
	var modelResources []userconfig.ModelResource

	if hasSingleModel {
		modelResources = []userconfig.ModelResource{
			{
				Name:         consts.SingleModelName,
				ModelPath:    *predictor.ModelPath,
				SignatureKey: predictor.SignatureKey,
			},
		}
		modelWrapError = func(err error) error {
			return errors.Wrap(err, userconfig.ModelPathKey)
		}
	}
	if hasMultiModels {
		if len(predictor.Models.Paths) > 0 {
			modelWrapError = func(err error) error {
				return errors.Wrap(err, userconfig.ModelsKey, userconfig.ModelsPathsKey)
			}

			for _, path := range predictor.Models.Paths {
				if path.SignatureKey == nil && predictor.Models.SignatureKey != nil {
					path.SignatureKey = predictor.Models.SignatureKey
				}
				modelResources = append(modelResources, *path)
			}
		}

		if predictor.Models.Dir != nil {
			modelWrapError = func(err error) error {
				return errors.Wrap(err, userconfig.ModelsKey, userconfig.ModelsDirKey)
			}

			var err error
			modelResources, err = retrieveModelsResourcesFromPath(*predictor.Models.Dir, projectFiles, awsClient)
			if err != nil {
				return modelWrapError(err)
			}
			if predictor.Models.SignatureKey != nil {
				for i := range modelResources {
					modelResources[i].SignatureKey = predictor.Models.SignatureKey
				}
			}
		}
	}
	var err error
	*models, err = modelResourceToCurated(modelResources, predictor.Type, projectFiles)
	if err != nil {
		return err
	}

	if hasMultiModels {
		for _, model := range *models {
			if model.Name == consts.SingleModelName {
				return modelWrapError(ErrorIllegalModelName(model.Name))
			}
		}
	}

	if err := checkDuplicateModelNames(*models); err != nil {
		return modelWrapError(err)
	}

	for i := range *models {
		if err := validateTensorFlowModel(&(*models)[i], api, providerType, projectFiles, awsClient); err != nil {
			return modelWrapError(err)
		}
	}

	return nil
}

func validateTensorFlowModel(
	modelResource *CuratedModelResource,
	api *userconfig.API,
	providerType types.ProviderType,
	projectFiles ProjectFiles,
	awsClient *aws.Client,
) error {

	modelName := modelResource.Name
	if modelName == consts.SingleModelName {
		modelName = ""
	}

	if modelResource.S3Path {
		awsClientForBucket, err := aws.NewFromClientS3Path(modelResource.ModelPath, awsClient)
		if err != nil {
			return errors.Wrap(err, modelName)
		}

		_, err = cr.S3PathValidator(modelResource.ModelPath)
		if err != nil {
			return errors.Wrap(err, modelName)
		}

		isNeuronExport := api.Compute.Inf > 0
		versions, err := getTFServingVersionsFromS3Path(modelResource.ModelPath, isNeuronExport, awsClientForBucket)
		if err != nil {
			if errors.GetKind(err) == errors.ErrNotCortexError || errors.GetKind(err) == ErrModelPathNotDirectory {
				return errors.Wrap(err, modelName)
			}

			modelSubPaths, err := awsClientForBucket.GetPrefixesFromS3Path(modelResource.ModelPath, false, pointer.Int64(1000))
			if err != nil {
				return errors.Wrap(err, modelName)
			}

			if err = validateTFServingS3ModelDir(modelResource.ModelPath, isNeuronExport, awsClientForBucket); err != nil {
				if errors.GetKind(err) == errors.ErrNotCortexError {
					return errors.Wrap(err, modelName)
				}
				return errors.Wrap(ErrorInvalidTensorFlowModelPath(modelResource.ModelPath, isNeuronExport, modelResource.S3Path, true, true, modelSubPaths), modelName)
			}
		}
		modelResource.Versions = versions
	} else {
		if providerType == types.AWSProviderType {
			return ErrorLocalModelPathNotSupportedByAWSProvider()
		}

		versions, err := getTFServingVersionsFromLocalPath(modelResource.ModelPath)
		if err != nil {
			if errors.GetKind(err) == errors.ErrNotCortexError || errors.GetKind(err) == ErrModelPathNotDirectory {
				return errors.Wrap(err, modelName)
			}

			modelSubPaths, err := files.ListDirRecursive(modelResource.ModelPath, false)
			if err != nil {
				return errors.Wrap(err, modelName)
			}

			if err = validateTFServingLocalModelDir(modelResource.ModelPath); err != nil {
				if errors.GetKind(err) == errors.ErrNotCortexError {
					return errors.Wrap(err, modelName)
				}
				return errors.Wrap(ErrorInvalidTensorFlowModelPath(modelResource.ModelPath, false, modelResource.S3Path, true, true, modelSubPaths), modelName)
			}
		}
		modelResource.Versions = versions
	}

	return nil
}

func validateONNXPredictor(predictor *userconfig.Predictor, models *[]CuratedModelResource, providerType types.ProviderType, projectFiles ProjectFiles, awsClient *aws.Client) error {
	if predictor.SignatureKey != nil {
		return ErrorFieldNotSupportedByPredictorType(userconfig.SignatureKeyKey, predictor.Type)
	}
	if predictor.TensorFlowServingImage != "" {
		return ErrorFieldNotSupportedByPredictorType(userconfig.TensorFlowServingImageKey, predictor.Type)
	}

	hasSingleModel := predictor.ModelPath != nil
	hasMultiModels := predictor.Models != nil

	if !hasSingleModel && !hasMultiModels {
		return ErrorMissingModel(predictor.Type)
	}

	var modelWrapError func(error) error
	var modelResources []userconfig.ModelResource

	if hasSingleModel {
		modelResources = []userconfig.ModelResource{
			{
				Name:      consts.SingleModelName,
				ModelPath: *predictor.ModelPath,
			},
		}
		modelWrapError = func(err error) error {
			return errors.Wrap(err, userconfig.ModelPathKey)
		}
	}
	if hasMultiModels {
		if predictor.Models.SignatureKey != nil {
			return errors.Wrap(ErrorFieldNotSupportedByPredictorType(userconfig.SignatureKeyKey, predictor.Type), userconfig.ModelsKey)
		}

		if len(predictor.Models.Paths) > 0 {
			modelWrapError = func(err error) error {
				return errors.Wrap(err, userconfig.ModelsKey, userconfig.ModelsPathsKey)
			}

			for _, path := range predictor.Models.Paths {
				if path.SignatureKey != nil {
					return errors.Wrap(
						ErrorFieldNotSupportedByPredictorType(userconfig.SignatureKeyKey, predictor.Type),
						userconfig.ModelsKey,
						userconfig.ModelsPathsKey,
						path.Name,
					)
				}
				modelResources = append(modelResources, *path)
			}
		}

		if predictor.Models.Dir != nil {
			modelWrapError = func(err error) error {
				return errors.Wrap(err, userconfig.ModelsKey, userconfig.ModelsDirKey)
			}

			var err error
			modelResources, err = retrieveModelsResourcesFromPath(*predictor.Models.Dir, projectFiles, awsClient)
			if err != nil {
				return modelWrapError(err)
			}
		}
	}
	var err error
	*models, err = modelResourceToCurated(modelResources, predictor.Type, projectFiles)
	if err != nil {
		return err
	}

	if hasMultiModels {
		for _, model := range *models {
			if model.Name == consts.SingleModelName {
				return modelWrapError(ErrorIllegalModelName(model.Name))
			}
		}
	}

	if err := checkDuplicateModelNames(*models); err != nil {
		return modelWrapError(err)
	}

	for i := range *models {
		if err := validateONNXModel(&(*models)[i], providerType, projectFiles, awsClient); err != nil {
			return modelWrapError(err)
		}
	}

	return nil
}

func validateONNXModel(
	modelResource *CuratedModelResource,
	providerType types.ProviderType,
	projectFiles ProjectFiles,
	awsClient *aws.Client,
) error {

	modelName := modelResource.Name
	if modelName == consts.SingleModelName {
		modelName = ""
	}

	if modelResource.S3Path {
		awsClientForBucket, err := aws.NewFromClientS3Path(modelResource.ModelPath, awsClient)
		if err != nil {
			return errors.Wrap(err, modelName)
		}

		_, err = cr.S3PathValidator(modelResource.ModelPath)
		if err != nil {
			return errors.Wrap(err, modelName)
		}

		versions, err := getONNXVersionsFromS3Path(modelResource.ModelPath, awsClientForBucket)
		if err != nil {
			if errors.GetKind(err) == errors.ErrNotCortexError || errors.GetKind(err) == ErrModelPathNotDirectory {
				return errors.Wrap(err, modelName)
			}

			modelSubPaths, err := awsClientForBucket.GetPrefixesFromS3Path(modelResource.ModelPath, false, pointer.Int64(1000))
			if err != nil {
				return errors.Wrap(err, modelName)
			}

			if err := validateONNXS3ModelDir(modelResource.ModelPath, awsClientForBucket); err != nil {
				if errors.GetKind(err) == errors.ErrNotCortexError {
					return errors.Wrap(err, modelName)
				}
				return errors.Wrap(ErrorInvalidONNXModelPath(modelResource.ModelPath, modelResource.S3Path, true, true, modelSubPaths), modelName)
			}
		}
		modelResource.Versions = versions
	} else {
		if providerType == types.AWSProviderType {
			return ErrorLocalModelPathNotSupportedByAWSProvider()
		}

		versions, err := getONNXVersionsFromLocalPath(modelResource.ModelPath)
		if err != nil {
			if errors.GetKind(err) == errors.ErrNotCortexError || errors.GetKind(err) == ErrModelPathNotDirectory {
				return errors.Wrap(err, modelName)
			}

			modelSubPaths, err := files.ListDirRecursive(modelResource.ModelPath, false)
			if err != nil {
				return errors.Wrap(err, modelName)
			}

			if err := validateONNXLocalModelDir(modelResource.ModelPath); err != nil {
				if errors.GetKind(err) == errors.ErrNotCortexError {
					return errors.Wrap(err, modelName)
				}
				return errors.Wrap(ErrorInvalidONNXModelPath(modelResource.ModelPath, modelResource.S3Path, true, true, modelSubPaths), modelName)
			}
		}
		modelResource.Versions = versions
	}

	return nil
}

func validatePythonPath(predictor *userconfig.Predictor, projectFiles ProjectFiles) error {
	if !projectFiles.HasDir(*predictor.PythonPath) {
		return ErrorPythonPathNotFound(*predictor.PythonPath)
	}

	return nil
}

func validateAutoscaling(api *userconfig.API) error {
	autoscaling := api.Autoscaling
	predictor := api.Predictor

	if autoscaling.TargetReplicaConcurrency == nil {
		autoscaling.TargetReplicaConcurrency = pointer.Float64(float64(predictor.ProcessesPerReplica * predictor.ThreadsPerProcess))
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
		processesPerReplica := int64(predictor.ProcessesPerReplica)
		if !libmath.IsDivisibleByInt64(numNeuronCores, processesPerReplica) {
			return ErrorInvalidNumberOfInfProcesses(processesPerReplica, api.Compute.Inf, numNeuronCores)
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
