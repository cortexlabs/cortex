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

	"github.com/cortexlabs/cortex/pkg/lib/sets/strset"
	s "github.com/cortexlabs/cortex/pkg/lib/strings"
	"github.com/cortexlabs/cortex/pkg/operator/api/resource"
)

type ErrorKind int

const (
	ErrUnknown ErrorKind = iota
	ErrDuplicateResourceName
	ErrDuplicateConfig
	ErrMalformedConfig
	ErrParseConfig
	ErrReadConfig
	ErrMissingAppDefinition
	ErrUndefinedResource
	ErrSpecifyAllOrNone
	ErrSpecifyOnlyOne
	ErrOneOfPrerequisitesNotDefined
	ErrCannotBeNull
	ErrMinReplicasGreaterThanMax
	ErrInitReplicasGreaterThanMax
	ErrInitReplicasLessThanMin
	ErrSpecifyOnlyOneMissing
	ErrImplDoesNotExist
	ErrUnableToInferModelFormat
	ErrExternalNotFound
	ErrONNXDoesntSupportZip
	ErrInvalidTensorflowDir
	ErrTFServingOptionsForTFOnly
)

var errorKinds = []string{
	"err_unknown",
	"err_duplicate_resource_name",
	"err_duplicate_config",
	"err_malformed_config",
	"err_parse_config",
	"err_read_config",
	"err_missing_app_definition",
	"err_undefined_resource",
	"err_specify_all_or_none",
	"err_specify_only_one",
	"err_one_of_prerequisites_not_defined",
	"err_cannot_be_null",
	"err_min_replicas_greater_than_max",
	"err_init_replicas_greater_than_max",
	"err_init_replicas_less_than_min",
	"err_specify_only_one_missing",
	"err_impl_does_not_exist",
	"err_unable_to_infer_model_format",
	"err_external_not_found",
	"err_onnx_doesnt_support_zip",
	"err_invalid_tensorflow_dir",
	"err_tf_serving_options_for_tf_only",
}

var _ = [1]int{}[int(ErrTFServingOptionsForTFOnly)-(len(errorKinds)-1)] // Ensure list length matches

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

func ErrorDuplicateResourceName(resources ...Resource) error {
	filePaths := strset.New()
	resourceTypes := strset.New()

	for _, res := range resources {
		resourceTypes.Add(res.GetResourceType().Plural())
		filePaths.Add(res.GetFilePath())
	}

	return Error{
		Kind:    ErrDuplicateResourceName,
		message: fmt.Sprintf("name %s must be unique across %s (defined in %s)", s.UserStr(resources[0].GetName()), s.StrsAnd(resourceTypes.Slice()), s.StrsAnd(filePaths.Slice())),
	}
}

func ErrorDuplicateConfig(resourceType resource.Type) error {
	return Error{
		Kind:    ErrDuplicateConfig,
		message: fmt.Sprintf("%s resource may only be defined once", resourceType.String()),
	}
}

func ErrorMalformedConfig() error {
	return Error{
		Kind:    ErrMalformedConfig,
		message: fmt.Sprintf("cortex YAML configuration files must contain a list of maps"),
	}
}

func ErrorParseConfig() error {
	parseErr := Error{
		Kind:    ErrParseConfig,
		message: fmt.Sprintf("failed to parse config file"),
	}

	return parseErr
}

func ErrorReadConfig() error {
	readErr := Error{
		Kind:    ErrReadConfig,
		message: fmt.Sprintf("failed to read config file"),
	}

	return readErr
}

func ErrorMissingAppDefinition() error {
	return Error{
		Kind:    ErrMissingAppDefinition,
		message: fmt.Sprintf("cortex.yaml must define a deployment resource"),
	}
}

func ErrorUndefinedResource(resourceName string, resourceTypes ...resource.Type) error {
	message := fmt.Sprintf("%s is not defined", s.UserStr(resourceName))

	if len(resourceTypes) == 1 {
		message = fmt.Sprintf("%s %s is not defined", resourceTypes[0].String(), s.UserStr(resourceName))
	} else if len(resourceTypes) > 1 {
		message = fmt.Sprintf("%s is not defined as a %s", s.UserStr(resourceName), s.StrsOr(resource.Types(resourceTypes).StringList()))
	}

	return Error{
		Kind:    ErrUndefinedResource,
		message: message,
	}
}

func ErrorSpecifyAllOrNone(vals ...string) error {
	message := fmt.Sprintf("please specify all or none of %s", s.UserStrsAnd(vals))
	if len(vals) == 2 {
		message = fmt.Sprintf("please specify both %s and %s or neither of them", s.UserStr(vals[0]), s.UserStr(vals[1]))
	}

	return Error{
		Kind:    ErrSpecifyAllOrNone,
		message: message,
	}
}

func ErrorSpecifyOnlyOne(vals ...string) error {
	message := fmt.Sprintf("please specify exactly one of %s", s.UserStrsOr(vals))
	if len(vals) == 2 {
		message = fmt.Sprintf("please specify either %s or %s, but not both", s.UserStr(vals[0]), s.UserStr(vals[1]))
	}

	return Error{
		Kind:    ErrSpecifyOnlyOne,
		message: message,
	}
}

func ErrorOneOfPrerequisitesNotDefined(argName string, prerequisites ...string) error {
	message := fmt.Sprintf("%s specified without specifying %s", s.UserStr(argName), s.UserStrsOr(prerequisites))

	return Error{
		Kind:    ErrOneOfPrerequisitesNotDefined,
		message: message,
	}
}

func ErrorCannotBeNull() error {
	return Error{
		Kind:    ErrCannotBeNull,
		message: "cannot be null",
	}
}

func ErrorMinReplicasGreaterThanMax(min int32, max int32) error {
	return Error{
		Kind:    ErrMinReplicasGreaterThanMax,
		message: fmt.Sprintf("%s cannot be greater than %s (%d > %d)", MinReplicasKey, MaxReplicasKey, min, max),
	}
}

func ErrorInitReplicasGreaterThanMax(init int32, max int32) error {
	return Error{
		Kind:    ErrInitReplicasGreaterThanMax,
		message: fmt.Sprintf("%s cannot be greater than %s (%d > %d)", InitReplicasKey, MaxReplicasKey, init, max),
	}
}

func ErrorInitReplicasLessThanMin(init int32, min int32) error {
	return Error{
		Kind:    ErrInitReplicasLessThanMin,
		message: fmt.Sprintf("%s cannot be less than %s (%d < %d)", InitReplicasKey, MinReplicasKey, init, min),
	}
}

func ErrorSpecifyOnlyOneMissing(vals ...string) error {
	message := fmt.Sprintf("please specify one of %s", s.UserStrsOr(vals))
	if len(vals) == 2 {
		message = fmt.Sprintf("please specify either %s or %s", s.UserStr(vals[0]), s.UserStr(vals[1]))
	}

	return Error{
		Kind:    ErrSpecifyOnlyOneMissing,
		message: message,
	}
}

func ErrorImplDoesNotExist(path string) error {
	return Error{
		Kind:    ErrImplDoesNotExist,
		message: fmt.Sprintf("%s: implementation file does not exist", path),
	}
}

func ErrorExternalNotFound(path string) error {
	return Error{
		Kind:    ErrExternalNotFound,
		message: fmt.Sprintf("%s: not found or insufficient permissions", path),
	}
}

func ErrorONNXDoesntSupportZip() error {
	return Error{
		Kind:    ErrONNXDoesntSupportZip,
		message: fmt.Sprintf("zip files are not supported for ONNX models"),
	}
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

func ErrorUnableToInferModelFormat(path string) error {
	message := fmt.Sprintf("%s not specified, and could not be inferred from %s path (%s)\n%s\n%s", ModelFormatKey, ModelKey, path, onnxExpectedStructMessage, tfExpectedStructMessage)
	return Error{
		Kind:    ErrUnableToInferModelFormat,
		message: message,
	}
}

func ErrorInvalidTensorflowDir(path string) error {
	message := "invalid TF export directory.\n"
	message += tfExpectedStructMessage
	return Error{
		Kind:    ErrInvalidTensorflowDir,
		message: message,
	}
}

func ErrorTFServingOptionsForTFOnly(format ModelFormat) error {
	return Error{
		Kind:    ErrTFServingOptionsForTFOnly,
		message: fmt.Sprintf("TensorFlow serving options were provided but the model format is %s", format.String()),
	}
}
