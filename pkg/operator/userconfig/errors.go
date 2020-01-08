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

package userconfig

import (
	"fmt"

	"github.com/cortexlabs/cortex/pkg/lib/errors"
	"github.com/cortexlabs/cortex/pkg/lib/sets/strset"
	s "github.com/cortexlabs/cortex/pkg/lib/strings"
)

type ErrorKind int

const (
	ErrUnknown ErrorKind = iota
	ErrDuplicateName
	ErrMalformedConfig
	ErrUndefinedAPI
	ErrSpecifyAllOrNone
	ErrOneOfPrerequisitesNotDefined
	ErrMinReplicasGreaterThanMax
	ErrInitReplicasGreaterThanMax
	ErrInitReplicasLessThanMin
	ErrImplDoesNotExist
	ErrExternalNotFound
	ErrONNXDoesntSupportZip
	ErrInvalidTensorFlowDir
	ErrFieldMustBeDefinedForPredictorType
	ErrFieldNotSupportedByPredictorType
)

var errorKinds = []string{
	"err_unknown",
	"err_duplicate_name",
	"err_malformed_config",
	"err_undefined_api",
	"err_specify_all_or_none",
	"err_one_of_prerequisites_not_defined",
	"err_min_replicas_greater_than_max",
	"err_init_replicas_greater_than_max",
	"err_init_replicas_less_than_min",
	"err_impl_does_not_exist",
	"err_external_not_found",
	"err_onnx_doesnt_support_zip",
	"err_invalid_tensorflow_dir",
	"err_field_must_be_defined_for_predictor_type",
	"err_field_not_supported_by_predictor_type",
}

var _ = [1]int{}[int(ErrFieldNotSupportedByPredictorType)-(len(errorKinds)-1)] // Ensure list length matches

func (t ErrorKind) String() string {
	return errorKinds[t]
}

// MarshalText satisfies TextMarshaler
func (t ErrorKind) MarshalText() ([]byte, error) {
	return []byte(t.String()), nil
}

// UnmarshalText satisfies TextUnmarshaler
func (t *ErrorKind) UnmarshalText(text []byte) error {
	enum := string(text)
	for i := 0; i < len(errorKinds); i++ {
		if enum == errorKinds[i] {
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

func ErrorDuplicateName(apis []*API) error {
	filePaths := strset.New()
	for _, api := range apis {
		filePaths.Add(api.FilePath)
	}

	return errors.WithStack(Error{
		Kind:    ErrDuplicateName,
		message: fmt.Sprintf("name %s must be unique across apis (defined in %s)", s.UserStr(apis[0].Name), s.StrsAnd(filePaths.Slice())),
	})
}

func ErrorMalformedConfig() error {
	return errors.WithStack(Error{
		Kind:    ErrMalformedConfig,
		message: fmt.Sprintf("cortex YAML configuration files must contain a list of maps"),
	})
}

func ErrorUndefinedAPI(apiName string) error {
	return errors.WithStack(Error{
		Kind:    ErrUndefinedAPI,
		message: fmt.Sprintf("api %s is not defined", s.UserStr(apiName)),
	})
}

func ErrorSpecifyAllOrNone(vals ...string) error {
	message := fmt.Sprintf("please specify all or none of %s", s.UserStrsAnd(vals))
	if len(vals) == 2 {
		message = fmt.Sprintf("please specify both %s and %s or neither of them", s.UserStr(vals[0]), s.UserStr(vals[1]))
	}

	return errors.WithStack(Error{
		Kind:    ErrSpecifyAllOrNone,
		message: message,
	})
}

func ErrorOneOfPrerequisitesNotDefined(argName string, prerequisites ...string) error {
	message := fmt.Sprintf("%s specified without specifying %s", s.UserStr(argName), s.UserStrsOr(prerequisites))

	return errors.WithStack(Error{
		Kind:    ErrOneOfPrerequisitesNotDefined,
		message: message,
	})
}

func ErrorMinReplicasGreaterThanMax(min int32, max int32) error {
	return errors.WithStack(Error{
		Kind:    ErrMinReplicasGreaterThanMax,
		message: fmt.Sprintf("%s cannot be greater than %s (%d > %d)", MinReplicasKey, MaxReplicasKey, min, max),
	})
}

func ErrorInitReplicasGreaterThanMax(init int32, max int32) error {
	return errors.WithStack(Error{
		Kind:    ErrInitReplicasGreaterThanMax,
		message: fmt.Sprintf("%s cannot be greater than %s (%d > %d)", InitReplicasKey, MaxReplicasKey, init, max),
	})
}

func ErrorInitReplicasLessThanMin(init int32, min int32) error {
	return errors.WithStack(Error{
		Kind:    ErrInitReplicasLessThanMin,
		message: fmt.Sprintf("%s cannot be less than %s (%d < %d)", InitReplicasKey, MinReplicasKey, init, min),
	})
}

func ErrorImplDoesNotExist(path string) error {
	return errors.WithStack(Error{
		Kind:    ErrImplDoesNotExist,
		message: fmt.Sprintf("%s: implementation file does not exist", path),
	})
}

func ErrorExternalNotFound(path string) error {
	return errors.WithStack(Error{
		Kind:    ErrExternalNotFound,
		message: fmt.Sprintf("%s: not found or insufficient permissions", path),
	})
}

func ErrorONNXDoesntSupportZip() error {
	return errors.WithStack(Error{
		Kind:    ErrONNXDoesntSupportZip,
		message: fmt.Sprintf("zip files are not supported for ONNX models"),
	})
}

var onnxExpectedStructMessage = `For ONNX models, the path should end in .onnx`

var tfExpectedStructMessage = `For TensorFlow models, the path must contain a directory with the following structure:
  1523423423/ (Version prefix, usually a timestamp)
  ├── saved_model.pb
  └── variables/
      ├── variables.index
      ├── variables.data-00000-of-00003
      ├── variables.data-00001-of-00003
      └── variables.data-00002-of-...`

func ErrorInvalidTensorFlowDir(path string) error {
	message := "invalid TensorFlow export directory.\n"
	message += tfExpectedStructMessage
	return errors.WithStack(Error{
		Kind:    ErrInvalidTensorFlowDir,
		message: message,
	})
}

func ErrorFieldMustBeDefinedForPredictorType(fieldKey string, predictorType PredictorType) error {
	return errors.WithStack(Error{
		Kind:    ErrFieldMustBeDefinedForPredictorType,
		message: fmt.Sprintf("%s field must be defined for the %s predictor type", fieldKey, predictorType.String()),
	})
}

func ErrorFieldNotSupportedByPredictorType(fieldKey string, predictorType PredictorType) error {
	return errors.WithStack(Error{
		Kind:    ErrFieldNotSupportedByPredictorType,
		message: fmt.Sprintf("%s is not a supported field for the %s predictor type", fieldKey, predictorType.String()),
	})
}
