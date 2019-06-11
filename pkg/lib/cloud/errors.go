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
	ErrNotAnAbsolutePath
	ErrFailedToListBlobs
	ErrUnsupportedFilePath
)

var errorKinds = []string{
	"err_unknown",
	"err_required_field_not_defined",
	"err_invalid_hadoop_path",
	"err_unsupported_provider_type",
	"err_not_an_absolute_path",
	"err_failed_to_list_blobs",
	"err_unsupported_file_path",
}

var _ = [1]int{}[int(ErrUnsupportedFilePath)-(len(errorKinds)-1)] // Ensure list length matches

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

func ErrorNotAnAbsolutePath(path string) error {
	return Error{
		Kind:    ErrNotAnAbsolutePath,
		message: fmt.Sprintf("%s is not an absolute file path", path),
	}
}

func ErrorFailedToListBlobs() error {
	return Error{
		Kind:    ErrFailedToListBlobs,
		message: "failed to list blobs",
	}
}

func ErrorUnsupportedFilePath(path string, scheme string, schemeList ...string) error {
	schemeList = append(schemeList, scheme)
	return Error{
		Kind:    ErrUnsupportedFilePath,
		message: fmt.Sprintf("expected path of type %s but found %s", s.StrsOr(schemeList), path),
	}
}
