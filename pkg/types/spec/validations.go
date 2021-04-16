/*
Copyright 2021 Cortex Labs, Inc.

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
	"bytes"
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/cortexlabs/cortex/pkg/consts"
	"github.com/cortexlabs/cortex/pkg/lib/aws"
	"github.com/cortexlabs/cortex/pkg/lib/cast"
	cr "github.com/cortexlabs/cortex/pkg/lib/configreader"
	"github.com/cortexlabs/cortex/pkg/lib/docker"
	"github.com/cortexlabs/cortex/pkg/lib/errors"
	"github.com/cortexlabs/cortex/pkg/lib/files"
	libjson "github.com/cortexlabs/cortex/pkg/lib/json"
	"github.com/cortexlabs/cortex/pkg/lib/k8s"
	libmath "github.com/cortexlabs/cortex/pkg/lib/math"
	"github.com/cortexlabs/cortex/pkg/lib/pointer"
	"github.com/cortexlabs/cortex/pkg/lib/regex"
	s "github.com/cortexlabs/cortex/pkg/lib/strings"
	libtime "github.com/cortexlabs/cortex/pkg/lib/time"
	"github.com/cortexlabs/cortex/pkg/lib/urls"
	"github.com/cortexlabs/cortex/pkg/types/userconfig"
	dockertypes "github.com/docker/docker/api/types"
	pbparser "github.com/emicklei/proto"
	kresource "k8s.io/apimachinery/pkg/api/resource"
)

var AutoscalingTickInterval = 10 * time.Second

const _dockerPullSecretName = "registry-credentials"

func apiValidation(resource userconfig.Resource) *cr.StructValidation {
	var structFieldValidations []*cr.StructFieldValidation

	switch resource.Kind {
	case userconfig.RealtimeAPIKind:
		structFieldValidations = append(resourceStructValidations,
			predictorValidation(),
			networkingValidation(),
			computeValidation(),
			autoscalingValidation(),
			updateStrategyValidation(),
		)
	case userconfig.AsyncAPIKind:
		structFieldValidations = append(resourceStructValidations,
			predictorValidation(),
			networkingValidation(),
			computeValidation(),
			autoscalingValidation(),
			updateStrategyValidation(),
		)
	case userconfig.BatchAPIKind:
		structFieldValidations = append(resourceStructValidations,
			predictorValidation(),
			networkingValidation(),
			computeValidation(),
		)
	case userconfig.TaskAPIKind:
		structFieldValidations = append(resourceStructValidations,
			taskDefinitionValidation(),
			networkingValidation(),
			computeValidation(),
		)
	case userconfig.TrafficSplitterKind:
		structFieldValidations = append(resourceStructValidations,
			multiAPIsValidation(),
			networkingValidation(),
		)
	}
	return &cr.StructValidation{
		StructFieldValidations: structFieldValidations,
	}
}

var resourceStructValidations = []*cr.StructFieldValidation{
	{
		StructField: "Name",
		StringValidation: &cr.StringValidation{
			Required:        true,
			DNS1035:         true,
			InvalidPrefixes: []string{"b-"}, // collides with our sqs names
			MaxLength:       42,             // k8s adds 21 characters to the pod name, and 63 is the max before it starts to truncate
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

func multiAPIsValidation() *cr.StructFieldValidation {
	return &cr.StructFieldValidation{
		StructField: "APIs",
		StructListValidation: &cr.StructListValidation{
			Required:         true,
			TreatNullAsEmpty: true,
			StructValidation: &cr.StructValidation{
				StructFieldValidations: []*cr.StructFieldValidation{
					{
						StructField: "Name",
						StringValidation: &cr.StringValidation{
							Required:   true,
							AllowEmpty: false,
						},
					},
					{
						StructField: "Weight",
						Int32Validation: &cr.Int32Validation{
							Required:             true,
							GreaterThanOrEqualTo: pointer.Int32(0),
							LessThanOrEqualTo:    pointer.Int32(100),
						},
					},
					{
						StructField:    "Shadow",
						BoolValidation: &cr.BoolValidation{},
					},
				},
			},
		},
	}
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
						Required:            true,
						AllowedValues:       userconfig.PredictorTypeStrings(),
						HiddenAllowedValues: []string{"onnx"},
					},
					Parser: func(str string) (interface{}, error) {
						if str == "onnx" {
							return nil, ErrorInvalidONNXPredictorType()
						}
						return userconfig.PredictorTypeFromString(str), nil
					},
				},
				{
					StructField: "Path",
					StringValidation: &cr.StringValidation{
						Required: true,
						Suffix:   ".py",
					},
				},
				{
					StructField: "ProtobufPath",
					StringPtrValidation: &cr.StringPtrValidation{
						Default:                   nil,
						AllowExplicitNull:         true,
						AlphaNumericDotUnderscore: true,
						Suffix:                    ".proto",
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
					StructField: "ShmSize",
					StringPtrValidation: &cr.StringPtrValidation{
						Default:           nil,
						AllowExplicitNull: true,
					},
					Parser: k8s.QuantityParser(&k8s.QuantityValidation{}),
				},
				{
					StructField: "LogLevel",
					StringValidation: &cr.StringValidation{
						Default:       "info",
						AllowedValues: userconfig.LogLevelTypes(),
					},
					Parser: func(str string) (interface{}, error) {
						return userconfig.LogLevelFromString(str), nil
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
				multiModelValidation("Models"),
				multiModelValidation("MultiModelReloading"),
				serverSideBatchingValidation(),
				dependencyPathValidation(),
			},
		},
	}
}

func taskDefinitionValidation() *cr.StructFieldValidation {
	return &cr.StructFieldValidation{
		StructField: "TaskDefinition",
		StructValidation: &cr.StructValidation{
			Required: true,
			StructFieldValidations: []*cr.StructFieldValidation{
				{
					StructField: "Path",
					StringValidation: &cr.StringValidation{
						Required: true,
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
					StructField: "LogLevel",
					StringValidation: &cr.StringValidation{
						Default:       "info",
						AllowedValues: userconfig.LogLevelTypes(),
					},
					Parser: func(str string) (interface{}, error) {
						return userconfig.LogLevelFromString(str), nil
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
				dependencyPathValidation(),
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
			},
		},
	}
}

func computeValidation() *cr.StructFieldValidation {
	return &cr.StructFieldValidation{
		StructField: "Compute",
		StructValidation: &cr.StructValidation{
			StructFieldValidations: []*cr.StructFieldValidation{
				{
					StructField: "CPU",
					StringPtrValidation: &cr.StringPtrValidation{
						Default:           pointer.String("200m"),
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
				{
					StructField: "Selector",
					StringPtrValidation: &cr.StringPtrValidation{
						Default:                    nil,
						AllowExplicitNull:          true,
						AlphaNumericDashUnderscore: true,
					},
				},
			},
		},
	}
}

func autoscalingValidation() *cr.StructFieldValidation {
	return &cr.StructFieldValidation{
		StructField: "Autoscaling",
		StructValidation: &cr.StructValidation{
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
					StructField: "MaxReplicaConcurrency",
					Int64Validation: &cr.Int64Validation{
						Default:     consts.DefaultMaxReplicaConcurrency,
						GreaterThan: pointer.Int64(0),
						// our configured nginx can theoretically accept up to 32768 connections, but during testing,
						// it has been observed that the number is just slightly lower, so it has been offset by 2678
						LessThanOrEqualTo: pointer.Int64(30000),
					},
				},
				{
					StructField: "TargetReplicaConcurrency",
					Float64PtrValidation: &cr.Float64PtrValidation{
						GreaterThan: pointer.Float64(0),
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

func updateStrategyValidation() *cr.StructFieldValidation {
	return &cr.StructFieldValidation{
		StructField: "UpdateStrategy",
		StructValidation: &cr.StructValidation{
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

func multiModelValidation(fieldName string) *cr.StructFieldValidation {
	return &cr.StructFieldValidation{
		StructField: fieldName,
		StructValidation: &cr.StructValidation{
			Required:   false,
			DefaultNil: true,
			StructFieldValidations: []*cr.StructFieldValidation{
				{
					StructField: "Path",
					StringPtrValidation: &cr.StringPtrValidation{
						Required:  false,
						Validator: checkForInvalidBucketScheme,
					},
				},
				multiModelPathsValidation(),
				{
					StructField: "Dir",
					StringPtrValidation: &cr.StringPtrValidation{
						Required:  false,
						Validator: checkForInvalidBucketScheme,
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
						StructField: "Path",
						StringValidation: &cr.StringValidation{
							Required:   true,
							AllowEmpty: false,
							Validator:  checkForInvalidBucketScheme,
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

func serverSideBatchingValidation() *cr.StructFieldValidation {
	return &cr.StructFieldValidation{
		StructField: "ServerSideBatching",
		StructValidation: &cr.StructValidation{
			Required:          false,
			DefaultNil:        true,
			AllowExplicitNull: true,
			StructFieldValidations: []*cr.StructFieldValidation{
				{
					StructField: "MaxBatchSize",
					Int32Validation: &cr.Int32Validation{
						Required:             true,
						GreaterThanOrEqualTo: pointer.Int32(2),
						LessThanOrEqualTo:    pointer.Int32(1024), // this is an arbitrary limit
					},
				},
				{
					StructField: "BatchInterval",
					StringValidation: &cr.StringValidation{
						Required: true,
					},
					Parser: cr.DurationParser(&cr.DurationValidation{
						GreaterThan: pointer.Duration(libtime.MustParseDuration("0s")),
					}),
				},
			},
		},
	}
}

func dependencyPathValidation() *cr.StructFieldValidation {
	return &cr.StructFieldValidation{
		StructField: "Dependencies",
		StructValidation: &cr.StructValidation{
			Required: false,
			StructFieldValidations: []*cr.StructFieldValidation{
				{
					StructField: "Pip",
					StringValidation: &cr.StringValidation{
						Default: "requirements.txt",
					},
				},
				{
					StructField: "Conda",
					StringValidation: &cr.StringValidation{
						Default: "conda-packages.txt",
					},
				},
				{
					StructField: "Shell",
					StringValidation: &cr.StringValidation{
						Default: "dependencies.sh",
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

func ExtractAPIConfigs(configBytes []byte, configFileName string) ([]userconfig.API, error) {
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
			return nil, errors.Append(err, fmt.Sprintf("\n\napi configuration schema can be found at https://docs.cortex.dev/v/%s/", consts.CortexVersionMinor))
		}

		errs = cr.Struct(&api, data, apiValidation(resourceStruct))
		if errors.HasError(errs) {
			name, _ := data[userconfig.NameKey].(string)
			kindString, _ := data[userconfig.KindKey].(string)
			kind := userconfig.KindFromString(kindString)
			err = errors.Wrap(errors.FirstError(errs...), userconfig.IdentifyAPI(configFileName, name, kind, i))
			return nil, errors.Append(err, fmt.Sprintf("\n\napi configuration schema can be found at https://docs.cortex.dev/v/%s/", consts.CortexVersionMinor))
		}
		api.Index = i
		api.FileName = configFileName

		interfaceMap, ok := cast.JSONMarshallable(data)
		if !ok {
			return nil, errors.ErrorUnexpected("unable to cast api spec to json") // unexpected
		}

		api.SubmittedAPISpec = interfaceMap

		if resourceStruct.Kind == userconfig.RealtimeAPIKind ||
			resourceStruct.Kind == userconfig.BatchAPIKind ||
			resourceStruct.Kind == userconfig.TaskAPIKind ||
			resourceStruct.Kind == userconfig.AsyncAPIKind {
			api.ApplyDefaultDockerPaths()
		}

		apis[i] = api
	}

	return apis, nil
}

func ValidateAPI(
	api *userconfig.API,
	models *[]CuratedModelResource,
	projectFiles ProjectFiles,
	awsClient *aws.Client,
	k8sClient *k8s.Client,
) error {

	// if models is nil, we need to set it to an empty slice to avoid nil pointer exceptions
	if models == nil {
		models = &[]CuratedModelResource{}
	}

	if api.Networking.Endpoint == nil && (api.Predictor == nil || (api.Predictor != nil && api.Predictor.ProtobufPath == nil)) {
		api.Networking.Endpoint = pointer.String("/" + api.Name)
	}

	switch api.Kind {
	case userconfig.TaskAPIKind:
		if err := validateTaskDefinition(api, projectFiles, awsClient, k8sClient); err != nil {
			return errors.Wrap(err, userconfig.TaskDefinitionKey)
		}
	default:
		if err := validatePredictor(api, models, projectFiles, awsClient, k8sClient); err != nil {
			if errors.GetKind(err) == ErrProtoInvalidNetworkingEndpoint {
				return errors.Wrap(err, userconfig.NetworkingKey, userconfig.EndpointKey)
			}
			return errors.Wrap(err, userconfig.PredictorKey)
		}
	}

	if api.Autoscaling != nil {
		if err := validateAutoscaling(api); err != nil {
			return errors.Wrap(err, userconfig.AutoscalingKey)
		}
	}

	if err := validateCompute(api); err != nil {
		return errors.Wrap(err, userconfig.ComputeKey)
	}

	if api.UpdateStrategy != nil {
		if err := validateUpdateStrategy(api.UpdateStrategy); err != nil {
			return errors.Wrap(err, userconfig.UpdateStrategyKey)
		}
	}

	if api.Predictor != nil && api.Predictor.ShmSize != nil && api.Compute.Mem != nil {
		if api.Predictor.ShmSize.Cmp(api.Compute.Mem.Quantity) > 0 {
			return ErrorShmSizeCannotExceedMem(*api.Predictor.ShmSize, *api.Compute.Mem)
		}
	}

	return nil
}

func validateTaskDefinition(
	api *userconfig.API,
	projectFiles ProjectFiles,
	awsClient *aws.Client,
	k8sClient *k8s.Client,
) error {
	taskDefinition := api.TaskDefinition

	if err := validateDockerImagePath(taskDefinition.Image, awsClient, k8sClient); err != nil {
		return errors.Wrap(err, userconfig.ImageKey)
	}

	for key := range taskDefinition.Env {
		if strings.HasPrefix(key, "CORTEX_") {
			return errors.Wrap(ErrorCortexPrefixedEnvVarNotAllowed(), userconfig.EnvKey, key)
		}
	}

	if !projectFiles.HasFile(taskDefinition.Path) {
		return errors.Wrap(files.ErrorFileDoesNotExist(taskDefinition.Path), userconfig.PathKey)
	}

	if taskDefinition.PythonPath != nil {
		if !projectFiles.HasDir(*taskDefinition.PythonPath) {
			return errors.Wrap(
				ErrorPythonPathNotFound(*taskDefinition.PythonPath),
				userconfig.PythonPathKey,
			)
		}
	}

	return nil
}

func ValidateTrafficSplitter(api *userconfig.API) error {
	if api.Networking.Endpoint == nil {
		api.Networking.Endpoint = pointer.String("/" + api.Name)
	}
	if err := verifyTotalWeight(api.APIs); err != nil {
		return err
	}
	if err := areTrafficSplitterAPIsUnique(api.APIs); err != nil {
		return err
	}

	hasShadow := false
	for _, api := range api.APIs {
		if api.Shadow {
			if hasShadow {
				return ErrorOneShadowPerTrafficSplitter()
			}
			hasShadow = true
		}
	}

	return nil
}

func validatePredictor(
	api *userconfig.API,
	models *[]CuratedModelResource,
	projectFiles ProjectFiles,
	awsClient *aws.Client,
	k8sClient *k8s.Client,
) error {
	predictor := api.Predictor

	if !projectFiles.HasFile(predictor.Path) {
		return errors.Wrap(files.ErrorFileDoesNotExist(predictor.Path), userconfig.PathKey)
	}

	if predictor.PythonPath != nil {
		if err := validatePythonPath(predictor, projectFiles); err != nil {
			return errors.Wrap(err, userconfig.PythonPathKey)
		}
	}

	if predictor.IsGRPC() {
		if api.Kind != userconfig.RealtimeAPIKind {
			return ErrorKeyIsNotSupportedForKind(userconfig.ProtobufPathKey, api.Kind)
		}

		if err := validateProtobufPath(api, projectFiles); err != nil {
			return err
		}
	}

	if err := validateMultiModelsFields(api); err != nil {
		return err
	}

	switch predictor.Type {
	case userconfig.PythonPredictorType:
		if err := validatePythonPredictor(api, models, awsClient); err != nil {
			return err
		}
	case userconfig.TensorFlowPredictorType:
		if err := validateTensorFlowPredictor(api, models, awsClient); err != nil {
			return err
		}
		if err := validateDockerImagePath(predictor.TensorFlowServingImage, awsClient, k8sClient); err != nil {
			return errors.Wrap(err, userconfig.TensorFlowServingImageKey)
		}
	}

	if api.Kind == userconfig.BatchAPIKind || api.Kind == userconfig.AsyncAPIKind {
		if predictor.MultiModelReloading != nil {
			return ErrorKeyIsNotSupportedForKind(userconfig.MultiModelReloadingKey, api.Kind)
		}

		if predictor.ServerSideBatching != nil {
			return ErrorKeyIsNotSupportedForKind(userconfig.ServerSideBatchingKey, api.Kind)
		}

		if predictor.ProcessesPerReplica > 1 {
			return ErrorKeyIsNotSupportedForKind(userconfig.ProcessesPerReplicaKey, api.Kind)
		}

		if predictor.ThreadsPerProcess > 1 {
			return ErrorKeyIsNotSupportedForKind(userconfig.ThreadsPerProcessKey, api.Kind)
		}
	}

	if err := validateDockerImagePath(predictor.Image, awsClient, k8sClient); err != nil {
		return errors.Wrap(err, userconfig.ImageKey)
	}

	for key := range predictor.Env {
		if strings.HasPrefix(key, "CORTEX_") {
			return errors.Wrap(ErrorCortexPrefixedEnvVarNotAllowed(), userconfig.EnvKey, key)
		}
	}

	return nil
}

func validateMultiModelsFields(api *userconfig.API) error {
	predictor := api.Predictor

	var models *userconfig.MultiModels
	if api.Predictor.Models != nil {
		if api.Predictor.Type == userconfig.PythonPredictorType {
			return ErrorFieldNotSupportedByPredictorType(userconfig.ModelsKey, api.Predictor.Type)
		}
		models = api.Predictor.Models
	}
	if api.Predictor.MultiModelReloading != nil {
		if api.Predictor.Type != userconfig.PythonPredictorType {
			return ErrorFieldNotSupportedByPredictorType(userconfig.MultiModelReloadingKey, api.Predictor.Type)
		}
		models = api.Predictor.MultiModelReloading
	}

	if models == nil {
		if api.Predictor.Type != userconfig.PythonPredictorType {
			return ErrorFieldMustBeDefinedForPredictorType(userconfig.ModelsKey, api.Predictor.Type)
		}
		return nil
	}

	if models.Path == nil && len(models.Paths) == 0 && models.Dir == nil {
		return errors.Wrap(ErrorSpecifyOnlyOneField(userconfig.ModelsPathKey, userconfig.ModelsPathsKey, userconfig.ModelsDirKey), userconfig.ModelsKey)
	}
	if models.Path != nil && len(models.Paths) > 0 && models.Dir != nil {
		return errors.Wrap(ErrorSpecifyOnlyOneField(userconfig.ModelsPathKey, userconfig.ModelsPathsKey, userconfig.ModelsDirKey), userconfig.ModelsKey)
	}

	if models.Path != nil && len(models.Paths) > 0 {
		return errors.Wrap(ErrorConflictingFields(userconfig.ModelsPathKey, userconfig.ModelsPathsKey), userconfig.ModelsKey)
	}
	if models.Dir != nil && len(models.Paths) > 0 {
		return errors.Wrap(ErrorConflictingFields(userconfig.ModelsPathsKey, userconfig.ModelsDirKey), userconfig.ModelsKey)
	}
	if models.Dir != nil && models.Path != nil {
		return errors.Wrap(ErrorConflictingFields(userconfig.ModelsPathKey, userconfig.ModelsDirKey), userconfig.ModelsKey)
	}

	if models.CacheSize != nil && api.Kind != userconfig.RealtimeAPIKind {
		return errors.Wrap(ErrorKeyIsNotSupportedForKind(userconfig.ModelsCacheSizeKey, api.Kind), userconfig.ModelsKey)
	}
	if models.DiskCacheSize != nil && api.Kind != userconfig.RealtimeAPIKind {
		return errors.Wrap(ErrorKeyIsNotSupportedForKind(userconfig.ModelsDiskCacheSizeKey, api.Kind), userconfig.ModelsKey)
	}

	if (models.CacheSize == nil && models.DiskCacheSize != nil) ||
		(models.CacheSize != nil && models.DiskCacheSize == nil) {
		return errors.Wrap(ErrorSpecifyAllOrNone(userconfig.ModelsCacheSizeKey, userconfig.ModelsDiskCacheSizeKey), userconfig.ModelsKey)
	}

	if models.CacheSize != nil && models.DiskCacheSize != nil {
		if *models.CacheSize > *models.DiskCacheSize {
			return errors.Wrap(ErrorConfigGreaterThanOtherConfig(userconfig.ModelsCacheSizeKey, *models.CacheSize, userconfig.ModelsDiskCacheSizeKey, *models.DiskCacheSize), userconfig.ModelsKey)
		}

		if predictor.ProcessesPerReplica > 1 {
			return ErrorModelCachingNotSupportedWhenMultiprocessingEnabled(predictor.ProcessesPerReplica)
		}
	}

	return nil
}

func validatePythonPredictor(api *userconfig.API, models *[]CuratedModelResource, awsClient *aws.Client) error {
	predictor := api.Predictor

	if predictor.Models != nil {
		return ErrorFieldNotSupportedByPredictorType(userconfig.ModelsKey, predictor.Type)
	}

	if predictor.ServerSideBatching != nil {
		if predictor.ServerSideBatching.MaxBatchSize != predictor.ThreadsPerProcess {
			return ErrorConcurrencyMismatchServerSideBatchingPython(
				predictor.ServerSideBatching.MaxBatchSize,
				predictor.ThreadsPerProcess,
			)
		}
	}
	if predictor.TensorFlowServingImage != "" {
		return ErrorFieldNotSupportedByPredictorType(userconfig.TensorFlowServingImageKey, predictor.Type)
	}

	if predictor.MultiModelReloading == nil {
		return nil
	}
	mmr := predictor.MultiModelReloading
	if mmr.SignatureKey != nil {
		return errors.Wrap(ErrorFieldNotSupportedByPredictorType(userconfig.ModelsSignatureKeyKey, predictor.Type), userconfig.MultiModelReloadingKey)
	}

	hasSingleModel := mmr.Path != nil
	hasMultiModels := !hasSingleModel

	var modelWrapError func(error) error
	var modelResources []userconfig.ModelResource

	if hasSingleModel {
		modelWrapError = func(err error) error {
			return errors.Wrap(err, userconfig.MultiModelReloadingKey, userconfig.ModelsPathKey)
		}
		modelResources = []userconfig.ModelResource{
			{
				Name: consts.SingleModelName,
				Path: *mmr.Path,
			},
		}
		*mmr.Path = s.EnsureSuffix(*mmr.Path, "/")
	}
	if hasMultiModels {
		if mmr.SignatureKey != nil {
			return errors.Wrap(ErrorFieldNotSupportedByPredictorType(userconfig.ModelsSignatureKeyKey, predictor.Type), userconfig.MultiModelReloadingKey)
		}

		if len(mmr.Paths) > 0 {
			modelWrapError = func(err error) error {
				return errors.Wrap(err, userconfig.MultiModelReloadingKey, userconfig.ModelsPathsKey)
			}

			for _, path := range mmr.Paths {
				if path.SignatureKey != nil {
					return errors.Wrap(
						ErrorFieldNotSupportedByPredictorType(userconfig.ModelsSignatureKeyKey, predictor.Type),
						userconfig.MultiModelReloadingKey,
						userconfig.ModelsKey,
						userconfig.ModelsPathsKey,
						path.Name,
					)
				}
				(*path).Path = s.EnsureSuffix((*path).Path, "/")
				modelResources = append(modelResources, *path)
			}
		}

		if mmr.Dir != nil {
			modelWrapError = func(err error) error {
				return errors.Wrap(err, userconfig.MultiModelReloadingKey, userconfig.ModelsDirKey)
			}
		}
	}

	var err error
	if hasMultiModels && mmr.Dir != nil {
		*models, err = validateDirModels(*mmr.Dir, nil, awsClient, generateErrorForPredictorTypeFn(api), nil)
	} else {
		*models, err = validateModels(modelResources, nil, awsClient, generateErrorForPredictorTypeFn(api), nil)
	}
	if err != nil {
		return modelWrapError(err)
	}

	if hasMultiModels {
		for _, model := range *models {
			if model.Name == consts.SingleModelName {
				return modelWrapError(ErrorReservedModelName(model.Name))
			}
		}
	}

	if err := checkDuplicateModelNames(*models); err != nil {
		return modelWrapError(err)
	}

	return nil
}

func validateTensorFlowPredictor(api *userconfig.API, models *[]CuratedModelResource, awsClient *aws.Client) error {
	predictor := api.Predictor

	if predictor.ServerSideBatching != nil {
		if api.Compute.Inf == 0 && predictor.ServerSideBatching.MaxBatchSize > predictor.ProcessesPerReplica*predictor.ThreadsPerProcess {
			return ErrorInsufficientBatchConcurrencyLevel(predictor.ServerSideBatching.MaxBatchSize, predictor.ProcessesPerReplica, predictor.ThreadsPerProcess)
		}
		if api.Compute.Inf > 0 && predictor.ServerSideBatching.MaxBatchSize > predictor.ThreadsPerProcess {
			return ErrorInsufficientBatchConcurrencyLevelInf(predictor.ServerSideBatching.MaxBatchSize, predictor.ThreadsPerProcess)
		}
	}

	if predictor.MultiModelReloading != nil {
		return ErrorFieldNotSupportedByPredictorType(userconfig.MultiModelReloadingKey, userconfig.PythonPredictorType)
	}

	hasSingleModel := predictor.Models.Path != nil
	hasMultiModels := !hasSingleModel

	var modelWrapError func(error) error
	var modelResources []userconfig.ModelResource

	if hasSingleModel {
		modelWrapError = func(err error) error {
			return errors.Wrap(err, userconfig.ModelsPathKey)
		}
		modelResources = []userconfig.ModelResource{
			{
				Name:         consts.SingleModelName,
				Path:         *predictor.Models.Path,
				SignatureKey: predictor.Models.SignatureKey,
			},
		}
		*predictor.Models.Path = s.EnsureSuffix(*predictor.Models.Path, "/")
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
				(*path).Path = s.EnsureSuffix((*path).Path, "/")
				modelResources = append(modelResources, *path)
			}
		}

		if predictor.Models.Dir != nil {
			modelWrapError = func(err error) error {
				return errors.Wrap(err, userconfig.ModelsKey, userconfig.ModelsDirKey)
			}
		}
	}

	var validators []modelValidator
	if api.Compute.Inf == 0 {
		validators = append(validators, tensorflowModelValidator)
	} else {
		validators = append(validators, tensorflowNeuronModelValidator)
	}

	var err error
	if hasMultiModels && predictor.Models.Dir != nil {
		*models, err = validateDirModels(*predictor.Models.Dir, predictor.Models.SignatureKey, awsClient, generateErrorForPredictorTypeFn(api), validators)
	} else {
		*models, err = validateModels(modelResources, predictor.Models.SignatureKey, awsClient, generateErrorForPredictorTypeFn(api), validators)
	}
	if err != nil {
		return modelWrapError(err)
	}

	if hasMultiModels {
		for _, model := range *models {
			if model.Name == consts.SingleModelName {
				return modelWrapError(ErrorReservedModelName(model.Name))
			}
		}
	}

	if err := checkDuplicateModelNames(*models); err != nil {
		return modelWrapError(err)
	}

	return nil
}

func validateProtobufPath(api *userconfig.API, projectFiles ProjectFiles) error {
	apiName := api.Name
	protobufPath := *api.Predictor.ProtobufPath

	if !projectFiles.HasFile(protobufPath) {
		return errors.Wrap(files.ErrorFileDoesNotExist(protobufPath), userconfig.ProtobufPathKey)
	}
	protoBytes, err := projectFiles.GetFile(protobufPath)
	if err != nil {
		return errors.Wrap(err, userconfig.ProtobufPathKey, *api.Predictor.ProtobufPath)
	}

	protoReader := bytes.NewReader(protoBytes)
	parser := pbparser.NewParser(protoReader)
	proto, err := parser.Parse()
	if err != nil {
		return errors.Wrap(errors.WithStack(err), userconfig.ProtobufPathKey, *api.Predictor.ProtobufPath)
	}

	var packageName string
	var serviceName string
	var serviceMethodName string = "Predict"
	var detectedMethodName string

	numServices := 0
	numRPCs := 0
	pbparser.Walk(proto,
		pbparser.WithPackage(func(pkg *pbparser.Package) {
			packageName = pkg.Name
		}),
		pbparser.WithService(func(service *pbparser.Service) {
			numServices++
			serviceName = service.Name
			for _, elem := range service.Elements {
				if s, ok := elem.(*pbparser.RPC); ok {
					numRPCs++
					detectedMethodName = s.Name
				}
			}
		}),
	)

	if numServices > 1 {
		return errors.Wrap(ErrorProtoNumServicesExceeded(numServices), userconfig.ProtobufPathKey, *api.Predictor.ProtobufPath)
	}

	if numRPCs > 1 {
		return errors.Wrap(ErrorProtoNumServiceMethodsExceeded(numRPCs, serviceName), userconfig.ProtobufPathKey, *api.Predictor.ProtobufPath)
	}

	if serviceMethodName != detectedMethodName {
		return errors.Wrap(ErrorProtoInvalidServiceMethod(detectedMethodName, serviceMethodName, serviceName), userconfig.ProtobufPathKey, *api.Predictor.ProtobufPath)
	}

	var requiredPackageName string
	requiredPackageName = strings.ReplaceAll(apiName, "-", "_")

	if api.Predictor.ServerSideBatching != nil {
		return ErrorConflictingFields(userconfig.ProtobufPathKey, userconfig.ServerSideBatchingKey)
	}

	if packageName == "" {
		return errors.Wrap(ErrorProtoMissingPackageName(requiredPackageName), userconfig.ProtobufPathKey, *api.Predictor.ProtobufPath)
	}
	if packageName != requiredPackageName {
		return errors.Wrap(ErrorProtoInvalidPackageName(packageName, requiredPackageName), userconfig.ProtobufPathKey, *api.Predictor.ProtobufPath)
	}

	requiredEndpoint := "/" + requiredPackageName + "." + serviceName + "/" + serviceMethodName
	if api.Networking.Endpoint == nil {
		api.Networking.Endpoint = pointer.String(requiredEndpoint)
	}
	if *api.Networking.Endpoint != requiredEndpoint {
		return ErrorProtoInvalidNetworkingEndpoint(requiredEndpoint)
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

func validateCompute(api *userconfig.API) error {
	compute := api.Compute

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

func validateDockerImagePath(
	image string,
	awsClient *aws.Client,
	k8sClient *k8s.Client,
) error {
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

	dockerAuthStr := docker.NoAuth

	if regex.IsValidECRURL(image) {
		dockerAuthStr, err = docker.AWSAuthConfig(awsClient)
		if err != nil {
			return err
		}
	} else if k8sClient != nil {
		dockerAuthStr, err = getDockerAuthStrFromK8s(dockerClient, k8sClient)
		if err != nil {
			return err
		}
	}

	if err := docker.CheckImageAccessible(dockerClient, image, dockerAuthStr); err != nil {
		return err
	}

	return nil
}

func getDockerAuthStrFromK8s(dockerClient *docker.Client, k8sClient *k8s.Client) (string, error) {
	secretData, err := k8sClient.GetSecretData(_dockerPullSecretName)
	if err != nil {
		return "", err
	}

	// check if the user provided the registry auth secret
	if secretData == nil {
		return docker.NoAuth, nil
	}

	authData, ok := secretData[".dockerconfigjson"]
	if !ok {
		return "", ErrorUnexpectedDockerSecretData("should contain \".dockerconfigjson\" key", secretData)
	}

	var authSecret struct {
		Auths map[string]struct {
			Username string `json:"username"`
			Password string `json:"password"`
		} `json:"auths"`
	}

	err = libjson.Unmarshal(authData, &authSecret)
	if err != nil {
		return "", ErrorUnexpectedDockerSecretData(errors.Message(err), secretData)
	}
	if len(authSecret.Auths) != 1 {
		return "", ErrorUnexpectedDockerSecretData("should contain a single set of credentials", secretData)
	}

	var dockerAuth dockertypes.AuthConfig
	for registryAddress, creds := range authSecret.Auths {
		dockerAuth = dockertypes.AuthConfig{
			Username:      creds.Username,
			Password:      creds.Password,
			ServerAddress: registryAddress,
		}
	}
	if dockerAuth.Username == "" {
		return "", ErrorUnexpectedDockerSecretData("missing username", secretData)
	}
	if dockerAuth.Password == "" {
		return "", ErrorUnexpectedDockerSecretData("missing password", secretData)
	}
	if dockerAuth.ServerAddress == "" {
		return "", ErrorUnexpectedDockerSecretData("missing registry address", secretData)
	}

	_, err = dockerClient.RegistryLogin(context.Background(), dockerAuth)
	if err != nil {
		return "", err
	}

	dockerAuthStr, err := docker.EncodeAuthConfig(dockerAuth)
	if err != nil {
		return "", err
	}

	return dockerAuthStr, nil
}
