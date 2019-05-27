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

package cloud

import (
	"fmt"

	"gocloud.dev/gcerrors"

	s "github.com/cortexlabs/cortex/pkg/lib/strings"
)

type ErrorKind int

const (
	ErrUnknown ErrorKind = iota
	ErrRequiredFieldNotDefined
	ErrInvalidHadoopPath
	ErrUnsupportedProviderType
	ErrUnrecognizedFilepath
	ErrFailedToListBlobs
)

var errorKinds = []string{
	"err_unknown",
	"err_required_field_not_defined",
	"err_invalid_hadoop_path",
	"err_unsupported_provider_type",
	"err_unrecognized_filepath",
	"err_failed_to_list_blobs",
}

var _ = [1]int{}[int(ErrFailedToListBlobs)-(len(errorKinds)-1)] // Ensure list length matches

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

func IsNoSuchKeyErr(err error) bool {
	return gcerrors.Code(err) == gcerrors.NotFound
}

type Error struct {
	Kind    ErrorKind
	message string
}

func (e Error) Error() string {
	return e.message
}

func ErrorInvalidHadoopPath(provided string) error {
	return Error{
		Kind:    ErrInvalidHadoopPath,
		message: fmt.Sprintf("%s is not a valid hadoop path", s.UserStr(provided)),
	}
}

func ErrorRequiredFieldNotDefined(field string) error {
	return Error{
		Kind:    ErrRequiredFieldNotDefined,
		message: fmt.Sprintf("%s is not defined", field),
	}
}

func ErrorUnsupportedProviderType(provider string) error {
	return Error{
		Kind:    ErrUnsupportedProviderType,
		message: fmt.Sprintf("%s is not one of the supported cloud provider types (%s)", provider, s.UserStrsOr(ProviderTypeStrings())),
	}
}

func ErrorUnrecognizedFilepath() error {
	return Error{
		Kind:    ErrUnrecognizedFilepath,
		message: "file path not recognized",
	}
}

func ErrorFailedToListBlobs() error {
	return Error{
		Kind:    ErrFailedToListBlobs,
		message: "failed to list blobs",
	}
}
