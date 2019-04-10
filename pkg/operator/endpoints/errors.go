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

package endpoints

import (
	"fmt"

	s "github.com/cortexlabs/cortex/pkg/api/strings"
)

type ErrorKind int

const (
	ErrUnknown ErrorKind = iota
	ErrAuthHeaderMissing
	ErrAuthHeaderMalformed
	ErrAuthAPIError
	ErrAuthForbidden
	ErrAppNotDeployed
	ErrFormFileMustBeProvided
)

var (
	errorKinds = []string{
		"err_unknown",
		"err_auth_header_missing",
		"err_auth_header_malformed",
		"err_auth_api_error",
		"err_auth_forbidden",
		"err_app_not_deployed",
		"err_form_file_must_be_provided",
	}
)

var _ = [1]int{}[int(ErrFormFileMustBeProvided)-(len(errorKinds)-1)] // Ensure list length matches

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

func ErrorAuthHeaderMissing() error {
	return Error{
		Kind:    ErrAuthHeaderMissing,
		message: "auth header missing",
	}
}

func ErrorAuthHeaderMalformed() error {
	return Error{
		Kind:    ErrAuthHeaderMalformed,
		message: "auth header malformed",
	}
}

func ErrorAuthAPIError() error {
	return Error{
		Kind:    ErrAuthAPIError,
		message: "the operator is unable to verify user's credentials using AWS STS; export AWS_ACCESS_KEY_ID and AWS_SECRET_ACCESS_KEY, and run `./cortex-installer.sh update operator` to update the operator's AWS credentials",
	}
}

func ErrorAuthForbidden() error {
	return Error{
		Kind:    ErrAuthForbidden,
		message: "invalid AWS credentials; run `cortex configure` to configure your CLI with credentials for any IAM user in the same AWS account as the operator",
	}
}

func ErrorAppNotDeployed(appName string) error {
	return Error{
		Kind:    ErrAuthForbidden,
		message: fmt.Sprintf("app %s is not deployed", s.UserStr(appName)),
	}
}

func ErrorFormFileMustBeProvided(fileName string) error {
	return Error{
		Kind:    ErrFormFileMustBeProvided,
		message: fmt.Sprintf("request form file %s must be provided", s.UserStr(fileName)),
	}
}

func ErrorQueryParamRequired(paramNames ...string) error {
	return Error{
		Kind:    ErrFormFileMustBeProvided,
		message: fmt.Sprintf("query params required: %s", s.UserStrsOr(paramNames)),
	}
}

func ErrorPathParamRequired(paramNames ...string) error {
	return Error{
		Kind:    ErrFormFileMustBeProvided,
		message: fmt.Sprintf("path params required: %s", s.UserStrsOr(paramNames)),
	}
}
