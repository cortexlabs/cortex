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

	"github.com/cortexlabs/cortex/pkg/lib/errors"
	"github.com/cortexlabs/cortex/pkg/lib/sets/strset"
	s "github.com/cortexlabs/cortex/pkg/lib/strings"
	"github.com/cortexlabs/cortex/pkg/types/userconfig"
)

const (
	ErrCortexInstallationBroken           = "operator.cortex_installation_broken"
	ErrLoadBalancerInitializing           = "operator.load_balancer_initializing"
	ErrMalformedConfig                    = "operator.malformed_config"
	ErrNoAPIs                             = "operator.no_apis"
	ErrAPIUpdating                        = "operator.api_updating"
	ErrDuplicateName                      = "operator.duplicate_name"
	ErrDuplicateEndpointInOneDeploy       = "operator.duplicate_endpoint_in_one_deploy"
	ErrDuplicateEndpoint                  = "operator.duplicate_endpoint"
	ErrSpecifyAllOrNone                   = "operator.specify_all_or_none"
	ErrOneOfPrerequisitesNotDefined       = "operator.one_of_prerequisites_not_defined"
	ErrMinReplicasGreaterThanMax          = "operator.min_replicas_greater_than_max"
	ErrInitReplicasGreaterThanMax         = "operator.init_replicas_greater_than_max"
	ErrInitReplicasLessThanMin            = "operator.init_replicas_less_than_min"
	ErrInvalidSurgeOrUnavailable          = "operator.invalid_surge_or_unavailable"
	ErrSurgeAndUnavailableBothZero        = "operator.surge_and_unavailable_both_zero"
	ErrImplDoesNotExist                   = "operator.impl_does_not_exist"
	ErrS3FileNotFound                     = "operator.s3_file_not_found"
	ErrS3DirNotFoundOrEmpty               = "operator.s3_dir_not_found_or_empty"
	ErrONNXDoesntSupportZip               = "operator.onnx_doesnt_support_zip"
	ErrInvalidTensorFlowDir               = "operator.invalid_tensorflow_dir"
	ErrFieldMustBeDefinedForPredictorType = "operator.field_must_be_defined_for_predictor_type"
	ErrFieldNotSupportedByPredictorType   = "operator.field_not_supported_by_predictor_type"
	ErrNoAvailableNodeComputeLimit        = "operator.no_available_node_compute_limit"
	ErrCortexPrefixedEnvVarNotAllowed     = "operator.cortex_prefixed_env_var_not_allowed"
	ErrAPINotDeployed                     = "operator.api_not_deployed"
	ErrInvalidOverriddenImage             = "operator.invalid_overridden_image"
)

func ErrorCortexInstallationBroken() error {
	return errors.WithStack(&errors.Error{
		Kind:    ErrCortexInstallationBroken,
		Message: "cortex is out of date or not installed properly; run `cortex cluster update`, or spin down your cluster with `cortex cluster down` and create a new one with `cortex cluster up`",
	})
}

func ErrorLoadBalancerInitializing() error {
	return errors.WithStack(&errors.Error{
		Kind:    ErrLoadBalancerInitializing,
		Message: "load balancer is still initializing",
	})
}

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

func ErrorAPIUpdating(apiName string) error {
	return errors.WithStack(&errors.Error{
		Kind:    ErrAPIUpdating,
		Message: fmt.Sprintf("%s is updating (override with --force)", apiName),
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

func ErrorS3FileNotFound(path string) error {
	return errors.WithStack(&errors.Error{
		Kind:    ErrS3FileNotFound,
		Message: fmt.Sprintf("%s: not found or insufficient permissions", path),
	})
}

func ErrorS3DirNotFoundOrEmpty(path string) error {
	return errors.WithStack(&errors.Error{
		Kind:    ErrS3DirNotFoundOrEmpty,
		Message: fmt.Sprintf("%s: directory not found or empty", path),
	})
}

func ErrorONNXDoesntSupportZip() error {
	return errors.WithStack(&errors.Error{
		Kind:    ErrONNXDoesntSupportZip,
		Message: fmt.Sprintf("zip files are not supported for ONNX models"),
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

func ErrorNoAvailableNodeComputeLimit(resource string, reqStr string, maxStr string) error {
	message := fmt.Sprintf("no instances can satisfy the requested %s quantity - requested %s %s but instances only have %s %s available", resource, reqStr, resource, maxStr, resource)
	if maxStr == "0" {
		message = fmt.Sprintf("no instances can satisfy the requested %s quantity - requested %s %s but instances don't have any %s", resource, reqStr, resource, resource)
	}
	return errors.WithStack(&errors.Error{
		Kind:    ErrNoAvailableNodeComputeLimit,
		Message: message,
	})
}

func ErrorCortexPrefixedEnvVarNotAllowed() error {
	return errors.WithStack(&errors.Error{
		Kind:    ErrCortexPrefixedEnvVarNotAllowed,
		Message: fmt.Sprintf("environment variables starting with CORTEX_ are reserved"),
	})
}

func ErrorAPINotDeployed(apiName string) error {
	return errors.WithStack(&errors.Error{
		Kind:    ErrAPINotDeployed,
		Message: fmt.Sprintf("%s is not deployed", apiName), // note: if modifying this string, search the codebase for it and change all occurrences
	})
}

func ErrorInvalidOverriddenImage(image string) error {
	return errors.WithStack(&errors.Error{
		Kind:    ErrInvalidOverriddenImage,
		Message: fmt.Sprintf("%s is not accessible", image),
	})
}
