/*
Copyright 2019 Cortex Labs, Inc.

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

type ErrorKind int

const (
	ErrUnknown ErrorKind = iota
	ErrCortexInstallationBroken
	ErrLoadBalancerInitializing
	ErrMalformedConfig
	ErrNoAPIs
	ErrDuplicateName
	ErrSpecifyAllOrNone
	ErrOneOfPrerequisitesNotDefined
	ErrMinReplicasGreaterThanMax
	ErrInitReplicasGreaterThanMax
	ErrInitReplicasLessThanMin
	ErrImplDoesNotExist
	ErrS3FileNotFound
	ErrS3DirNotFoundOrEmpty
	ErrONNXDoesntSupportZip
	ErrInvalidTensorFlowDir
	ErrFieldMustBeDefinedForPredictorType
	ErrFieldNotSupportedByPredictorType
	ErrNoAvailableNodeComputeLimit
	ErrAPINotDeployed
	ErrDuplicateEndpoint
)

var _errorKinds = []string{
	"err_unknown",
	"err_cortex_installation_broken",
	"err_load_balancer_initializing",
	"err_malformed_config",
	"err_no_apis",
	"err_duplicate_name",
	"err_specify_all_or_none",
	"err_one_of_prerequisites_not_defined",
	"err_min_replicas_greater_than_max",
	"err_init_replicas_greater_than_max",
	"err_init_replicas_less_than_min",
	"err_impl_does_not_exist",
	"err_s3_file_not_found",
	"err_s3_dir_not_found_or_empty",
	"err_onnx_doesnt_support_zip",
	"err_invalid_tensorflow_dir",
	"err_field_must_be_defined_for_predictor_type",
	"err_field_not_supported_by_predictor_type",
	"err_no_available_node_compute_limit",
	"err_api_not_deployed",
	"err_duplicate_endpoint",
}

var _ = [1]int{}[int(ErrDuplicateEndpoint)-(len(_errorKinds)-1)] // Ensure list length matches

func (t ErrorKind) String() string {
	return _errorKinds[t]
}

// MarshalText satisfies TextMarshaler
func (t ErrorKind) MarshalText() ([]byte, error) {
	return []byte(t.String()), nil
}

// UnmarshalText satisfies TextUnmarshaler
func (t *ErrorKind) UnmarshalText(text []byte) error {
	enum := string(text)
	for i := 0; i < len(_errorKinds); i++ {
		if enum == _errorKinds[i] {
			*t = ErrorKind(i)
			return nil
		}
	}

	*t = ErrUnknown
	return nil
}

// UnmarshalBinary satisfies BinaryUnmarshaler
// Needed for msgpack
func (t *ErrorKind) UnmarshalBinary(data []byte) error {
	return t.UnmarshalText(data)
}

// MarshalBinary satisfies BinaryMarshaler
func (t ErrorKind) MarshalBinary() ([]byte, error) {
	return []byte(t.String()), nil
}

type Error struct {
	Kind    ErrorKind
	message string
}

func (e Error) Error() string {
	return e.message
}

func ErrorCortexInstallationBroken() error {
	return errors.WithStack(Error{
		Kind:    ErrCortexInstallationBroken,
		message: "cortex is out of date, or not installed properly on your cluster; run `cortex cluster update`",
	})
}

func ErrorLoadBalancerInitializing() error {
	return errors.WithStack(Error{
		Kind:    ErrLoadBalancerInitializing,
		message: "load balancer is still initializing",
	})
}

func ErrorMalformedConfig() error {
	return errors.WithStack(Error{
		Kind:    ErrMalformedConfig,
		message: fmt.Sprintf("cortex YAML configuration files must contain a list of maps (see https://cortex.dev for documentation)"),
	})
}

func ErrorNoAPIs() error {
	return errors.WithStack(Error{
		Kind:    ErrNoAPIs,
		message: fmt.Sprintf("at least one API must be configured (see https://cortex.dev for documentation)"),
	})
}

func ErrorDuplicateName(apis []userconfig.API) error {
	filePaths := strset.New()
	for _, api := range apis {
		filePaths.Add(api.FilePath)
	}

	return errors.WithStack(Error{
		Kind:    ErrDuplicateName,
		message: fmt.Sprintf("name %s must be unique across apis (defined in %s)", s.UserStr(apis[0].Name), s.StrsAnd(filePaths.Slice())),
	})
}

func ErrorSpecifyAllOrNone(val string, vals ...string) error {
	allVals := append(vals, val)
	message := fmt.Sprintf("please specify all or none of %s", s.UserStrsAnd(allVals))
	if len(allVals) == 2 {
		message = fmt.Sprintf("please specify both %s and %s or neither of them", s.UserStr(allVals[0]), s.UserStr(allVals[1]))
	}

	return errors.WithStack(Error{
		Kind:    ErrSpecifyAllOrNone,
		message: message,
	})
}

func ErrorOneOfPrerequisitesNotDefined(argName string, prerequisite string, prerequisites ...string) error {
	allPrerequisites := append(prerequisites, prerequisite)
	message := fmt.Sprintf("%s specified without specifying %s", s.UserStr(argName), s.UserStrsOr(allPrerequisites))

	return errors.WithStack(Error{
		Kind:    ErrOneOfPrerequisitesNotDefined,
		message: message,
	})
}

func ErrorMinReplicasGreaterThanMax(min int32, max int32) error {
	return errors.WithStack(Error{
		Kind:    ErrMinReplicasGreaterThanMax,
		message: fmt.Sprintf("%s cannot be greater than %s (%d > %d)", userconfig.MinReplicasKey, userconfig.MaxReplicasKey, min, max),
	})
}

func ErrorInitReplicasGreaterThanMax(init int32, max int32) error {
	return errors.WithStack(Error{
		Kind:    ErrInitReplicasGreaterThanMax,
		message: fmt.Sprintf("%s cannot be greater than %s (%d > %d)", userconfig.InitReplicasKey, userconfig.MaxReplicasKey, init, max),
	})
}

func ErrorInitReplicasLessThanMin(init int32, min int32) error {
	return errors.WithStack(Error{
		Kind:    ErrInitReplicasLessThanMin,
		message: fmt.Sprintf("%s cannot be less than %s (%d < %d)", userconfig.InitReplicasKey, userconfig.MinReplicasKey, init, min),
	})
}

func ErrorImplDoesNotExist(path string) error {
	return errors.WithStack(Error{
		Kind:    ErrImplDoesNotExist,
		message: fmt.Sprintf("%s: implementation file does not exist", path),
	})
}

func ErrorS3FileNotFound(path string) error {
	return errors.WithStack(Error{
		Kind:    ErrS3FileNotFound,
		message: fmt.Sprintf("%s: not found or insufficient permissions", path),
	})
}

func ErrorS3DirNotFoundOrEmpty(path string) error {
	return errors.WithStack(Error{
		Kind:    ErrS3DirNotFoundOrEmpty,
		message: fmt.Sprintf("%s: dir not found, empty or insufficient permissions", path),
	})
}

func ErrorONNXDoesntSupportZip() error {
	return errors.WithStack(Error{
		Kind:    ErrONNXDoesntSupportZip,
		message: fmt.Sprintf("zip files are not supported for ONNX models"),
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
	return errors.WithStack(Error{
		Kind:    ErrInvalidTensorFlowDir,
		message: message,
	})
}

func ErrorFieldMustBeDefinedForPredictorType(fieldKey string, predictorType userconfig.PredictorType) error {
	return errors.WithStack(Error{
		Kind:    ErrFieldMustBeDefinedForPredictorType,
		message: fmt.Sprintf("%s field must be defined for the %s predictor type", fieldKey, predictorType.String()),
	})
}

func ErrorFieldNotSupportedByPredictorType(fieldKey string, predictorType userconfig.PredictorType) error {
	return errors.WithStack(Error{
		Kind:    ErrFieldNotSupportedByPredictorType,
		message: fmt.Sprintf("%s is not a supported field for the %s predictor type", fieldKey, predictorType.String()),
	})
}

func ErrorNoAvailableNodeComputeLimit(resource string, reqStr string, maxStr string) error {
	message := fmt.Sprintf("no available nodes can satisfy the requested %s quantity - requested %s %s but nodes only have %s available %s", resource, reqStr, resource, maxStr, resource)
	if maxStr == "0" {
		message = fmt.Sprintf("no available nodes can satisfy the requested %s quantity - requested %s %s but nodes don't have any %s", resource, reqStr, resource, resource)
	}
	return errors.WithStack(Error{
		Kind:    ErrNoAvailableNodeComputeLimit,
		message: message,
	})
}

func ErrorAPINotDeployed(apiName string) error {
	return errors.WithStack(Error{
		Kind:    ErrAPINotDeployed,
		message: fmt.Sprintf("%s api is not deployed", apiName), // note: if modifying this string, search the codebase for it and change all occurrences
	})
}

func ErrorDuplicateEndpoint(apiName string) error {
	return errors.WithStack(Error{
		Kind:    ErrDuplicateEndpoint,
		message: fmt.Sprintf("endpoint is already being used by the %s api", apiName),
	})
}
