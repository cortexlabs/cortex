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

package cmd

import (
	"fmt"

	s "github.com/cortexlabs/cortex/pkg/api/strings"
)

type ErrorKind int

const (
	ErrUnknown ErrorKind = iota
	ErrCliAlreadyInAppDir
	ErrAPINotReady
	ErrAPINotFound
	ErrFailedToConnect
	ErrCwdDirExists
	ErrCreateFile
	ErrReadFile
	ErrCliNotInAppDir
	ErrUnmarshalJSON
	ErrMarshalJSON
	ErrCantMakeRequest
	ErrRead
)

var errorKinds = []string{
	"err_unknown",
	"err_cli_already_in_app_dir",
	"err_api_not_ready",
	"err_api_not_found",
	"err_failed_to_connect",
	"err_cwd_dir_exists",
	"err_create_file",
	"err_read_file",
	"err_cli_not_in_app_dir",
	"err_unmarshal_json",
	"err_marshal_json",
	"err_cant_make_request",
	"err_read",
}

var _ = [1]int{}[int(ErrRead)-(len(errorKinds)-1)] // Ensure list length matches

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

func ErrorCliAlreadyInAppDir(dirPath string) error {
	return Error{
		Kind:    ErrCliAlreadyInAppDir,
		message: fmt.Sprintf("your current working directory is already in a cortex app directory (%s)", dirPath),
	}
}

func ErrorAPINotReady(apiName string, status string) error {
	return Error{
		Kind:    ErrAPINotReady,
		message: fmt.Sprintf("api %s is %s", s.UserStr(apiName), status),
	}
}

func ErrorAPINotFound(apiName string) error {
	return Error{
		Kind:    ErrAPINotFound,
		message: fmt.Sprintf("api %s not found", s.UserStr(apiName)),
	}
}

func ErrorFailedToConnect(urlStr string) error {
	return Error{
		Kind:    ErrFailedToConnect,
		message: fmt.Sprintf("failed to connect to the operator (%s), run `cortex configure` if you need to update the operator URL", urlStr),
	}
}

func ErrorCwdDirExists(dirName string) error {
	return Error{
		Kind:    ErrCwdDirExists,
		message: fmt.Sprintf("a directory named %s already exists in your current working directory", s.UserStr(dirName)),
	}
}

func ErrorCreateFile(path string) error {
	return Error{
		Kind:    ErrCreateFile,
		message: fmt.Sprintf("%s: unable to create file", path),
	}
}

func ErrorReadFile(path string) error {
	return Error{
		Kind:    ErrReadFile,
		message: fmt.Sprintf("%s: unable to read file", path),
	}
}

func ErrorCliNotInAppDir() error {
	return Error{
		Kind:    ErrCliNotInAppDir,
		message: "your current working directory is not in or under a cortex app directory (identified via a top-level app.yaml file)",
	}
}
func ErrorUnmarshalJSON() error {
	return Error{
		Kind:    ErrUnmarshalJSON,
		message: "invalid json",
	}
}

func ErrorMarshalJSON() error {
	return Error{
		Kind:    ErrMarshalJSON,
		message: "invalid json cannot be serialized",
	}
}

func ErrorCantMakeRequest() error {
	return Error{
		Kind:    ErrCantMakeRequest,
		message: "unable to make request",
	}
}

func ErrorRead() error {
	return Error{
		Kind:    ErrRead,
		message: "unable to read",
	}
}
