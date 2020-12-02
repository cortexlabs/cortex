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
	"context"
	"fmt"
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
	libjson "github.com/cortexlabs/cortex/pkg/lib/json"
	"github.com/cortexlabs/cortex/pkg/lib/k8s"
	libmath "github.com/cortexlabs/cortex/pkg/lib/math"
	"github.com/cortexlabs/cortex/pkg/lib/pointer"
	"github.com/cortexlabs/cortex/pkg/lib/regex"
	s "github.com/cortexlabs/cortex/pkg/lib/strings"
	libtime "github.com/cortexlabs/cortex/pkg/lib/time"
	"github.com/cortexlabs/cortex/pkg/lib/urls"
	"github.com/cortexlabs/cortex/pkg/types"
	"github.com/cortexlabs/cortex/pkg/types/clusterconfig"
	"github.com/cortexlabs/cortex/pkg/types/userconfig"
	"github.com/cortexlabs/yaml"
	dockertypes "github.com/docker/docker/api/types"
	kresource "k8s.io/apimachinery/pkg/api/resource"
)

var AutoscalingTickInterval = 10 * time.Second

const _dockerPullSecretName = "registry-credentials"

func apiValidation(
	provider types.ProviderType,
	resource userconfig.Resource,
	awsClusterConfig *clusterconfig.Config, // should be omitted if running locally
	gcpClusterConfig *clusterconfig.GCPConfig, // should be omitted if running locally
) *cr.StructValidation {

	structFieldValidations := []*cr.StructFieldValidation{}
	switch resource.Kind {
	case userconfig.RealtimeAPIKind:
		structFieldValidations = append(resourceStructValidations,
			predictorValidation(),
			networkingValidation(resource.Kind, awsClusterConfig, gcpClusterConfig),
			computeValidation(provider),
			monitoringValidation(),
			autoscalingValidation(provider),
			updateStrategyValidation(provider),
		)
	case userconfig.BatchAPIKind:
		structFieldValidations = append(resourceStructValidations,
			predictorValidation(),
			networkingValidation(resource.Kind, awsClusterConfig, gcpClusterConfig),
			computeValidation(provider),
		)
	case userconfig.TrafficSplitterKind:
		structFieldValidations = append(resourceStructValidations,
			multiAPIsValidation(),
			networkingValidation(resource.Kind, awsClusterConfig, gcpClusterConfig),
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
				serverSideBatchingValidation(),
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

func networkingValidation(
	kind userconfig.Kind,
	awsClusterConfig *clusterconfig.Config, // should be omitted if running locally
	gcpClusterConfig *clusterconfig.GCPConfig, // should be omitted if running locally
) *cr.StructFieldValidation {

	defaultAPIGatewayType := userconfig.PublicAPIGatewayType
	if awsClusterConfig != nil && awsClusterConfig.APIGatewaySetting == clusterconfig.NoneAPIGatewaySetting {
		defaultAPIGatewayType = userconfig.NoneAPIGatewayType
	}

	structFieldValidation := []*cr.StructFieldValidation{
		{
			StructField: "Endpoint",
			StringPtrValidation: &cr.StringPtrValidation{
				Validator: urls.ValidateEndpoint,
				MaxLength: 1000, // no particular reason other than it works
			},
		},
		{
			StructField: "APIGateway",
			StringValidation: &cr.StringValidation{
				AllowedValues: userconfig.APIGatewayTypeStrings(),
				Default:       defaultAPIGatewayType.String(),
			},
			Parser: func(str string) (interface{}, error) {
				return userconfig.APIGatewayTypeFromString(str), nil
			},
		},
	}
	if kind == userconfig.RealtimeAPIKind {
		structFieldValidation = append(structFieldValidation, &cr.StructFieldValidation{
			StructField: "LocalPort",
			IntPtrValidation: &cr.IntPtrValidation{
				GreaterThan:       pointer.Int(0),
				LessThanOrEqualTo: pointer.Int(math.MaxUint16),
			},
		})
	}
	return &cr.StructFieldValidation{
		StructField: "Networking",
		StructValidation: &cr.StructValidation{
			StructFieldValidations: structFieldValidation,
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
						Default:     consts.DefaultMaxReplicaConcurrency,
						GreaterThan: pointer.Int64(0),
						// our configured nginx can theoretically accept up to 32768 connections, but during testing,
						// it has been observed that the number is just slightly lower, so it has been offset by 2678
						LessThanOrEqualTo: pointer.Int64(30000),
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

var resourceStructValidation = cr.StructValidation{
	AllowExtraFields:       true,
	StructFieldValidations: resourceStructValidations,
}

func ExtractAPIConfigs(
	configBytes []byte,
	provider types.ProviderType,
	configFileName string,
	awsClusterConfig *clusterconfig.Config, // should be omitted if running locally
	gcpClusterConfig *clusterconfig.GCPConfig, // should be omitted if running locally
) ([]userconfig.API, error) {

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
			switch provider {
			case types.LocalProviderType:
				return nil, errors.Append(err, fmt.Sprintf("\n\napi configuration schema for Realtime API can be found at https://docs.cortex.dev/v/%s/deployments/realtime-api/api-configuration", consts.CortexVersionMinor))
			case types.AWSProviderType:
				return nil, errors.Append(err, fmt.Sprintf("\n\napi configuration schema can be found here:\n  → Realtime API: https://docs.cortex.dev/v/%s/deployments/realtime-api/api-configuration\n  → Batch API: https://docs.cortex.dev/v/%s/deployments/batch-api/api-configuration\n  → Traffic Splitter: https://docs.cortex.dev/v/%s/deployments/realtime-api/traffic-splitter", consts.CortexVersionMinor, consts.CortexVersionMinor, consts.CortexVersionMinor))
			case types.GCPProviderType:
				return nil, errors.Append(err, fmt.Sprintf("\n\nto add api configuration schema"))
			}
		}

		if resourceStruct.Kind == userconfig.BatchAPIKind || resourceStruct.Kind == userconfig.TrafficSplitterKind {
			if provider == types.LocalProviderType {
				return nil, errors.Wrap(ErrorKindIsNotSupportedByProvider(resourceStruct.Kind, types.LocalProviderType), userconfig.IdentifyAPI(configFileName, resourceStruct.Name, resourceStruct.Kind, i))
			}
		}

		errs = cr.Struct(&api, data, apiValidation(provider, resourceStruct, awsClusterConfig, gcpClusterConfig))
		if errors.HasError(errs) {
			name, _ := data[userconfig.NameKey].(string)
			kindString, _ := data[userconfig.KindKey].(string)
			kind := userconfig.KindFromString(kindString)
			err = errors.Wrap(errors.FirstError(errs...), userconfig.IdentifyAPI(configFileName, name, kind, i))
			switch kind {
			case userconfig.RealtimeAPIKind:
				return nil, errors.Append(err, fmt.Sprintf("\n\napi configuration schema for Realtime API can be found at https://docs.cortex.dev/v/%s/deployments/realtime-api/api-configuration", consts.CortexVersionMinor))
			case userconfig.BatchAPIKind:
				return nil, errors.Append(err, fmt.Sprintf("\n\napi configuration schema for Batch API can be found at https://docs.cortex.dev/v/%s/deployments/batch-api/api-configuration", consts.CortexVersionMinor))
			case userconfig.TrafficSplitterKind:
				return nil, errors.Append(err, fmt.Sprintf("\n\napi configuration schema for Traffic Splitter can be found at https://docs.cortex.dev/v/%s/deployments/realtime-api/traffic-splitter", consts.CortexVersionMinor))
			}
		}
		api.Index = i
		api.FileName = configFileName

		rawYAMLBytes, err := yaml.Marshal([]map[string]interface{}{data})
		if err != nil {
			return nil, errors.Wrap(err, api.Identify())
		}
		api.RawYAMLBytes = rawYAMLBytes

		if resourceStruct.Kind == userconfig.RealtimeAPIKind || resourceStruct.Kind == userconfig.BatchAPIKind {
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
	providerType types.ProviderType,
	awsClient *aws.Client,
	k8sClient *k8s.Client, // will be nil for local provider
) error {

	if providerType == types.GCPProviderType && api.Kind != userconfig.RealtimeAPIKind {
		return ErrorKindIsNotSupportedByProvider(api.Kind, providerType)
	}

	if api.Networking.Endpoint == nil {
		api.Networking.Endpoint = pointer.String("/" + api.Name)
	}

	if err := validatePredictor(api, models, projectFiles, providerType, awsClient, k8sClient); err != nil {
		return errors.Wrap(err, userconfig.PredictorKey)
	}

	if api.Autoscaling != nil { // should only be nil for local provider
		if err := validateAutoscaling(api); err != nil {
			return errors.Wrap(err, userconfig.AutoscalingKey)
		}
	}

	if err := validateCompute(api, providerType); err != nil {
		return errors.Wrap(err, userconfig.ComputeKey)
	}

	if api.UpdateStrategy != nil { // should only be nil for local provider
		if err := validateUpdateStrategy(api.UpdateStrategy); err != nil {
			return errors.Wrap(err, userconfig.UpdateStrategyKey)
		}
	}

	return nil
}

func ValidateTrafficSplitter(
	api *userconfig.API,
	providerType types.ProviderType,
	awsClient *aws.Client,
) error {

	if providerType == types.LocalProviderType {
		return ErrorKindIsNotSupportedByProvider(userconfig.TrafficSplitterKind, providerType)
	}

	if api.Networking.Endpoint == nil {
		api.Networking.Endpoint = pointer.String("/" + api.Name)
	}
	if err := verifyTotalWeight(api.APIs); err != nil {
		return err
	}
	if err := areTrafficSplitterAPIsUnique(api.APIs); err != nil {
		return err
	}

	return nil
}

func validatePredictor(
	api *userconfig.API,
	models *[]CuratedModelResource,
	projectFiles ProjectFiles,
	providerType types.ProviderType,
	awsClient *aws.Client,
	k8sClient *k8s.Client, // will be nil for local provider
) error {
	predictor := api.Predictor

	if providerType == types.AWSProviderType {
		if predictor.Models != nil && predictor.ModelPath != nil {
			return ErrorConflictingFields(userconfig.ModelPathKey, userconfig.ModelsKey)
		}
		if predictor.Models != nil {
			if err := validateMultiModelsFields(api); err != nil {
				return err
			}
		}
	} else {
		if predictor.Models != nil {
			return ErrorFieldNotSupportedByProvider(userconfig.ModelsKey, providerType)
		}
		if predictor.ModelPath != nil {
			return ErrorFieldNotSupportedByProvider(userconfig.ModelPathKey, providerType)
		}
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
		if err := validateDockerImagePath(predictor.TensorFlowServingImage, providerType, awsClient, k8sClient); err != nil {
			return errors.Wrap(err, userconfig.TensorFlowServingImageKey)
		}
	case userconfig.ONNXPredictorType:
		if err := validateONNXPredictor(predictor, models, providerType, projectFiles, awsClient); err != nil {
			return err
		}
	}

	if api.Kind == userconfig.BatchAPIKind {
		if predictor.ProcessesPerReplica > 1 {
			return ErrorKeyIsNotSupportedForKind(userconfig.ProcessesPerReplicaKey, userconfig.BatchAPIKind)
		}

		if predictor.ThreadsPerProcess > 1 {
			return ErrorKeyIsNotSupportedForKind(userconfig.ThreadsPerProcessKey, userconfig.BatchAPIKind)
		}
	}

	if err := validateDockerImagePath(predictor.Image, providerType, awsClient, k8sClient); err != nil {
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

func validateMultiModelsFields(api *userconfig.API) error {
	predictor := api.Predictor

	if len(predictor.Models.Paths) == 0 && predictor.Models.Dir == nil {
		return errors.Wrap(ErrorSpecifyOneOrTheOther(userconfig.ModelsPathsKey, userconfig.ModelsDirKey), userconfig.ModelsKey)
	}
	if len(predictor.Models.Paths) > 0 && predictor.Models.Dir != nil {
		return errors.Wrap(ErrorConflictingFields(userconfig.ModelsPathsKey, userconfig.ModelsDirKey), userconfig.ModelsKey)
	}

	if predictor.Models.CacheSize != nil && api.Kind != userconfig.RealtimeAPIKind {
		return errors.Wrap(ErrorKeyIsNotSupportedForKind(userconfig.ModelsCacheSizeKey, api.Kind), userconfig.ModelsKey)
	}
	if predictor.Models.DiskCacheSize != nil && api.Kind != userconfig.RealtimeAPIKind {
		return errors.Wrap(ErrorKeyIsNotSupportedForKind(userconfig.ModelsDiskCacheSizeKey, api.Kind), userconfig.ModelsKey)
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
			return ErrorModelCachingNotSupportedWhenMultiprocessingEnabled(predictor.ProcessesPerReplica)
		}
	}

	return nil
}

func validatePythonPredictor(predictor *userconfig.Predictor, models *[]CuratedModelResource, providerType types.ProviderType, projectFiles ProjectFiles, awsClient *aws.Client) error {
	if predictor.SignatureKey != nil {
		return ErrorFieldNotSupportedByPredictorType(userconfig.SignatureKeyKey, predictor.Type)
	}
	if predictor.ServerSideBatching != nil {
		return ErrorFieldNotSupportedByPredictorType(userconfig.ServerSideBatchingKey, predictor.Type)
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
		*predictor.ModelPath = s.EnsureSuffix(*predictor.ModelPath, "/")
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
				(*path).ModelPath = s.EnsureSuffix((*path).ModelPath, "/")
				modelResources = append(modelResources, *path)
			}
		}

		if predictor.Models.Dir != nil {
			modelWrapError = func(err error) error {
				return errors.Wrap(err, userconfig.ModelsKey, userconfig.ModelsDirKey)
			}

			*(predictor.Models.Dir) = s.EnsureSuffix(*(predictor.Models.Dir), "/")

			var err error
			modelResources, err = listModelResourcesFromPath(*predictor.Models.Dir, projectFiles, awsClient)
			if err != nil {
				return modelWrapError(err)
			}
		}
	}
	var err error
	*models, err = modelResourceToCurated(modelResources, projectFiles.ProjectDir())
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

		versions, err := getPythonVersionsFromS3Path(modelResource.ModelPath, awsClientForBucket)
		if err != nil {
			if errors.GetKind(err) == ErrModelPathNotDirectory {
				return errors.Wrap(err, modelName)
			}

			modelSubS3Objects, err := awsClientForBucket.ListS3PathDir(modelResource.ModelPath, false, pointer.Int64(1000))
			if err != nil {
				return errors.Wrap(err, modelName)
			}
			modelSubPaths := aws.ConvertS3ObjectsToKeys(modelSubS3Objects...)

			if err = validatePythonS3ModelDir(modelResource.ModelPath, awsClientForBucket); err != nil {
				return errors.Wrap(errors.Append(err, "\n\n"+ErrorInvalidPythonModelPath(modelResource.ModelPath, modelSubPaths).Error()), modelName)
			}
		}
		modelResource.Versions = versions
	} else {
		if providerType == types.AWSProviderType {
			return ErrorLocalModelPathNotSupportedByAWSProvider()
		}

		versions, err := getPythonVersionsFromLocalPath(modelResource.ModelPath)
		if err != nil {
			if errors.GetKind(err) == ErrModelPathNotDirectory {
				return errors.Wrap(err, modelName)
			}

			modelSubPaths, err := files.ListDirRecursive(modelResource.ModelPath, false)
			if err != nil {
				return errors.Wrap(err, modelName)
			}

			if err = validatePythonLocalModelDir(modelResource.ModelPath); err != nil {
				return errors.Wrap(errors.Append(err, "\n\n"+ErrorInvalidPythonModelPath(modelResource.ModelPath, modelSubPaths).Error()), modelName)
			}
		}
		modelResource.Versions = versions
	}

	return nil
}

func validateTensorFlowPredictor(api *userconfig.API, models *[]CuratedModelResource, providerType types.ProviderType, projectFiles ProjectFiles, awsClient *aws.Client) error {
	predictor := api.Predictor

	if providerType == types.GCPProviderType {
		return ErrorPredictorIsNotSupportedByProvider(predictor.Type, providerType)
	}

	if predictor.ServerSideBatching != nil {
		if api.Compute.Inf == 0 && predictor.ServerSideBatching.MaxBatchSize > predictor.ProcessesPerReplica*predictor.ThreadsPerProcess {
			return ErrorInsufficientBatchConcurrencyLevel(predictor.ServerSideBatching.MaxBatchSize, predictor.ProcessesPerReplica, predictor.ThreadsPerProcess)
		}
		if api.Compute.Inf > 0 && predictor.ServerSideBatching.MaxBatchSize > predictor.ThreadsPerProcess {
			return ErrorInsufficientBatchConcurrencyLevelInf(predictor.ServerSideBatching.MaxBatchSize, predictor.ThreadsPerProcess)
		}
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
				Name:         consts.SingleModelName,
				ModelPath:    *predictor.ModelPath,
				SignatureKey: predictor.SignatureKey,
			},
		}
		*predictor.ModelPath = s.EnsureSuffix(*predictor.ModelPath, "/")
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
				(*path).ModelPath = s.EnsureSuffix((*path).ModelPath, "/")
				modelResources = append(modelResources, *path)
			}
		}

		if predictor.Models.Dir != nil {
			modelWrapError = func(err error) error {
				return errors.Wrap(err, userconfig.ModelsKey, userconfig.ModelsDirKey)
			}

			*(predictor.Models.Dir) = s.EnsureSuffix(*(predictor.Models.Dir), "/")

			var err error
			modelResources, err = listModelResourcesFromPath(*predictor.Models.Dir, projectFiles, awsClient)
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
	*models, err = modelResourceToCurated(modelResources, projectFiles.ProjectDir())
	if err != nil {
		return err
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
			if errors.GetKind(err) == ErrModelPathNotDirectory {
				return errors.Wrap(err, modelName)
			}

			modelSubS3Objects, err := awsClientForBucket.ListS3PathDir(modelResource.ModelPath, false, pointer.Int64(1000))
			if err != nil {
				return errors.Wrap(err, modelName)
			}
			modelSubPaths := aws.ConvertS3ObjectsToKeys(modelSubS3Objects...)

			if err = validateTFServingS3ModelDir(modelResource.ModelPath, isNeuronExport, awsClientForBucket); err != nil {
				if errors.GetKind(err) != ErrInvalidTensorFlowModelPath {
					return errors.Wrap(errors.Append(err, "\n\n"+ErrorInvalidTensorFlowModelPath(modelResource.ModelPath, isNeuronExport, modelSubPaths).Error()), modelName)
				}
				return errors.Wrap(ErrorInvalidTensorFlowModelPath(modelResource.ModelPath, isNeuronExport, modelSubPaths), modelName)
			}
		}
		modelResource.Versions = versions
	} else {
		if providerType == types.AWSProviderType {
			return ErrorLocalModelPathNotSupportedByAWSProvider()
		}

		versions, err := getTFServingVersionsFromLocalPath(modelResource.ModelPath)
		if err != nil {
			if errors.GetKind(err) == ErrModelPathNotDirectory {
				return errors.Wrap(err, modelName)
			}

			modelSubPaths, err := files.ListDirRecursive(modelResource.ModelPath, false)
			if err != nil {
				return errors.Wrap(err, modelName)
			}

			if err = validateTFServingLocalModelDir(modelResource.ModelPath); err != nil {
				if errors.GetKind(err) != ErrInvalidTensorFlowModelPath {
					return errors.Wrap(errors.Append(err, "\n\n"+ErrorInvalidTensorFlowModelPath(modelResource.ModelPath, false, modelSubPaths).Error()), modelName)
				}
				return errors.Wrap(ErrorInvalidTensorFlowModelPath(modelResource.ModelPath, false, modelSubPaths), modelName)
			}
		}
		modelResource.Versions = versions
	}

	return nil
}

func validateONNXPredictor(predictor *userconfig.Predictor, models *[]CuratedModelResource, providerType types.ProviderType, projectFiles ProjectFiles, awsClient *aws.Client) error {
	if providerType == types.GCPProviderType {
		return ErrorPredictorIsNotSupportedByProvider(predictor.Type, providerType)
	}

	if predictor.SignatureKey != nil {
		return ErrorFieldNotSupportedByPredictorType(userconfig.SignatureKeyKey, predictor.Type)
	}
	if predictor.ServerSideBatching != nil {
		return ErrorFieldNotSupportedByPredictorType(userconfig.ServerSideBatchingKey, predictor.Type)
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
		*predictor.ModelPath = s.EnsureSuffix(*predictor.ModelPath, "/")
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
				(*path).ModelPath = s.EnsureSuffix((*path).ModelPath, "/")
				modelResources = append(modelResources, *path)
			}
		}

		if predictor.Models.Dir != nil {
			modelWrapError = func(err error) error {
				return errors.Wrap(err, userconfig.ModelsKey, userconfig.ModelsDirKey)
			}

			*(predictor.Models.Dir) = s.EnsureSuffix(*(predictor.Models.Dir), "/")

			var err error
			modelResources, err = listModelResourcesFromPath(*predictor.Models.Dir, projectFiles, awsClient)
			if err != nil {
				return modelWrapError(err)
			}
		}
	}
	var err error
	*models, err = modelResourceToCurated(modelResources, projectFiles.ProjectDir())
	if err != nil {
		return err
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
			if errors.GetKind(err) == ErrModelPathNotDirectory {
				return errors.Wrap(err, modelName)
			}

			modelSubS3Objects, err := awsClientForBucket.ListS3PathDir(modelResource.ModelPath, false, pointer.Int64(1000))
			if err != nil {
				return errors.Wrap(err, modelName)
			}
			modelSubPaths := aws.ConvertS3ObjectsToKeys(modelSubS3Objects...)

			if err := validateONNXS3ModelDir(modelResource.ModelPath, awsClientForBucket); err != nil {
				if errors.GetKind(err) != ErrInvalidONNXModelPath {
					return errors.Wrap(errors.Append(err, "\n\n"+ErrorInvalidONNXModelPath(modelResource.ModelPath, modelSubPaths).Error()), modelName)
				}
				return errors.Wrap(ErrorInvalidONNXModelPath(modelResource.ModelPath, modelSubPaths), modelName)
			}
		}
		modelResource.Versions = versions
	} else {
		if providerType == types.AWSProviderType {
			return ErrorLocalModelPathNotSupportedByAWSProvider()
		}

		versions, err := getONNXVersionsFromLocalPath(modelResource.ModelPath)
		if err != nil {
			if errors.GetKind(err) == ErrModelPathNotDirectory {
				return errors.Wrap(err, modelName)
			}

			modelSubPaths, err := files.ListDirRecursive(modelResource.ModelPath, false)
			if err != nil {
				return errors.Wrap(err, modelName)
			}

			if err := validateONNXLocalModelDir(modelResource.ModelPath); err != nil {
				if errors.GetKind(err) != ErrInvalidONNXModelPath {
					return errors.Wrap(errors.Append(err, "\n\n"+ErrorInvalidONNXModelPath(modelResource.ModelPath, modelSubPaths).Error()), modelName)
				}
				return errors.Wrap(ErrorInvalidONNXModelPath(modelResource.ModelPath, modelSubPaths), modelName)
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

	if compute.Inf > 0 && providerType != types.AWSProviderType {
		return ErrorUnsupportedComputeResourceForProvider(userconfig.InfKey, providerType)
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

func validateDockerImagePath(
	image string,
	providerType types.ProviderType,
	awsClient *aws.Client,
	k8sClient *k8s.Client, // will be nil for local provider)
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

	if providerType == types.LocalProviderType {
		// short circuit if the image is already available locally
		if err := docker.CheckImageExistsLocally(dockerClient, image); err == nil {
			return nil
		}
	}

	dockerAuthStr := docker.NoAuth

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

		dockerAuthStr, err = docker.AWSAuthConfig(awsClient)
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
	} else if k8sClient != nil {
		dockerAuthStr, err = getDockerAuthStrFromK8s(dockerClient, k8sClient)
		if err != nil {
			return err
		}
	}

	if err := docker.CheckImageAccessible(dockerClient, image, dockerAuthStr, providerType); err != nil {
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
