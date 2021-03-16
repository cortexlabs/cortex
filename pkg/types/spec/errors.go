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
	"fmt"
	"regexp"

	"github.com/cortexlabs/cortex/pkg/consts"
	"github.com/cortexlabs/cortex/pkg/lib/errors"
	"github.com/cortexlabs/cortex/pkg/lib/files"
	"github.com/cortexlabs/cortex/pkg/lib/k8s"
	libmath "github.com/cortexlabs/cortex/pkg/lib/math"
	"github.com/cortexlabs/cortex/pkg/lib/sets/strset"
	s "github.com/cortexlabs/cortex/pkg/lib/strings"
	"github.com/cortexlabs/cortex/pkg/types"
	"github.com/cortexlabs/cortex/pkg/types/userconfig"
)

const (
	ErrMalformedConfig              = "spec.malformed_config"
	ErrNoAPIs                       = "spec.no_apis"
	ErrDuplicateName                = "spec.duplicate_name"
	ErrDuplicateEndpointInOneDeploy = "spec.duplicate_endpoint_in_one_deploy"
	ErrDuplicateEndpoint            = "spec.duplicate_endpoint"
	ErrConflictingFields            = "spec.conflicting_fields"
	ErrSpecifyOnlyOneField          = "spec.specify_only_one_field"
	ErrSpecifyOneOrTheOther         = "spec.specify_one_or_the_other"
	ErrSpecifyAllOrNone             = "spec.specify_all_or_none"
	ErrOneOfPrerequisitesNotDefined = "spec.one_of_prerequisites_not_defined"
	ErrConfigGreaterThanOtherConfig = "spec.config_greater_than_other_config"

	ErrMinReplicasGreaterThanMax  = "spec.min_replicas_greater_than_max"
	ErrInitReplicasGreaterThanMax = "spec.init_replicas_greater_than_max"
	ErrInitReplicasLessThanMin    = "spec.init_replicas_less_than_min"

	ErrInvalidSurgeOrUnavailable   = "spec.invalid_surge_or_unavailable"
	ErrSurgeAndUnavailableBothZero = "spec.surge_and_unavailable_both_zero"

	ErrModelCachingNotSupportedWhenMultiprocessingEnabled = "spec.model_caching_not_supported_when_multiprocessing_enabled"

	ErrShmSizeCannotExceedMem = "spec.shm_size_cannot_exceed_mem"

	ErrFileNotFound              = "spec.file_not_found"
	ErrDirIsEmpty                = "spec.dir_is_empty"
	ErrMustBeRelativeProjectPath = "spec.must_be_relative_project_path"
	ErrPythonPathNotFound        = "spec.python_path_not_found"

	ErrS3FileNotFound = "spec.s3_file_not_found"
	ErrS3DirNotFound  = "spec.s3_dir_not_found"
	ErrS3DirIsEmpty   = "spec.s3_dir_is_empty"

	ErrModelPathNotDirectory      = "spec.model_path_not_directory"
	ErrInvalidModelPathProvider   = "spec.invalid_model_path_provider"
	ErrInvalidPythonModelPath     = "spec.invalid_python_model_path"
	ErrInvalidTensorFlowModelPath = "spec.invalid_tensorflow_model_path"
	ErrInvalidONNXModelPath       = "spec.invalid_onnx_model_path"
	ErrInvalidONNXModelFilePath   = "spec.invalid_onnx_model_file_path"

	ErrDuplicateModelNames = "spec.duplicate_model_names"
	ErrReservedModelName   = "spec.reserved_model_name"

	ErrFieldMustBeDefinedForPredictorType          = "spec.field_must_be_defined_for_predictor_type"
	ErrFieldNotSupportedByPredictorType            = "spec.field_not_supported_by_predictor_type"
	ErrPredictorTypeNotSupportedForKind            = "spec.predictor_type_not_supported_by_kind"
	ErrNoAvailableNodeComputeLimit                 = "spec.no_available_node_compute_limit"
	ErrCortexPrefixedEnvVarNotAllowed              = "spec.cortex_prefixed_env_var_not_allowed"
	ErrUnsupportedComputeResourceForProvider       = "spec.unsupported_compute_resource_for_provider"
	ErrRegistryInDifferentRegion                   = "spec.registry_in_different_region"
	ErrRegistryAccountIDMismatch                   = "spec.registry_account_id_mismatch"
	ErrKindIsNotSupportedByProvider                = "spec.kind_is_not_supported_by_provider"
	ErrKeyIsNotSupportedForKind                    = "spec.key_is_not_supported_for_kind"
	ErrComputeResourceConflict                     = "spec.compute_resource_conflict"
	ErrInvalidNumberOfInfProcesses                 = "spec.invalid_number_of_inf_processes"
	ErrInvalidNumberOfInfs                         = "spec.invalid_number_of_infs"
	ErrInsufficientBatchConcurrencyLevel           = "spec.insufficient_batch_concurrency_level"
	ErrInsufficientBatchConcurrencyLevelInf        = "spec.insufficient_batch_concurrency_level_inf"
	ErrConcurrencyMismatchServerSideBatchingPython = "spec.concurrency_mismatch_server_side_batching_python"
	ErrIncorrectTrafficSplitterWeight              = "spec.incorrect_traffic_splitter_weight"
	ErrTrafficSplitterAPIsNotUnique                = "spec.traffic_splitter_apis_not_unique"
	ErrOneShadowPerTrafficSplitter                 = "spec.one_shadow_per_traffic_splitter"
	ErrUnexpectedDockerSecretData                  = "spec.unexpected_docker_secret_data"
)

var _modelCurrentStructure = `
  but its current structure is

%s`

func ErrorMalformedConfig() error {
	return errors.WithStack(&errors.Error{
		Kind:    ErrMalformedConfig,
		Message: fmt.Sprintf("cortex YAML configuration files must contain a list of maps (see https://docs.cortex.dev/v/%s/ for api configuration schema)", consts.CortexVersionMinor),
	})
}

func ErrorNoAPIs() error {
	return errors.WithStack(&errors.Error{
		Kind:    ErrNoAPIs,
		Message: fmt.Sprintf("at least one API must be configured (see https://docs.cortex.dev/v/%s/ for api configuration schema)", consts.CortexVersionMinor),
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

func ErrorSpecifyOnlyOneField(fields ...string) error {
	return errors.WithStack(&errors.Error{
		Kind:    ErrSpecifyOnlyOneField,
		Message: fmt.Sprintf("please specify only one of the following fields %s", s.UserStrsOr(fields)),
	})
}

func ErrorSpecifyOneOrTheOther(fieldKeyA, fieldKeyB string) error {
	return errors.WithStack(&errors.Error{
		Kind:    ErrSpecifyOneOrTheOther,
		Message: fmt.Sprintf("please specify either the %s field or %s field (cannot be both empty at the same time)", fieldKeyA, fieldKeyB),
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

func ErrorShmSizeCannotExceedMem(shmSize k8s.Quantity, mem k8s.Quantity) error {
	return errors.WithStack(&errors.Error{
		Kind:    ErrShmSizeCannotExceedMem,
		Message: fmt.Sprintf("predictor.shm_size (%s) cannot exceed compute.mem (%s)", shmSize.UserString, mem.UserString),
	})
}

func ErrorModelCachingNotSupportedWhenMultiprocessingEnabled(desiredProcesses int32) error {
	const maxNumProcesses int32 = 1
	return errors.WithStack(&errors.Error{
		Kind: ErrModelCachingNotSupportedWhenMultiprocessingEnabled,
		Message: fmt.Sprintf("when dynamic model caching is enabled (%s < provided models), the max value %s can take is %d, while currently it's set to %d",
			userconfig.ModelsCacheSizeKey, userconfig.ProcessesPerReplicaKey, maxNumProcesses, desiredProcesses),
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

func ErrorS3DirIsEmpty(path string) error {
	return errors.WithStack(&errors.Error{
		Kind:    ErrS3DirIsEmpty,
		Message: fmt.Sprintf("%s: S3 directory is empty", path),
	})
}

func ErrorModelPathNotDirectory(modelPath string) error {
	return errors.WithStack(&errors.Error{
		Kind:    ErrModelPathNotDirectory,
		Message: fmt.Sprintf("%s: model path must be a directory", modelPath),
	})
}

func ErrorInvalidModelPathProvider(modelPath string) error {
	return errors.WithStack(&errors.Error{
		Kind:    ErrInvalidModelPathProvider,
		Message: fmt.Sprintf("%s: model path must be an S3 path (e.g. s3://bucket/my-dir/) or a GCS path (e.g. gs://bucket/my-dir)", modelPath),
	})
}

var _pythonModelTemplates = `
  %s
  ├── 1523423423/ (Version prefix)
  |   └── * // Model-specific files (i.e. model.h5, model.pkl, labels.json, etc)
  └── 2434389194/ (Version prefix)
	  └── * // Model-specific files (i.e. model.h5, model.pkl, labels.json, etc)

or like

  %s
  └── * // Model-specific files (i.e. model.h5, model.pkl, labels.json, etc)
`

func ErrorInvalidPythonModelPath(modelPath string, modelSubPaths []string) error {
	message := fmt.Sprintf("%s: invalid %s model path. ", modelPath, userconfig.PythonPredictorType.CasedString())
	message += " " + fmt.Sprintf("For models provided for the %s predictor type, the path must be a directory with one of the following structures:\n", userconfig.PythonPredictorType)

	message += fmt.Sprintf(_pythonModelTemplates, modelPath, modelPath)

	if len(modelSubPaths) > 0 {
		message += "\n" + "but its current structure is (limited to 50 sub-paths)" + "\n\n"
		if len(modelSubPaths) > 50 {
			message += s.Indent(files.FileTree(modelSubPaths[:50], "", files.DirsSorted), "  ")
			message += "\n  ..."
		} else {
			message += s.Indent(files.FileTree(modelSubPaths, "", files.DirsSorted), "  ")
		}
	} else {
		message += "\n" + "but its current directory is empty"
	}

	return errors.WithStack(&errors.Error{
		Kind:    ErrInvalidPythonModelPath,
		Message: message,
	})
}

var _tfVersionedExpectedStructMessage = `
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
          └── variables.data-00002-of-...

or like

  %s
  ├── saved_model.pb
  └── variables/
      ├── variables.index
      ├── variables.data-00000-of-00003
      ├── variables.data-00001-of-00003
      └── variables.data-00002-of-...
`
var _neuronTfVersionedExpectedStructMessage = `
  %s
  ├── 1523423423/ (Version prefix, usually a timestamp)
  |   └── saved_model.pb
  └── 2434389194/ (Version prefix, usually a timestamp)
      └── saved_model.pb

or like

  %s
  └── saved_model.pb
`

func ErrorInvalidTensorFlowModelPath(modelPath string, neuronExport bool, modelSubPaths []string) error {
	predictorType := userconfig.TensorFlowPredictorType.CasedString()
	if neuronExport {
		predictorType = "Neuron " + predictorType
	}
	message := fmt.Sprintf("%s: invalid %s model path.", modelPath, predictorType)
	message += " " + fmt.Sprintf("For models provided for the %s predictor type, the path must be a directory with one of the following structures:\n", userconfig.TensorFlowPredictorType)

	if !neuronExport {
		message += fmt.Sprintf(_tfVersionedExpectedStructMessage, modelPath, modelPath)
	} else {
		message += fmt.Sprintf(_neuronTfVersionedExpectedStructMessage, modelPath, modelPath)
	}

	if len(modelSubPaths) > 0 {
		message += "\n" + "but its current structure is (limited to 50 sub-paths)" + "\n\n"
		if len(modelSubPaths) > 50 {
			message += s.Indent(files.FileTree(modelSubPaths[:50], "", files.DirsSorted), "  ")
			message += "\n  ..."
		} else {
			message += s.Indent(files.FileTree(modelSubPaths, "", files.DirsSorted), "  ")
		}
	} else {
		message += "\n" + "but its current directory is empty"
	}

	return errors.WithStack(&errors.Error{
		Kind:    ErrInvalidTensorFlowModelPath,
		Message: message,
	})
}

var _onnxVersionedExpectedStructMessage = `
  %s
  ├── 1523423423/ (Version prefix)
  |   └── <model-name>.onnx // ONNX-exported file
  └── 2434389194/ (Version prefix)
      └── <model-name>.onnx // ONNX-exported file

or like

  %s
  └── <model-name>.onnx // ONNX-exported file
`

func ErrorInvalidONNXModelPath(modelPath string, modelSubPaths []string) error {
	message := fmt.Sprintf("%s: invalid %s model path. ", modelPath, userconfig.ONNXPredictorType.CasedString())
	message += " " + fmt.Sprintf("For models provided for the %s predictor type, the path must be a directory with one of the following structures:\n", userconfig.PythonPredictorType)

	message += fmt.Sprintf(_onnxVersionedExpectedStructMessage, modelPath, modelPath)

	if len(modelSubPaths) > 0 {
		message += "\n" + "but its current structure is (limited to 50 sub-paths)" + "\n\n"
		if len(modelSubPaths) > 50 {
			message += s.Indent(files.FileTree(modelSubPaths[:50], "", files.DirsSorted), "  ")
			message += "\n  ..."
		} else {
			message += s.Indent(files.FileTree(modelSubPaths, "", files.DirsSorted), "  ")
		}
	} else {
		message += "\n" + "but its current directory is empty"
	}

	return errors.WithStack(&errors.Error{
		Kind:    ErrInvalidONNXModelPath,
		Message: message,
	})
}

func ErrorInvalidONNXModelFilePath(filePath string) error {
	message := fmt.Sprintf("%s: invalid %s model file path; specify an ONNX file path or provide a directory with one of the following structures:\n", filePath, userconfig.ONNXPredictorType.CasedString())
	templateModelPath := "path/to/model/directory/"
	message += fmt.Sprintf(_onnxVersionedExpectedStructMessage, templateModelPath, templateModelPath)

	return errors.WithStack(&errors.Error{
		Kind:    ErrInvalidONNXModelFilePath,
		Message: message,
	})
}

func ErrorDuplicateModelNames(duplicateModel string) error {
	return errors.WithStack(&errors.Error{
		Kind:    ErrDuplicateModelNames,
		Message: fmt.Sprintf("cannot have multiple models with the same name (%s)", duplicateModel),
	})
}

func ErrorReservedModelName(reservedModel string) error {
	return errors.WithStack(&errors.Error{
		Kind:    ErrReservedModelName,
		Message: fmt.Sprintf("%s: is a reserved name; please specify a different model name", reservedModel),
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

func ErrorUnsupportedComputeResourceForProvider(resourceType string, provider types.ProviderType) error {
	return errors.WithStack(&errors.Error{
		Kind:    ErrUnsupportedComputeResourceForProvider,
		Message: fmt.Sprintf("%s compute resources cannot be used for the %s provider", resourceType, provider.String()),
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

func ErrorKindIsNotSupportedByProvider(kind userconfig.Kind, provider types.ProviderType) error {
	return errors.WithStack(&errors.Error{
		Kind:    ErrKindIsNotSupportedByProvider,
		Message: fmt.Sprintf("%s kind is not supported on %s provider", kind.String(), provider.String()),
	})
}

func ErrorKeyIsNotSupportedForKind(key string, kind userconfig.Kind) error {
	return errors.WithStack(&errors.Error{
		Kind:    ErrKeyIsNotSupportedForKind,
		Message: fmt.Sprintf("%s key is not supported for %s kind", key, kind.String()),
	})
}

func ErrorPredictorTypeNotSupportedForKind(predictorType userconfig.PredictorType, kind userconfig.Kind) error {
	return errors.WithStack(&errors.Error{
		Kind:    ErrPredictorTypeNotSupportedForKind,
		Message: fmt.Sprintf("%s predictor type is not supported for %s kind", predictorType.String(), kind.String()),
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

func ErrorInsufficientBatchConcurrencyLevel(maxBatchSize int32, processesPerReplica int32, threadsPerProcess int32) error {
	return errors.WithStack(&errors.Error{
		Kind: ErrInsufficientBatchConcurrencyLevel,
		Message: fmt.Sprintf(
			"%s (%d) must be less than or equal to %s * %s (%d * %d = %d)",
			userconfig.MaxBatchSizeKey, maxBatchSize, userconfig.ProcessesPerReplicaKey, userconfig.ThreadsPerProcessKey, processesPerReplica, threadsPerProcess, processesPerReplica*threadsPerProcess,
		),
	})
}

func ErrorInsufficientBatchConcurrencyLevelInf(maxBatchSize int32, threadsPerProcess int32) error {
	return errors.WithStack(&errors.Error{
		Kind: ErrInsufficientBatchConcurrencyLevelInf,
		Message: fmt.Sprintf(
			"%s (%d) must be less than or equal to %s (%d)",
			userconfig.MaxBatchSizeKey, maxBatchSize, userconfig.ThreadsPerProcessKey, threadsPerProcess,
		),
	})
}

func ErrorConcurrencyMismatchServerSideBatchingPython(maxBatchsize int32, threadsPerProcess int32) error {
	return errors.WithStack(
		&errors.Error{
			Kind: ErrConcurrencyMismatchServerSideBatchingPython,
			Message: fmt.Sprintf(
				"%s (%d) must be equal to %s (%d) when using server side batching with the python predictor",
				userconfig.ThreadsPerProcessKey, threadsPerProcess, userconfig.MaxBatchSizeKey, maxBatchsize,
			),
		},
	)
}

func ErrorIncorrectTrafficSplitterWeightTotal(totalWeight int32) error {
	return errors.WithStack(&errors.Error{
		Kind:    ErrIncorrectTrafficSplitterWeight,
		Message: fmt.Sprintf("expected weights of all non-shadow apis to sum to 100 but found %d", totalWeight),
	})
}

func ErrorTrafficSplitterAPIsNotUnique(names []string) error {
	return errors.WithStack(&errors.Error{
		Kind:    ErrTrafficSplitterAPIsNotUnique,
		Message: fmt.Sprintf("%s not unique: %s", s.PluralS("api", len(names)), s.StrsSentence(names, "")),
	})
}

func ErrorOneShadowPerTrafficSplitter() error {
	return errors.WithStack(&errors.Error{
		Kind:    ErrOneShadowPerTrafficSplitter,
		Message: fmt.Sprintf("multiple shadow apis detected; only one api is allowed to be marked as a shadow"),
	})
}

var _pwRegex = regexp.MustCompile(`"password":"[^"]+"`)
var _authRegex = regexp.MustCompile(`"auth":"[^"]+"`)

func ErrorUnexpectedDockerSecretData(reason string, secretData map[string][]byte) error {
	secretDataStrMap := map[string]string{}

	for key, value := range secretData {
		valueStr := string(value)
		valueStr = _pwRegex.ReplaceAllString(valueStr, `"password":"<omitted>"`)
		valueStr = _authRegex.ReplaceAllString(valueStr, `"auth":"<omitted>"`)
		secretDataStrMap[key] = valueStr
	}

	return errors.WithStack(&errors.Error{
		Kind:    ErrUnexpectedDockerSecretData,
		Message: fmt.Sprintf("docker registry secret named \"%s\" was found, but contains unexpected data (%s); got: %s", _dockerPullSecretName, reason, s.UserStr(secretDataStrMap)),
	})
}
