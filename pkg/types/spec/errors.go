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

	"github.com/cortexlabs/cortex/pkg/consts"
	"github.com/cortexlabs/cortex/pkg/lib/errors"
	libmath "github.com/cortexlabs/cortex/pkg/lib/math"
	"github.com/cortexlabs/cortex/pkg/lib/sets/strset"
	s "github.com/cortexlabs/cortex/pkg/lib/strings"
	"github.com/cortexlabs/cortex/pkg/types/userconfig"
)

const (
	ErrMalformedConfig                   = "spec.malformed_config"
	ErrNoAPIs                            = "spec.no_apis"
	ErrDuplicateName                     = "spec.duplicate_name"
	ErrDuplicateEndpointInOneDeploy      = "spec.duplicate_endpoint_in_one_deploy"
	ErrDuplicateEndpoint                 = "spec.duplicate_endpoint"
	ErrConflictingFields                 = "spec.conflicting_fields"
	ErrSpecifyOneOrTheOther              = "spec.specify_one_or_the_other"
	ErrSpecifyAllOrNone                  = "spec.specify_all_or_none"
	ErrOneOfPrerequisitesNotDefined      = "spec.one_of_prerequisites_not_defined"
	ErrConfigGreaterThanOtherConfig      = "spec.config_greater_than_other_config"
	ErrMinReplicasGreaterThanMax         = "spec.min_replicas_greater_than_max"
	ErrInitReplicasGreaterThanMax        = "spec.init_replicas_greater_than_max"
	ErrInitReplicasLessThanMin           = "spec.init_replicas_less_than_min"
	ErrInvalidSurgeOrUnavailable         = "spec.invalid_surge_or_unavailable"
	ErrSurgeAndUnavailableBothZero       = "spec.surge_and_unavailable_both_zero"
	ErrCacheSizeGreaterThanNumModels     = "spec.cache_size_greater_than_num_models"
	ErrDiskCacheSizeGreaterThanNumModels = "spec.disk_cache_size_greater_than_num_models"

	ErrInvalidPath               = "spec.invalid_path"
	ErrInvalidDirPath            = "spec.invalid_dir_path"
	ErrFileNotFound              = "spec.file_not_found"
	ErrDirIsEmpty                = "spec.dir_is_empty"
	ErrMustBeRelativeProjectPath = "spec.must_be_relative_project_path"
	ErrPythonPathNotFound        = "spec.python_path_not_found"

	ErrS3FileNotFound = "spec.s3_file_not_found"
	ErrS3DirNotFound  = "spec.s3_dir_not_found"

	ErrInvalidONNXModelPath            = "spec.invalid_onnx_model_path"
	ErrInvalidONNXModelPathSuffix      = "spec.invalid_onnx_model_path_suffix"
	ErrNoVersionsFoundForONNXModelPath = "spec.no_versions_found_for_onnx_model_path"
	ErrONNXModelVersionPathMustBeDir   = "spec.onnx_model_version_path_must_be_dir"

	ErrInvalidPythonModelPath            = "spec.invalid_python_model_path"
	ErrNoVersionsFoundForPythonModelPath = "spec.no_versions_found_for_python_model_path"
	ErrPythonModelVersionPathMustBeDir   = "spec.python_model_version_path_must_be_dir"

	ErrInvalidTensorFlowModelPath            = "spec.invalid_tensorflow_model_path"
	ErrNoVersionsFoundForTensorFlowModelPath = "spec.no_versions_found_for_tensorflow_model_path"
	ErrTensorFlowModelVersionPathMustBeDir   = "spec.tensorflow_model_version_path_must_be_dir"

	ErrMissingModel        = "spec.missing_model"
	ErrDuplicateModelNames = "spec.duplicate_model_names"
	ErrIllegalModelName    = "spec.illegal_model_name"

	ErrFieldMustBeDefinedForPredictorType   = "spec.field_must_be_defined_for_predictor_type"
	ErrFieldNotSupportedByPredictorType     = "spec.field_not_supported_by_predictor_type"
	ErrNoAvailableNodeComputeLimit          = "spec.no_available_node_compute_limit"
	ErrCortexPrefixedEnvVarNotAllowed       = "spec.cortex_prefixed_env_var_not_allowed"
	ErrLocalPathNotSupportedByAWSProvider   = "spec.local_path_not_supported_by_aws_provider"
	ErrUnsupportedLocalComputeResource      = "spec.unsupported_local_compute_resource"
	ErrRegistryInDifferentRegion            = "spec.registry_in_different_region"
	ErrRegistryAccountIDMismatch            = "spec.registry_account_id_mismatch"
	ErrCannotAccessECRWithAnonymousAWSCreds = "spec.cannot_access_ecr_with_anonymous_aws_creds"
	ErrComputeResourceConflict              = "spec.compute_resource_conflict"
	ErrInvalidNumberOfInfProcesses          = "spec.invalid_number_of_inf_processes"
	ErrInvalidNumberOfInfs                  = "spec.invalid_number_of_infs"
)

func ErrorMalformedConfig() error {
	return errors.WithStack(&errors.Error{
		Kind:    ErrMalformedConfig,
		Message: fmt.Sprintf("cortex YAML configuration files must contain a list of maps (see https://docs.cortex.dev/v/%s/deployments/api-configuration for documentation)", consts.CortexVersionMinor),
	})
}

func ErrorNoAPIs() error {
	return errors.WithStack(&errors.Error{
		Kind:    ErrNoAPIs,
		Message: fmt.Sprintf("at least one API must be configured (see https://docs.cortex.dev/v/%s/deployments/api-configuration for documentation)", consts.CortexVersionMinor),
	})
}

func ErrorDuplicateName(apis []userconfig.API) error {
	filePaths := strset.New()
	for _, api := range apis {
		filePaths.Add(api.FileName)
	}

	return errors.WithStack(&errors.Error{
		Kind:    ErrDuplicateName,
		Message: fmt.Sprintf("name %s must be unique across apis (defined in %s)", s.UserStr(apis[0].Name), s.StrsAnd(filePaths.Slice())),
	})
}

func ErrorDuplicateEndpointInOneDeploy(apis []userconfig.API) error {
	names := make([]string, len(apis))
	for i, api := range apis {
		names[i] = api.Name
	}

	return errors.WithStack(&errors.Error{
		Kind:    ErrDuplicateEndpointInOneDeploy,
		Message: fmt.Sprintf("endpoint %s must be unique across apis (defined in %s)", s.UserStr(*apis[0].Networking.Endpoint), s.StrsAnd(names)),
	})
}

func ErrorDuplicateEndpoint(apiName string) error {
	return errors.WithStack(&errors.Error{
		Kind:    ErrDuplicateEndpoint,
		Message: fmt.Sprintf("endpoint is already being used by %s", apiName),
	})
}

func ErrorConflictingFields(fieldKeyA, fieldKeyB string) error {
	return errors.WithStack(&errors.Error{
		Kind:    ErrConflictingFields,
		Message: fmt.Sprintf("please specify either the %s or %s field (both cannot be specified at the same time)", fieldKeyA, fieldKeyB),
	})
}

func ErrorSpecifyOneOrTheOther(fieldKeyA, fieldKeyB string) error {
	return errors.WithStack(&errors.Error{
		Kind:    ErrSpecifyOneOrTheOther,
		Message: fmt.Sprintf("please specify either the %s field or %s field (cannot be both empty at the same time", fieldKeyA, fieldKeyB),
	})
}

func ErrorSpecifyAllOrNone(val string, vals ...string) error {
	allVals := append([]string{val}, vals...)
	message := fmt.Sprintf("please specify all or none of %s", s.UserStrsAnd(allVals))
	if len(allVals) == 2 {
		message = fmt.Sprintf("please specify both %s and %s or neither of them", s.UserStr(allVals[0]), s.UserStr(allVals[1]))
	}

	return errors.WithStack(&errors.Error{
		Kind:    ErrSpecifyAllOrNone,
		Message: message,
	})
}

func ErrorOneOfPrerequisitesNotDefined(argName string, prerequisite string, prerequisites ...string) error {
	allPrerequisites := append([]string{prerequisite}, prerequisites...)
	message := fmt.Sprintf("%s specified without specifying %s", s.UserStr(argName), s.UserStrsOr(allPrerequisites))

	return errors.WithStack(&errors.Error{
		Kind:    ErrOneOfPrerequisitesNotDefined,
		Message: message,
	})
}

func ErrorConfigGreaterThanOtherConfig(tooBigKey string, tooBigVal interface{}, tooSmallKey string, tooSmallVal interface{}) error {
	return errors.WithStack(&errors.Error{
		Kind:    ErrConfigGreaterThanOtherConfig,
		Message: fmt.Sprintf("%s (%s) cannot be greater than %s (%s)", tooBigKey, s.UserStr(tooBigVal), tooSmallKey, s.UserStr(tooSmallVal)),
	})
}

func ErrorMinReplicasGreaterThanMax(min int32, max int32) error {
	return errors.WithStack(&errors.Error{
		Kind:    ErrMinReplicasGreaterThanMax,
		Message: fmt.Sprintf("%s cannot be greater than %s (%d > %d)", userconfig.MinReplicasKey, userconfig.MaxReplicasKey, min, max),
	})
}

func ErrorInitReplicasGreaterThanMax(init int32, max int32) error {
	return errors.WithStack(&errors.Error{
		Kind:    ErrInitReplicasGreaterThanMax,
		Message: fmt.Sprintf("%s cannot be greater than %s (%d > %d)", userconfig.InitReplicasKey, userconfig.MaxReplicasKey, init, max),
	})
}

func ErrorInitReplicasLessThanMin(init int32, min int32) error {
	return errors.WithStack(&errors.Error{
		Kind:    ErrInitReplicasLessThanMin,
		Message: fmt.Sprintf("%s cannot be less than %s (%d < %d)", userconfig.InitReplicasKey, userconfig.MinReplicasKey, init, min),
	})
}

func ErrorInvalidSurgeOrUnavailable(val string) error {
	return errors.WithStack(&errors.Error{
		Kind:    ErrInvalidSurgeOrUnavailable,
		Message: fmt.Sprintf("%s is not a valid value - must be an integer percentage (e.g. 25%%, to denote a percentage of desired replicas) or a positive integer (e.g. 5, to denote a number of replicas)", s.UserStr(val)),
	})
}

func ErrorSurgeAndUnavailableBothZero() error {
	return errors.WithStack(&errors.Error{
		Kind:    ErrSurgeAndUnavailableBothZero,
		Message: fmt.Sprintf("%s and %s cannot both be zero", userconfig.MaxSurgeKey, userconfig.MaxUnavailableKey),
	})
}

func ErrorCacheSizeGreaterThanNumModels(cacheSize, numModels int) error {
	return errors.WithStack(&errors.Error{
		Kind:    ErrCacheSizeGreaterThanNumModels,
		Message: fmt.Sprintf("%s cannot be greater than the number of provided models (%d > %d)", userconfig.ModelsCacheSizeKey, cacheSize, numModels),
	})
}

func ErrorDiskCacheSizeGreaterThanNumModels(cacheSize, numModels int) error {
	return errors.WithStack(&errors.Error{
		Kind:    ErrDiskCacheSizeGreaterThanNumModels,
		Message: fmt.Sprintf("%s cannot be greater than the number of provided models (%d > %d)", userconfig.ModelsDiskCacheSizeKey, cacheSize, numModels),
	})
}

func ErrorInvalidPath(path string) error {
	return errors.WithStack(&errors.Error{
		Kind:    ErrInvalidPath,
		Message: fmt.Sprintf("%s: is not a valid path", path),
	})
}

func ErrorInvalidDirPath(path string) error {
	return errors.WithStack(&errors.Error{
		Kind:    ErrInvalidDirPath,
		Message: fmt.Sprintf("%s: invalid directory path", path),
	})
}

func ErrorFileNotFound(path string) error {
	return errors.WithStack(&errors.Error{
		Kind:    ErrFileNotFound,
		Message: fmt.Sprintf("%s: not found or insufficient permissions", path),
	})
}

func ErrorDirIsEmpty(path string) error {
	return errors.WithStack(&errors.Error{
		Kind:    ErrDirIsEmpty,
		Message: fmt.Sprintf("%s: directory is empty", path),
	})
}

func ErrorMustBeRelativeProjectPath(path string) error {
	return errors.WithStack(&errors.Error{
		Kind:    ErrMustBeRelativeProjectPath,
		Message: fmt.Sprintf("%s: must be a relative path (relative to the directory containing your API configuration file)", path),
	})
}

func ErrorPythonPathNotFound(pythonPath string) error {
	return errors.WithStack(&errors.Error{
		Kind:    ErrPythonPathNotFound,
		Message: fmt.Sprintf("%s: path does not exist, or has been excluded from your project directory", pythonPath),
	})
}

func ErrorS3FileNotFound(path string) error {
	return errors.WithStack(&errors.Error{
		Kind:    ErrS3FileNotFound,
		Message: fmt.Sprintf("%s: file not found or insufficient permissions", path),
	})
}

func ErrorS3DirNotFound(path string) error {
	return errors.WithStack(&errors.Error{
		Kind:    ErrS3DirNotFound,
		Message: fmt.Sprintf("%s: dir not found or insufficient permissions", path),
	})
}

var _onnxExpectedStructMessage = `
  %s
  ├── 1523423423/ (Version prefix)
  |   └── <model-name>.onnx // ONNX-exported file
  └── 2434389194/ (Version prefix)
      └── <model-name>.onnx // ONNX-exported file`

func ErrorInvalidONNXModelPath(path string) error {
	message := fmt.Sprintf("%s: %s model path must be an ONNX-exported file ending in `.onnx`, or can be a directory with the following structure.\n", path, userconfig.ONNXPredictorType.CasedString())
	message += fmt.Sprintf(_onnxExpectedStructMessage, path)
	return errors.WithStack(&errors.Error{
		Kind:    ErrInvalidONNXModelPath,
		Message: message,
	})
}

func ErrorInvalidONNXModelPathSuffix(path string) error {
	message := fmt.Sprintf("%s: when %s model path ends with `.onnx`, it must be a file representing the ONNX-exported model", path, userconfig.ONNXPredictorType.CasedString())
	return errors.WithStack(&errors.Error{
		Kind:    ErrInvalidONNXModelPath,
		Message: message,
	})
}

func ErrorNoVersionsFoundForONNXModelPath(path string) error {
	message := fmt.Sprintf("%s: no versions found for %s model. Each model must be structured the following way.\n", path, userconfig.ONNXPredictorType.CasedString())
	message += fmt.Sprintf(_onnxExpectedStructMessage, path)
	return errors.WithStack(&errors.Error{
		Kind:    ErrNoVersionsFoundForONNXModelPath,
		Message: message,
	})
}

func ErrorONNXModelVersionPathMustBeDir(path, versionedPath string) error {
	message := fmt.Sprintf("%s: when the %s model is a directory, each version of the model must be a directory too. The model directory must be structured the following way.\n", versionedPath, userconfig.ONNXPredictorType)
	message += fmt.Sprintf(_onnxExpectedStructMessage, path)
	return errors.WithStack(&errors.Error{
		Kind:    ErrONNXModelVersionPathMustBeDir,
		Message: message,
	})
}

var _pythonExpectedStructMessage = `For models provided for the %s predictor type, the path must be a directory with the following structure:
  %s
  ├── 1523423423/ (Version prefix)
  |   └── * // Model-specific files (i.e. model.h5, model.pkl, labels.json, etc)
  └── 2434389194/ (Version prefix)
      └── * // Model-specific files (i.e. model.h5, model.pkl, labels.json, etc)`

func ErrorInvalidPythonModelPath(path string) error {
	message := fmt.Sprintf("invalid %s model path.\n", userconfig.PythonPredictorType.CasedString())
	message += fmt.Sprintf(_pythonExpectedStructMessage, userconfig.PythonPredictorType, path)
	return errors.WithStack(&errors.Error{
		Kind:    ErrInvalidPythonModelPath,
		Message: message,
	})
}

func ErrorNoVersionsFoundForPythonModelPath(path string) error {
	message := fmt.Sprintf("%s: %s model path must have at least one version.\n", path, userconfig.PythonPredictorType.CasedString())
	message += fmt.Sprintf(_pythonExpectedStructMessage, userconfig.PythonPredictorType, path)
	return errors.WithStack(&errors.Error{
		Kind:    ErrNoVersionsFoundForPythonModelPath,
		Message: message,
	})
}

func ErrorPythonModelVersionPathMustBeDir(path, versionedPath string) error {
	message := fmt.Sprintf("%s: %s model version path must be a directory.\n", versionedPath, userconfig.PythonPredictorType.CasedString())
	message += fmt.Sprintf(_pythonExpectedStructMessage, userconfig.PythonPredictorType, path)
	return errors.WithStack(&errors.Error{
		Kind:    ErrPythonModelVersionPathMustBeDir,
		Message: message,
	})
}

var _tfExpectedStructMessage = `
  %s
  ├── 1523423423/ (Version prefix, usually a timestamp)
  |   ├── saved_model.pb
  |   └── variables/
  |       ├── variables.index
  |       ├── variables.data-00000-of-00003
  |       ├── variables.data-00001-of-00003
  |       └── variables.data-00002-of-...
  └── 2434389194/ (Version prefix, usually a timestamp)
      ├── saved_model.pb
      └── variables/
          ├── variables.index
          ├── variables.data-00000-of-00003
          ├── variables.data-00001-of-00003
          └── variables.data-00002-of-...`

var _neuronTfExpectedStructMessage = `
  %s
  ├── 1523423423/ (Version prefix, usually a timestamp)
  |   └── saved_model.pb
  └── 2434389194/ (Version prefix, usually a timestamp)
      └── saved_model.pb`

func ErrorInvalidTensorFlowModelPath(path string, neuronExport bool) error {
	neuronKey := ""
	if neuronExport {
		neuronKey = "Neuron "
	}
	message := fmt.Sprintf("%s: each %s%s SavedModel model must be structured the following way.\n", path, neuronKey, userconfig.TensorFlowPredictorType.CasedString())
	if neuronExport {
		message += fmt.Sprintf(_neuronTfExpectedStructMessage, path)
	} else {
		message += fmt.Sprintf(_tfExpectedStructMessage, path)
	}
	return errors.WithStack(&errors.Error{
		Kind:    ErrInvalidTensorFlowModelPath,
		Message: message,
	})
}

func ErrorNoVersionsFoundForTensorFlowModelPath(path string, neuronExport bool) error {
	neuronKey := ""
	if neuronExport {
		neuronKey = "Neuron "
	}
	message := fmt.Sprintf("%s: no versions found for %s%s SavedModel. The SavedModel model must be structured the following way.\n", path, neuronKey, userconfig.TensorFlowPredictorType.CasedString())
	if neuronExport {
		message += fmt.Sprintf(_neuronTfExpectedStructMessage, path)
	} else {
		message += fmt.Sprintf(_tfExpectedStructMessage, path)
	}
	return errors.WithStack(&errors.Error{
		Kind:    ErrInvalidTensorFlowModelPath,
		Message: message,
	})
}

func ErrorTensorFlowModelVersionPathMustBeDir(path, versionedPath string, neuronExport bool) error {
	neuronKey := ""
	if neuronExport {
		neuronKey = "Neuron "
	}
	message := fmt.Sprintf("%s: each %s%s SavedModel version must be a directory. The SavedModel model must be structured the following way.\n", versionedPath, neuronKey, userconfig.TensorFlowPredictorType.CasedString())
	if neuronExport {
		message += fmt.Sprintf(_neuronTfExpectedStructMessage, path)
	} else {
		message += fmt.Sprintf(_tfExpectedStructMessage, path)
	}
	return errors.WithStack(&errors.Error{
		Kind:    ErrInvalidTensorFlowModelPath,
		Message: message,
	})
}

func ErrorMissingModel(predictorType userconfig.PredictorType) error {
	return errors.WithStack(&errors.Error{
		Kind:    ErrMissingModel,
		Message: fmt.Sprintf("at least one model must be specified for %s predictor type; use fields %s:%s or %s:%s to add model(s)", predictorType, userconfig.PredictorKey, userconfig.ModelPathKey, userconfig.PredictorKey, userconfig.ModelsKey),
	})
}

func ErrorDuplicateModelNames(duplicateModel string) error {
	return errors.WithStack(&errors.Error{
		Kind:    ErrDuplicateModelNames,
		Message: fmt.Sprintf("cannot have multiple models with the same name (%s)", duplicateModel),
	})
}

func ErrorIllegalModelName(illegalModel string) error {
	return errors.WithStack(&errors.Error{
		Kind:    ErrIllegalModelName,
		Message: fmt.Sprintf("%s: use a different model name", illegalModel),
	})
}

func ErrorFieldMustBeDefinedForPredictorType(fieldKey string, predictorType userconfig.PredictorType) error {
	return errors.WithStack(&errors.Error{
		Kind:    ErrFieldMustBeDefinedForPredictorType,
		Message: fmt.Sprintf("%s field must be defined for the %s predictor type", fieldKey, predictorType.String()),
	})
}

func ErrorFieldNotSupportedByPredictorType(fieldKey string, predictorType userconfig.PredictorType) error {
	return errors.WithStack(&errors.Error{
		Kind:    ErrFieldNotSupportedByPredictorType,
		Message: fmt.Sprintf("%s is not a supported field for the %s predictor type", fieldKey, predictorType.String()),
	})
}

func ErrorCortexPrefixedEnvVarNotAllowed() error {
	return errors.WithStack(&errors.Error{
		Kind:    ErrCortexPrefixedEnvVarNotAllowed,
		Message: fmt.Sprintf("environment variables starting with CORTEX_ are reserved"),
	})
}

func ErrorLocalModelPathNotSupportedByAWSProvider() error {
	return errors.WithStack(&errors.Error{
		Kind:    ErrLocalPathNotSupportedByAWSProvider,
		Message: fmt.Sprintf("local model paths are not supported for aws provider, please specify an S3 path"),
	})
}

func ErrorUnsupportedLocalComputeResource(resourceType string) error {
	return errors.WithStack(&errors.Error{
		Kind:    ErrUnsupportedLocalComputeResource,
		Message: fmt.Sprintf("%s compute resources cannot be used locally", resourceType),
	})
}

func ErrorRegistryInDifferentRegion(registryRegion string, awsClientRegion string) error {
	return errors.WithStack(&errors.Error{
		Kind:    ErrRegistryInDifferentRegion,
		Message: fmt.Sprintf("registry region (%s) does not match cortex's region (%s); images can only be pulled from repositories in the same region as cortex", registryRegion, awsClientRegion),
	})
}

func ErrorRegistryAccountIDMismatch(regID, opID string) error {
	return errors.WithStack(&errors.Error{
		Kind:    ErrRegistryAccountIDMismatch,
		Message: fmt.Sprintf("registry account ID (%s) doesn't match your AWS account ID (%s), and using an ECR registry in a different AWS account is not supported", regID, opID),
	})
}

func ErrorCannotAccessECRWithAnonymousAWSCreds() error {
	return errors.WithStack(&errors.Error{
		Kind:    ErrCannotAccessECRWithAnonymousAWSCreds,
		Message: fmt.Sprintf("cannot access ECR with anonymous aws credentials; run `cortex env configure local` to specify AWS credentials with access to ECR"),
	})
}

func ErrorComputeResourceConflict(resourceA, resourceB string) error {
	return errors.WithStack(&errors.Error{
		Kind:    ErrComputeResourceConflict,
		Message: fmt.Sprintf("%s and %s resources cannot be used together", resourceA, resourceB),
	})
}

func ErrorInvalidNumberOfInfProcesses(processesPerReplica int64, numInf int64, numNeuronCores int64) error {
	acceptableProcesses := libmath.FactorsInt64(numNeuronCores)
	return errors.WithStack(&errors.Error{
		Kind:    ErrInvalidNumberOfInfProcesses,
		Message: fmt.Sprintf("cannot evenly distribute %d Inferentia %s (%d NeuronCores total) over %d processes - acceptable numbers of processes are %s", numInf, s.PluralS("ASIC", numInf), numNeuronCores, processesPerReplica, s.UserStrsOr(acceptableProcesses)),
	})
}

func ErrorInvalidNumberOfInfs(requestedInfs int64) error {
	return errors.WithStack(&errors.Error{
		Kind:    ErrInvalidNumberOfInfs,
		Message: fmt.Sprintf("cannot request %d Infs (currently only 1 Inf can be used per API replica, due to AWS's bug: https://github.com/aws/aws-neuron-sdk/issues/110)", requestedInfs),
	})
}
