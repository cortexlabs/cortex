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

package operator

import (
	"fmt"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/cortexlabs/cortex/pkg/consts"
	"github.com/cortexlabs/cortex/pkg/lib/aws"
	"github.com/cortexlabs/cortex/pkg/lib/cast"
	cr "github.com/cortexlabs/cortex/pkg/lib/configreader"
	"github.com/cortexlabs/cortex/pkg/lib/errors"
	"github.com/cortexlabs/cortex/pkg/lib/k8s"
	"github.com/cortexlabs/cortex/pkg/lib/parallel"
	"github.com/cortexlabs/cortex/pkg/lib/pointer"
	s "github.com/cortexlabs/cortex/pkg/lib/strings"
	libtime "github.com/cortexlabs/cortex/pkg/lib/time"
	"github.com/cortexlabs/cortex/pkg/lib/urls"
	"github.com/cortexlabs/cortex/pkg/operator/config"
	"github.com/cortexlabs/cortex/pkg/types/userconfig"
	kresource "k8s.io/apimachinery/pkg/api/resource"
	kunstructured "k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

var _apiValidation = &cr.StructValidation{
	StructFieldValidations: []*cr.StructFieldValidation{
		{
			StructField: "Name",
			StringValidation: &cr.StringValidation{
				Required:  true,
				DNS1035:   true,
				MaxLength: 42, // k8s adds 21 characters to the pod name, and 63 is the max before it starts to truncate
			},
		},
		{
			StructField: "Endpoint",
			StringPtrValidation: &cr.StringPtrValidation{
				Validator: urls.ValidateEndpoint,
				MaxLength: 1000, // no particular reason other than it works
			},
		},
		_predictorValidation,
		_trackerValidation,
		_computeValidation,
		_autoscalingValidation,
		_updateStrategyValidation,
	},
}

var _predictorValidation = &cr.StructFieldValidation{
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
				StructField: "Model",
				StringPtrValidation: &cr.StringPtrValidation{
					Validator: cr.S3PathValidator(),
				},
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

var _trackerValidation = &cr.StructFieldValidation{
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
					AllowedValues: userconfig.ModelTypeStrings(),
				},
				Parser: func(str string) (interface{}, error) {
					return userconfig.ModelTypeFromString(str), nil
				},
			},
		},
	},
}

var _computeValidation = &cr.StructFieldValidation{
	StructField: "Compute",
	StructValidation: &cr.StructValidation{
		StructFieldValidations: []*cr.StructFieldValidation{
			{
				StructField: "CPU",
				StringValidation: &cr.StringValidation{
					Default:     "200m",
					CastNumeric: true,
				},
				Parser: k8s.QuantityParser(&k8s.QuantityValidation{
					GreaterThanOrEqualTo: k8s.QuantityPtr(kresource.MustParse("20m")),
				}),
			},
			{
				StructField: "Mem",
				StringPtrValidation: &cr.StringPtrValidation{
					Default: nil,
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
		},
	},
}

var _autoscalingValidation = &cr.StructFieldValidation{
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
				StructField: "WorkersPerReplica",
				Int32Validation: &cr.Int32Validation{
					Default:              4,
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
				StructField: "TargetQueueLength",
				Float64Validation: &cr.Float64Validation{
					Default:              0,
					GreaterThanOrEqualTo: pointer.Float64(0),
				},
			},
			{
				StructField: "Window",
				StringValidation: &cr.StringValidation{
					Default: "60s",
				},
				Parser: cr.DurationParser(&cr.DurationValidation{
					GreaterThanOrEqualTo: &consts.AutoscalingTickInterval,
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
					Default: "0s",
				},
				Parser: cr.DurationParser(&cr.DurationValidation{
					GreaterThanOrEqualTo: pointer.Duration(libtime.MustParseDuration("0s")),
				}),
			},
			{
				StructField: "MaxDownscaleFactor",
				Float64Validation: &cr.Float64Validation{
					Default:              0.5,
					GreaterThanOrEqualTo: pointer.Float64(0),
					LessThan:             pointer.Float64(1),
				},
			},
			{
				StructField: "MaxUpscaleFactor",
				Float64Validation: &cr.Float64Validation{
					Default:     10,
					GreaterThan: pointer.Float64(1),
				},
			},
			{
				StructField: "DownscaleTolerance",
				Float64Validation: &cr.Float64Validation{
					Default:              0.1,
					GreaterThanOrEqualTo: pointer.Float64(0),
					LessThan:             pointer.Float64(1),
				},
			},
			{
				StructField: "UpscaleTolerance",
				Float64Validation: &cr.Float64Validation{
					Default:              0.1,
					GreaterThanOrEqualTo: pointer.Float64(0),
				},
			},
		},
	},
}

var _updateStrategyValidation = &cr.StructFieldValidation{
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

func ExtractAPIConfigs(configBytes []byte, projectFileMap map[string][]byte, filePath string) ([]userconfig.API, error) {
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
		errs := cr.Struct(&api, data, _apiValidation)
		if errors.HasError(errs) {
			name, _ := data[userconfig.NameKey].(string)
			return nil, errors.Wrap(errors.FirstError(errs...), userconfig.IdentifyAPI(filePath, name, i))
		}
		api.Index = i
		api.FilePath = filePath
		apis[i] = api
	}

	if err := validateAPIs(apis, projectFileMap); err != nil {
		return nil, err
	}

	return apis, nil
}

func validateAPIs(apis []userconfig.API, projectFileMap map[string][]byte) error {
	if len(apis) == 0 {
		return ErrorNoAPIs()
	}

	virtualServices, maxMem, err := getValidationK8sResources()
	if err != nil {
		return err
	}

	for i := range apis {
		if err := validateAPI(&apis[i], projectFileMap, virtualServices, maxMem); err != nil {
			return err
		}
	}

	dups := findDuplicateNames(apis)
	if len(dups) > 0 {
		return ErrorDuplicateName(dups)
	}

	dups = findDuplicateEndpoints(apis)
	if len(dups) > 0 {
		return ErrorDuplicateEndpointInOneDeploy(dups)
	}

	return nil
}

func validateAPI(
	api *userconfig.API,
	projectFileMap map[string][]byte,
	virtualServices []kunstructured.Unstructured,
	maxMem *kresource.Quantity,
) error {

	if api.Endpoint == nil {
		api.Endpoint = pointer.String("/" + api.Name)
	}

	if err := validatePredictor(api.Predictor, projectFileMap); err != nil {
		return errors.Wrap(err, api.Identify(), userconfig.PredictorKey)
	}

	if err := validateCompute(api.Compute, maxMem); err != nil {
		return errors.Wrap(err, api.Identify(), userconfig.ComputeKey)
	}

	if err := validateAutoscaling(api.Autoscaling); err != nil {
		return errors.Wrap(err, api.Identify(), userconfig.AutoscalingKey)
	}

	if err := validateUpdateStrategy(api.UpdateStrategy); err != nil {
		return errors.Wrap(err, api.Identify(), userconfig.UpdateStrategyKey)
	}

	if err := validateEndpointCollisions(api, virtualServices); err != nil {
		return err
	}

	return nil
}

func validatePredictor(predictor *userconfig.Predictor, projectFileMap map[string][]byte) error {
	switch predictor.Type {
	case userconfig.PythonPredictorType:
		if err := validatePythonPredictor(predictor); err != nil {
			return err
		}
	case userconfig.TensorFlowPredictorType:
		if err := validateTensorFlowPredictor(predictor); err != nil {
			return err
		}
	case userconfig.ONNXPredictorType:
		if err := validateONNXPredictor(predictor); err != nil {
			return err
		}
	}

	for key := range predictor.Env {
		if strings.HasPrefix(key, "CORTEX_") {
			return errors.Wrap(ErrorCortexPrefixedEnvVarNotAllowed(), userconfig.EnvKey, key)
		}
	}

	if _, ok := projectFileMap[predictor.Path]; !ok {
		return errors.Wrap(ErrorImplDoesNotExist(predictor.Path), userconfig.PathKey)
	}

	if predictor.PythonPath != nil {
		if err := validatePythonPath(*predictor.PythonPath, projectFileMap); err != nil {
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

	return nil
}

func validateTensorFlowPredictor(predictor *userconfig.Predictor) error {
	if predictor.Model == nil {
		return ErrorFieldMustBeDefinedForPredictorType(userconfig.ModelKey, userconfig.TensorFlowPredictorType)
	}

	model := *predictor.Model

	awsClient, err := aws.NewFromEnvS3Path(model)
	if err != nil {
		return err
	}

	if strings.HasSuffix(model, ".zip") {
		if ok, err := awsClient.IsS3PathFile(model); err != nil || !ok {
			return errors.Wrap(ErrorS3FileNotFound(model), userconfig.ModelKey)
		}
	} else {
		path, err := getTFServingExportFromS3Path(model, awsClient)
		if err != nil {
			return err
		} else if path == "" {
			return errors.Wrap(ErrorInvalidTensorFlowDir(model), userconfig.ModelKey)
		}
		predictor.Model = pointer.String(path)
	}

	return nil
}

func validateONNXPredictor(predictor *userconfig.Predictor) error {
	if predictor.Model == nil {
		return ErrorFieldMustBeDefinedForPredictorType(userconfig.ModelKey, userconfig.ONNXPredictorType)
	}

	model := *predictor.Model

	awsClient, err := aws.NewFromEnvS3Path(model)
	if err != nil {
		return err
	}

	if ok, err := awsClient.IsS3PathFile(model); err != nil || !ok {
		return errors.Wrap(ErrorS3FileNotFound(model), userconfig.ModelKey)
	}

	if predictor.SignatureKey != nil {
		return ErrorFieldNotSupportedByPredictorType(userconfig.SignatureKeyKey, userconfig.ONNXPredictorType)
	}

	return nil
}

func getTFServingExportFromS3Path(path string, awsClient *aws.Client) (string, error) {
	if isValidTensorFlowS3Directory(path, awsClient) {
		return path, nil
	}

	bucket, _, err := aws.SplitS3Path(path)
	if err != nil {
		return "", err
	}

	objects, err := config.AWS.ListPathDir(path, 1000)
	if err != nil {
		return "", err
	} else if len(objects) == 0 {
		return "", ErrorS3DirNotFoundOrEmpty(path)
	}

	highestVersion := int64(0)
	var highestPath string
	for _, key := range objects {
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
		if version >= highestVersion && isValidTensorFlowS3Directory(possiblePath, awsClient) {
			highestVersion = version
			highestPath = possiblePath
		}
	}

	return highestPath, nil
}

// IsValidTensorFlowS3Directory checks that the path contains a valid S3 directory for TensorFlow models
// Must contain the following structure:
// - 1523423423/ (version prefix, usually a timestamp)
// 		- saved_model.pb
//		- variables/
//			- variables.index
//			- variables.data-00000-of-00001 (there are a variable number of these files)
func isValidTensorFlowS3Directory(path string, awsClient *aws.Client) bool {
	if valid, err := awsClient.IsS3PathFile(
		aws.JoinS3Path(path, "saved_model.pb"),
		aws.JoinS3Path(path, "variables/variables.index"),
	); err != nil || !valid {
		return false
	}

	if valid, err := awsClient.IsS3PathPrefix(
		aws.JoinS3Path(path, "variables/variables.data-00000-of"),
	); err != nil || !valid {
		return false
	}
	return true
}

func validatePythonPath(pythonPath string, projectFileMap map[string][]byte) error {
	validPythonPath := false
	for fileKey := range projectFileMap {
		if strings.HasPrefix(fileKey, pythonPath) {
			validPythonPath = true
			break
		}
	}
	if !validPythonPath {
		return ErrorImplDoesNotExist(pythonPath)
	}
	return nil
}

func validateCompute(compute *userconfig.Compute, maxMem *kresource.Quantity) error {
	maxMem.Sub(_cortexMemReserve)

	maxCPU := config.Cluster.InstanceMetadata.CPU
	maxCPU.Sub(_cortexCPUReserve)

	maxGPU := config.Cluster.InstanceMetadata.GPU
	if maxGPU > 0 {
		// Reserve resources for nvidia device plugin daemonset
		maxCPU.Sub(_nvidiaCPUReserve)
		maxMem.Sub(_nvidiaMemReserve)
	}

	if maxCPU.Cmp(compute.CPU.Quantity) < 0 {
		return ErrorNoAvailableNodeComputeLimit("CPU", compute.CPU.String(), maxCPU.String())
	}
	if compute.Mem != nil {
		if maxMem.Cmp(compute.Mem.Quantity) < 0 {
			return ErrorNoAvailableNodeComputeLimit("Memory", compute.Mem.String(), maxMem.String())
		}
	}
	if compute.GPU > maxGPU {
		return ErrorNoAvailableNodeComputeLimit("GPU", fmt.Sprintf("%d", compute.GPU), fmt.Sprintf("%d", maxGPU))
	}
	return nil
}

func validateAutoscaling(autoscaling *userconfig.Autoscaling) error {
	if autoscaling.MinReplicas > autoscaling.MaxReplicas {
		return ErrorMinReplicasGreaterThanMax(autoscaling.MinReplicas, autoscaling.MaxReplicas)
	}

	if autoscaling.InitReplicas > autoscaling.MaxReplicas {
		return ErrorInitReplicasGreaterThanMax(autoscaling.InitReplicas, autoscaling.MaxReplicas)
	}

	if autoscaling.InitReplicas < autoscaling.MinReplicas {
		return ErrorInitReplicasLessThanMin(autoscaling.InitReplicas, autoscaling.MinReplicas)
	}

	if autoscaling.Window.Nanoseconds()%consts.AutoscalingTickInterval.Nanoseconds() != 0 {
		return errors.Wrap(ErrorMustBeMultiple(autoscaling.Window.String(), consts.AutoscalingTickInterval.String()), userconfig.WindowKey)
	}

	return nil
}

func validateUpdateStrategy(updateStrategy *userconfig.UpdateStrategy) error {
	if (updateStrategy.MaxSurge == "0" || updateStrategy.MaxSurge == "0%") && (updateStrategy.MaxUnavailable == "0" || updateStrategy.MaxUnavailable == "0%") {
		return ErrorSurgeAndUnavailableBothZero()
	}

	return nil
}

func validateEndpointCollisions(api *userconfig.API, virtualServices []kunstructured.Unstructured) error {
	for _, virtualService := range virtualServices {
		gateways, err := k8s.ExtractVirtualServiceGateways(&virtualService)
		if err != nil {
			return err
		}
		if !gateways.Has("apis-gateway") {
			continue
		}

		endpoints, err := k8s.ExtractVirtualServiceEndpoints(&virtualService)
		if err != nil {
			return err
		}

		for endpoint := range endpoints {
			if s.EnsureSuffix(endpoint, "/") == s.EnsureSuffix(*api.Endpoint, "/") && virtualService.GetLabels()["apiName"] != api.Name {
				return errors.Wrap(ErrorDuplicateEndpoint(virtualService.GetLabels()["apiName"]), api.Identify(), userconfig.EndpointKey, endpoint)
			}
		}
	}

	return nil
}

func findDuplicateNames(apis []userconfig.API) []userconfig.API {
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

func findDuplicateEndpoints(apis []userconfig.API) []userconfig.API {
	endpoints := make(map[string][]userconfig.API)

	for _, api := range apis {
		endpoints[*api.Endpoint] = append(endpoints[*api.Endpoint], api)
	}

	for endpoint := range endpoints {
		if len(endpoints[endpoint]) > 1 {
			return endpoints[endpoint]
		}
	}

	return nil
}

func getValidationK8sResources() ([]kunstructured.Unstructured, *kresource.Quantity, error) {
	var virtualServices []kunstructured.Unstructured
	var maxMem *kresource.Quantity

	err := parallel.RunFirstErr(
		func() error {
			var err error
			virtualServices, err = config.K8s.ListVirtualServices(nil)
			return err
		},
		func() error {
			var err error
			maxMem, err = updateMemoryCapacityConfigMap()
			return err
		},
	)

	return virtualServices, maxMem, err
}
