/*
Copyright 2022 Cortex Labs, Inc.

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
	"strings"
	"time"

	"github.com/cortexlabs/cortex/pkg/consts"
	"github.com/cortexlabs/cortex/pkg/lib/aws"
	"github.com/cortexlabs/cortex/pkg/lib/cast"
	cr "github.com/cortexlabs/cortex/pkg/lib/configreader"
	"github.com/cortexlabs/cortex/pkg/lib/docker"
	"github.com/cortexlabs/cortex/pkg/lib/errors"
	libjson "github.com/cortexlabs/cortex/pkg/lib/json"
	"github.com/cortexlabs/cortex/pkg/lib/k8s"
	"github.com/cortexlabs/cortex/pkg/lib/pointer"
	"github.com/cortexlabs/cortex/pkg/lib/regex"
	"github.com/cortexlabs/cortex/pkg/lib/slices"
	s "github.com/cortexlabs/cortex/pkg/lib/strings"
	libtime "github.com/cortexlabs/cortex/pkg/lib/time"
	"github.com/cortexlabs/cortex/pkg/lib/urls"
	"github.com/cortexlabs/cortex/pkg/types/userconfig"
	dockertypes "github.com/docker/docker/api/types"
	kresource "k8s.io/apimachinery/pkg/api/resource"
)

var AutoscalingTickInterval = 10 * time.Second

const _dockerPullSecretName = "registry-credentials"

func apiValidation(resource userconfig.Resource) *cr.StructValidation {
	var structFieldValidations []*cr.StructFieldValidation

	switch resource.Kind {
	case userconfig.RealtimeAPIKind:
		structFieldValidations = append(resourceStructValidations,
			podValidation(userconfig.RealtimeAPIKind),
			nodegroupsValidation(),
			networkingValidation(),
			autoscalingValidation(),
			updateStrategyValidation(),
		)
	case userconfig.AsyncAPIKind:
		structFieldValidations = append(resourceStructValidations,
			podValidation(userconfig.AsyncAPIKind),
			nodegroupsValidation(),
			networkingValidation(),
			autoscalingValidation(),
			updateStrategyValidation(),
		)
	case userconfig.BatchAPIKind:
		structFieldValidations = append(resourceStructValidations,
			podValidation(userconfig.BatchAPIKind),
			nodegroupsValidation(),
			networkingValidation(),
		)
	case userconfig.TaskAPIKind:
		structFieldValidations = append(resourceStructValidations,
			podValidation(userconfig.TaskAPIKind),
			nodegroupsValidation(),
			networkingValidation(),
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

func podValidation(kind userconfig.Kind) *cr.StructFieldValidation {
	validation := &cr.StructFieldValidation{
		StructField: "Pod",
		StructValidation: &cr.StructValidation{
			StructFieldValidations: []*cr.StructFieldValidation{
				{
					StructField: "Port",
					Int32PtrValidation: &cr.Int32PtrValidation{
						Required:          false,
						Default:           nil, // it's a pointer because it's not required for the task API
						AllowExplicitNull: true,
						GreaterThan:       pointer.Int32(0),
						LessThanOrEqualTo: pointer.Int32(65535),
						DisallowedValues:  consts.ReservedContainerPorts,
					},
				},
				containersValidation(kind),
			},
		},
	}

	if kind == userconfig.RealtimeAPIKind {
		validation.StructValidation.StructFieldValidations = append(validation.StructValidation.StructFieldValidations,
			&cr.StructFieldValidation{
				StructField: "MaxQueueLength",
				Int64Validation: &cr.Int64Validation{
					Default:     consts.DefaultMaxQueueLength,
					GreaterThan: pointer.Int64(0),
					// the proxy can theoretically accept up to 32768 connections, but during testing,
					// it has been observed that the number is just slightly lower, so it has been offset by 2678
					LessThanOrEqualTo: pointer.Int64(30000),
				},
			},
			&cr.StructFieldValidation{
				StructField: "MaxConcurrency",
				Int64Validation: &cr.Int64Validation{
					Default:     consts.DefaultMaxConcurrency,
					GreaterThan: pointer.Int64(0),
					// the proxy can theoretically accept up to 32768 connections, but during testing,
					// it has been observed that the number is just slightly lower, so it has been offset by 2678
					LessThanOrEqualTo: pointer.Int64(30000),
				},
			},
		)
	}

	if kind == userconfig.AsyncAPIKind {
		validation.StructValidation.StructFieldValidations = append(validation.StructValidation.StructFieldValidations,
			&cr.StructFieldValidation{
				StructField: "MaxConcurrency",
				Int64Validation: &cr.Int64Validation{
					Default:           consts.DefaultMaxConcurrency,
					GreaterThan:       pointer.Int64(0),
					LessThanOrEqualTo: pointer.Int64(100),
				},
			},
		)
	}

	return validation
}

func containersValidation(kind userconfig.Kind) *cr.StructFieldValidation {
	validations := []*cr.StructFieldValidation{
		{
			StructField: "Name",
			StringValidation: &cr.StringValidation{
				Required:         true,
				AllowEmpty:       false,
				DNS1035:          true,
				MaxLength:        63,
				DisallowedValues: consts.ReservedContainerNames,
			},
		},
		{
			StructField: "Image",
			StringValidation: &cr.StringValidation{
				Required:    true,
				AllowEmpty:  false,
				DockerImage: true,
			},
		},
		{
			StructField: "Env",
			StringMapValidation: &cr.StringMapValidation{
				Required:   false,
				Default:    map[string]string{},
				AllowEmpty: true,
			},
		},
		{
			StructField: "Command",
			StringListValidation: &cr.StringListValidation{
				Required:          false,
				AllowExplicitNull: true,
				AllowEmpty:        true,
			},
		},
		{
			StructField: "Args",
			StringListValidation: &cr.StringListValidation{
				Required:          false,
				AllowExplicitNull: true,
				AllowEmpty:        true,
			},
		},
		computeValidation(),
		probeValidation("LivenessProbe", true),
	}

	if kind == userconfig.RealtimeAPIKind {
		validations = append(validations, probeValidation("ReadinessProbe", true))
	} else if kind == userconfig.AsyncAPIKind || kind == userconfig.BatchAPIKind {
		validations = append(validations, probeValidation("ReadinessProbe", false))
	}

	if kind == userconfig.RealtimeAPIKind || kind == userconfig.AsyncAPIKind {
		validations = append(validations, preStopValidation())
	}

	return &cr.StructFieldValidation{
		StructField: "Containers",
		StructListValidation: &cr.StructListValidation{
			Required:         true,
			TreatNullAsEmpty: true,
			MinLength:        1,
			StructValidation: &cr.StructValidation{
				StructFieldValidations: validations,
			},
		},
	}
}

func nodegroupsValidation() *cr.StructFieldValidation {
	return &cr.StructFieldValidation{
		StructField: "NodeGroups",
		StringListValidation: &cr.StringListValidation{
			Required:          false,
			Default:           nil,
			AllowExplicitNull: true,
			AllowEmpty:        false,
			ElementStringValidation: &cr.StringValidation{
				AlphaNumericDashUnderscore: true,
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

func probeValidation(structFieldName string, hasExecProbe bool) *cr.StructFieldValidation {
	validations := []*cr.StructFieldValidation{
		httpGetHandlerValidation(),
		tcpSocketHandlerValidation(),
		{
			StructField: "InitialDelaySeconds",
			Int32Validation: &cr.Int32Validation{
				Default:              0,
				GreaterThanOrEqualTo: pointer.Int32(0),
			},
		},
		{
			StructField: "TimeoutSeconds",
			Int32Validation: &cr.Int32Validation{
				Default:              1,
				GreaterThanOrEqualTo: pointer.Int32(0),
			},
		},
		{
			StructField: "PeriodSeconds",
			Int32Validation: &cr.Int32Validation{
				Default:              10,
				GreaterThanOrEqualTo: pointer.Int32(0),
			},
		},
		{
			StructField: "SuccessThreshold",
			Int32Validation: &cr.Int32Validation{
				Default:              1,
				GreaterThanOrEqualTo: pointer.Int32(0),
			},
		},
		{
			StructField: "FailureThreshold",
			Int32Validation: &cr.Int32Validation{
				Default:              3,
				GreaterThanOrEqualTo: pointer.Int32(0),
			},
		},
	}

	if hasExecProbe {
		validations = append(validations, execHandlerValidation())
	}

	return &cr.StructFieldValidation{
		StructField: structFieldName,
		StructValidation: &cr.StructValidation{
			Required:               false,
			AllowExplicitNull:      true,
			DefaultNil:             true,
			StructFieldValidations: validations,
		},
	}
}

func preStopValidation() *cr.StructFieldValidation {
	return &cr.StructFieldValidation{
		StructField: "PreStop",
		StructValidation: &cr.StructValidation{
			Required:          false,
			AllowExplicitNull: true,
			DefaultNil:        true,
			StructFieldValidations: []*cr.StructFieldValidation{
				httpGetHandlerValidation(),
				execHandlerValidation(),
			},
		},
	}
}

func httpGetHandlerValidation() *cr.StructFieldValidation {
	return &cr.StructFieldValidation{
		StructField: "HTTPGet",
		StructValidation: &cr.StructValidation{
			Required:          false,
			AllowExplicitNull: true,
			DefaultNil:        true,
			StructFieldValidations: []*cr.StructFieldValidation{
				{
					StructField: "Path",
					StringValidation: &cr.StringValidation{
						Required:  false,
						Default:   "/",
						Validator: urls.ValidateEndpointAllowEmptyPath,
					},
				},
				{
					StructField: "Port",
					Int32Validation: &cr.Int32Validation{
						Required:          true,
						GreaterThan:       pointer.Int32(0),
						LessThanOrEqualTo: pointer.Int32(65535),
						DisallowedValues:  consts.ReservedContainerPorts,
					},
				},
			},
		},
	}
}

func tcpSocketHandlerValidation() *cr.StructFieldValidation {
	return &cr.StructFieldValidation{
		StructField: "TCPSocket",
		StructValidation: &cr.StructValidation{
			Required:          false,
			AllowExplicitNull: true,
			DefaultNil:        true,
			StructFieldValidations: []*cr.StructFieldValidation{
				{
					StructField: "Port",
					Int32Validation: &cr.Int32Validation{
						Required:          true,
						GreaterThan:       pointer.Int32(0),
						LessThanOrEqualTo: pointer.Int32(65535),
						DisallowedValues:  consts.ReservedContainerPorts,
					},
				},
			},
		},
	}
}

func execHandlerValidation() *cr.StructFieldValidation {
	return &cr.StructFieldValidation{
		StructField: "Exec",
		StructValidation: &cr.StructValidation{
			Required:          false,
			AllowExplicitNull: true,
			DefaultNil:        true,
			StructFieldValidations: []*cr.StructFieldValidation{
				{
					StructField: "Command",
					StringListValidation: &cr.StringListValidation{
						Required: true,
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
					StructField: "Shm",
					StringPtrValidation: &cr.StringPtrValidation{
						Default:           nil,
						AllowExplicitNull: true,
					},
					Parser: k8s.QuantityParser(&k8s.QuantityValidation{}),
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
						Default:              1,
						GreaterThanOrEqualTo: pointer.Int32(0),
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
						GreaterThanOrEqualTo: pointer.Int32(0),
					},
				},
				{
					StructField: "TargetInFlight",
					Float64PtrValidation: &cr.Float64PtrValidation{
						Default:     nil,
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
			return nil, errors.Append(err, fmt.Sprintf("\n\napi configuration schema can be found at https://docs.cortexlabs.com/v/%s/", consts.CortexVersionMinor))
		}

		errs = cr.Struct(&api, data, apiValidation(resourceStruct))
		if errors.HasError(errs) {
			name, _ := data[userconfig.NameKey].(string)
			kindString, _ := data[userconfig.KindKey].(string)
			kind := userconfig.KindFromString(kindString)
			err = errors.Wrap(errors.FirstError(errs...), userconfig.IdentifyAPI(configFileName, name, kind, i))
			return nil, errors.Append(err, fmt.Sprintf("\n\napi configuration schema can be found at https://docs.cortexlabs.com/v/%s/", consts.CortexVersionMinor))
		}
		api.Index = i
		api.FileName = configFileName

		interfaceMap, ok := cast.JSONMarshallable(data)
		if !ok {
			return nil, errors.ErrorUnexpected("unable to cast api spec to json") // unexpected
		}
		api.SubmittedAPISpec = interfaceMap
		apis[i] = api
	}

	return apis, nil
}

func ValidateAPI(
	api *userconfig.API,
	awsClient *aws.Client,
	k8sClient *k8s.Client,
) error {

	if api.Networking.Endpoint == nil {
		api.Networking.Endpoint = pointer.String("/" + api.Name)
	}

	if api.Pod != nil {
		if err := validatePod(api, awsClient, k8sClient); err != nil {
			return errors.Wrap(err, userconfig.PodKey)
		}
	}

	if api.Autoscaling != nil {
		if err := validateAutoscaling(api); err != nil {
			return errors.Wrap(err, userconfig.AutoscalingKey)
		}
	}

	if api.UpdateStrategy != nil {
		if err := validateUpdateStrategy(api.UpdateStrategy); err != nil {
			return errors.Wrap(err, userconfig.UpdateStrategyKey)
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

func validatePod(
	api *userconfig.API,
	awsClient *aws.Client,
	k8sClient *k8s.Client,
) error {

	if api.Pod.Port != nil && api.Kind == userconfig.TaskAPIKind {
		return ErrorFieldIsNotSupportedForKind(userconfig.PortKey, api.Kind)
	}
	if api.Pod.Port == nil && api.Kind != userconfig.TaskAPIKind {
		api.Pod.Port = pointer.Int32(consts.DefaultUserPodPortInt32)
	}

	if err := validateCompute(api); err != nil {
		return errors.Wrap(err, userconfig.ComputeKey)
	}

	if err := validateContainers(api.Pod.Containers, api.Kind, awsClient, k8sClient); err != nil {
		return errors.Wrap(err, userconfig.ContainersKey)
	}

	return nil
}

func validateContainers(
	containers []*userconfig.Container,
	kind userconfig.Kind,
	awsClient *aws.Client,
	k8sClient *k8s.Client,
) error {
	containerNames := []string{}

	for i, container := range containers {
		if slices.HasString(containerNames, container.Name) {
			return errors.Wrap(ErrorDuplicateContainerName(container.Name), s.Index(i), userconfig.ImageKey)
		}
		containerNames = append(containerNames, container.Name)

		if container.Command == nil && (kind == userconfig.BatchAPIKind || kind == userconfig.TaskAPIKind) {
			return errors.Wrap(ErrorFieldMustBeSpecifiedForKind(userconfig.CommandKey, kind), s.Index(i), userconfig.CommandKey)
		}

		// FIXME (tfriedel): re-enable this code once we know to handle it
		// if err := validateDockerImagePath(container.Image, awsClient, k8sClient); err != nil {
		// 	return errors.Wrap(err, s.Index(i), userconfig.ImageKey)
		// }

		for key := range container.Env {
			if strings.HasPrefix(key, "CORTEX_") || strings.HasPrefix(key, "KUBEXIT_") {
				return errors.Wrap(ErrorCortexPrefixedEnvVarNotAllowed("CORTEX_", "KUBEXIT_"), s.Index(i), userconfig.EnvKey, key)
			}
		}

		if kind == userconfig.TaskAPIKind && container.ReadinessProbe != nil {
			return errors.Wrap(ErrorFieldIsNotSupportedForKind(userconfig.ReadinessProbeKey, kind), s.Index(i), userconfig.ReadinessProbeKey)
		}

		if container.ReadinessProbe != nil {
			supportsExecProbe := kind == userconfig.RealtimeAPIKind
			if err := validateProbe(*container.ReadinessProbe, supportsExecProbe); err != nil {
				return errors.Wrap(err, s.Index(i), userconfig.ReadinessProbeKey)
			}
		}

		if container.LivenessProbe != nil {
			if err := validateProbe(*container.LivenessProbe, true); err != nil {
				return errors.Wrap(err, s.Index(i), userconfig.LivenessProbeKey)
			}
		}

		if container.PreStop != nil {
			if err := validatePreStop(*container.PreStop); err != nil {
				return errors.Wrap(err, s.Index(i), userconfig.PreStopKey)
			}
		}

		compute := container.Compute
		if compute.Shm != nil && compute.Mem != nil && compute.Shm.Cmp(compute.Mem.Quantity) > 0 {
			return errors.Wrap(ErrorShmCannotExceedMem(*compute.Shm, *compute.Mem), s.Index(i), userconfig.ComputeKey)
		}

	}

	return nil
}

func validateProbe(probe userconfig.Probe, supportsExecProbe bool) error {
	numSpecifiedHandlers := 0
	if probe.HTTPGet != nil {
		numSpecifiedHandlers++
	}
	if probe.TCPSocket != nil {
		numSpecifiedHandlers++
	}
	if probe.Exec != nil {
		numSpecifiedHandlers++
	}

	if numSpecifiedHandlers != 1 {
		validHandlers := []string{userconfig.HTTPGetKey, userconfig.TCPSocketKey}
		if supportsExecProbe {
			validHandlers = append(validHandlers, userconfig.ExecKey)
		}
		return ErrorSpecifyExactlyOneField(numSpecifiedHandlers, validHandlers...)
	}

	return nil
}

func validatePreStop(preStop userconfig.PreStop) error {
	numSpecifiedHandlers := 0
	if preStop.HTTPGet != nil {
		numSpecifiedHandlers++
	}
	if preStop.Exec != nil {
		numSpecifiedHandlers++
	}

	if numSpecifiedHandlers != 1 {
		return ErrorSpecifyExactlyOneField(numSpecifiedHandlers, userconfig.HTTPGetKey, userconfig.ExecKey)
	}

	return nil
}

func validateAutoscaling(api *userconfig.API) error {
	autoscaling := api.Autoscaling
	pod := api.Pod

	if api.Kind == userconfig.RealtimeAPIKind {
		if autoscaling.TargetInFlight == nil {
			autoscaling.TargetInFlight = pointer.Float64(float64(pod.MaxConcurrency))
		}
		if *autoscaling.TargetInFlight > float64(pod.MaxConcurrency)+float64(pod.MaxQueueLength) {
			return ErrorTargetInFlightLimitReached(*autoscaling.TargetInFlight, pod.MaxConcurrency, pod.MaxQueueLength)
		}
	}

	if api.Kind == userconfig.AsyncAPIKind {
		if autoscaling.TargetInFlight == nil {
			autoscaling.TargetInFlight = pointer.Float64(float64(pod.MaxConcurrency))
		}
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

	return nil
}

func validateCompute(api *userconfig.API) error {
	compute := userconfig.GetPodComputeRequest(api)

	if compute.GPU > 0 && compute.Inf > 0 {
		return ErrorComputeResourceConflict(userconfig.GPUKey, userconfig.InfKey)
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
