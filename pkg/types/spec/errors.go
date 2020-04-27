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

	"github.com/cortexlabs/cortex/pkg/lib/errors"
	"github.com/cortexlabs/cortex/pkg/lib/sets/strset"
	s "github.com/cortexlabs/cortex/pkg/lib/strings"
	"github.com/cortexlabs/cortex/pkg/types/userconfig"
)

const (
	ErrMalformedConfig                      = "spec.malformed_config"
	ErrNoAPIs                               = "spec.no_apis"
	ErrDuplicateName                        = "spec.duplicate_name"
	ErrDuplicateEndpointInOneDeploy         = "spec.duplicate_endpoint_in_one_deploy"
	ErrDuplicateEndpoint                    = "spec.duplicate_endpoint"
	ErrSpecifyAllOrNone                     = "spec.specify_all_or_none"
	ErrOneOfPrerequisitesNotDefined         = "spec.one_of_prerequisites_not_defined"
	ErrMinReplicasGreaterThanMax            = "spec.min_replicas_greater_than_max"
	ErrInitReplicasGreaterThanMax           = "spec.init_replicas_greater_than_max"
	ErrInitReplicasLessThanMin              = "spec.init_replicas_less_than_min"
	ErrInvalidSurgeOrUnavailable            = "spec.invalid_surge_or_unavailable"
	ErrSurgeAndUnavailableBothZero          = "spec.surge_and_unavailable_both_zero"
	ErrImplDoesNotExist                     = "spec.impl_does_not_exist"
	ErrFileNotFound                         = "spec.file_not_found"
	ErrDirIsEmpty                           = "spec.dir_is_empty"
	ErrS3FileNotFound                       = "spec.s3_file_not_found"
	ErrInvalidTensorFlowDir                 = "spec.invalid_tensorflow_dir"
	ErrInvalidTensorFlowModelPath           = "spec.invalid_tensorflow_model_path"
	ErrInvalidONNXModelPath                 = "spec.invalid_onnx_model_path"
	ErrFieldMustBeDefinedForPredictorType   = "spec.field_must_be_defined_for_predictor_type"
	ErrFieldNotSupportedByPredictorType     = "spec.field_not_supported_by_predictor_type"
	ErrNoAvailableNodeComputeLimit          = "spec.no_available_node_compute_limit"
	ErrCortexPrefixedEnvVarNotAllowed       = "spec.cortex_prefixed_env_var_not_allowed"
	ErrLocalPathNotSupportedByAWSProvider   = "spec.local_path_not_supported_by_aws_provider"
	ErrRegistryInDifferentRegion            = "spec.registry_in_different_region"
	ErrRegistryAccountIDMismatch            = "spec.registry_account_id_mismatch"
	ErrCannotAccessECRWithAnonymousAWSCreds = "spec.cannot_access_ecr_with_anonymous_aws_creds"
)

func ErrorMalformedConfig() error {
	return errors.WithStack(&errors.Error{
		Kind:    ErrMalformedConfig,
		Message: fmt.Sprintf("cortex YAML configuration files must contain a list of maps (see https://cortex.dev for documentation)"),
	})
}

func ErrorNoAPIs() error {
	return errors.WithStack(&errors.Error{
		Kind:    ErrNoAPIs,
		Message: fmt.Sprintf("at least one API must be configured (see https://cortex.dev for documentation)"),
	})
}

func ErrorDuplicateName(apis []userconfig.API) error {
	filePaths := strset.New()
	for _, api := range apis {
		filePaths.Add(api.FilePath)
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
		Message: fmt.Sprintf("endpoint %s must be unique across apis (defined in %s)", s.UserStr(*apis[0].Endpoint), s.StrsAnd(names)),
	})
}

func ErrorDuplicateEndpoint(apiName string) error {
	return errors.WithStack(&errors.Error{
		Kind:    ErrDuplicateEndpoint,
		Message: fmt.Sprintf("endpoint is already being used by %s", apiName),
	})
}

func ErrorSpecifyAllOrNone(val string, vals ...string) error {
	allVals := append(vals, val)
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
	allPrerequisites := append(prerequisites, prerequisite)
	message := fmt.Sprintf("%s specified without specifying %s", s.UserStr(argName), s.UserStrsOr(allPrerequisites))

	return errors.WithStack(&errors.Error{
		Kind:    ErrOneOfPrerequisitesNotDefined,
		Message: message,
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

func ErrorImplDoesNotExist(path string) error {
	return errors.WithStack(&errors.Error{
		Kind:    ErrImplDoesNotExist,
		Message: fmt.Sprintf("%s: implementation file does not exist", path),
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

func ErrorS3FileNotFound(path string) error {
	return errors.WithStack(&errors.Error{
		Kind:    ErrS3FileNotFound,
		Message: fmt.Sprintf("%s: not found or insufficient permissions", path),
	})
}

var _tfExpectedStructMessage = `For TensorFlow models, the path must contain a directory with the following structure:
  1523423423/ (Version prefix, usually a timestamp)
  ├── saved_model.pb
  └── variables/
      ├── variables.index
      ├── variables.data-00000-of-00003
      ├── variables.data-00001-of-00003
      └── variables.data-00002-of-...`

func ErrorInvalidTensorFlowDir(path string) error {
	message := "invalid TensorFlow export directory.\n"
	message += _tfExpectedStructMessage
	return errors.WithStack(&errors.Error{
		Kind:    ErrInvalidTensorFlowDir,
		Message: message,
	})
}

func ErrorInvalidTensorFlowModelPath() error {
	return errors.WithStack(&errors.Error{
		Kind:    ErrInvalidTensorFlowModelPath,
		Message: "TensorFlow model path must be a directory or a zip file ending in `.zip`",
	})
}

func ErrorInvalidONNXModelPath() error {
	return errors.WithStack(&errors.Error{
		Kind:    ErrInvalidONNXModelPath,
		Message: "onnx model path must be an onnx exported file ending in `.onnx`",
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
		Message: fmt.Sprintf("cannot access ECR with anonymous aws credentials; run `cortex env configure <env_name>` to specify AWS credentials with access to ECR"),
	})
}
