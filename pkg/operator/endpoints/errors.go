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

package endpoints

import (
	"fmt"

	"github.com/cortexlabs/cortex/pkg/lib/errors"
	s "github.com/cortexlabs/cortex/pkg/lib/strings"
)

type ErrorKind int

const (
	ErrUnknown ErrorKind = iota
	ErrAPIVersionMismatch
	ErrAuthHeaderMissing
	ErrAuthHeaderMalformed
	ErrAuthAPIError
	ErrAuthInvalid
	ErrAuthOtherAccount
	ErrFormFileMustBeProvided
	ErrQueryParamRequired
	ErrPathParamRequired
	ErrAnyQueryParamRequired
	ErrAnyPathParamRequired
)

var _errorKinds = []string{
	"err_unknown",
	"err_api_version_mismatch",
	"err_auth_header_missing",
	"err_auth_header_malformed",
	"err_auth_api_error",
	"err_auth_invalid",
	"err_auth_other_account",
	"err_form_file_must_be_provided",
	"err_query_param_required",
	"err_path_param_required",
	"err_any_query_param_required",
	"err_any_path_param_required",
}

var _ = [1]int{}[int(ErrAnyPathParamRequired)-(len(_errorKinds)-1)] // Ensure list length matches

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

func ErrorAPIVersionMismatch(operatorVersion string, clientVersion string) error {
	return errors.WithStack(&errors.CortexError{
		Kind:    ErrAPIVersionMismatch,
		Message: fmt.Sprintf("API version mismatch (Cluster: %s; Client: %s)", operatorVersion, clientVersion),
		User:    true,
	})
}

func ErrorAuthHeaderMissing() error {
	return errors.WithStack(&errors.CortexError{
		Kind:    ErrAuthHeaderMissing,
		Message: "auth header missing",
		User:    true,
	})
}

func ErrorAuthHeaderMalformed() error {
	return errors.WithStack(&errors.CortexError{
		Kind:    ErrAuthHeaderMalformed,
		Message: "auth header malformed",
		User:    true,
	})
}

func ErrorAuthAPIError() error {
	return errors.WithStack(&errors.CortexError{
		Kind:    ErrAuthAPIError,
		Message: "the operator is unable to verify user's credentials using AWS STS; export AWS_ACCESS_KEY_ID and AWS_SECRET_ACCESS_KEY, and run `cortex cluster update` to update the operator's AWS credentials",
		User:    true,
	})
}

func ErrorAuthInvalid() error {
	return errors.WithStack(&errors.CortexError{
		Kind:    ErrAuthInvalid,
		Message: "invalid AWS credentials; run `cortex configure` to configure your CLI with credentials for any IAM user in the same AWS account as the operator",
		User:    true,
	})
}

func ErrorAuthOtherAccount() error {
	return errors.WithStack(&errors.CortexError{
		Kind:    ErrAuthOtherAccount,
		Message: "AWS account associated with CLI AWS credentials differs from account associated with cluster AWS credentials; run `cortex configure` to configure your CLI with credentials for any IAM user in the same AWS account as your cluster",
		User:    true,
	})
}

func ErrorFormFileMustBeProvided(fileName string) error {
	return errors.WithStack(&errors.CortexError{
		Kind:    ErrFormFileMustBeProvided,
		Message: fmt.Sprintf("request form file %s must be provided", s.UserStr(fileName)),
		User:    true,
	})
}
func ErrorQueryParamRequired(param string) error {
	return errors.WithStack(&errors.CortexError{
		Kind:    ErrQueryParamRequired,
		Message: fmt.Sprintf("query param required: %s", param),
		User:    true,
	})
}

func ErrorPathParamRequired(param string) error {
	return errors.WithStack(&errors.CortexError{
		Kind:    ErrPathParamRequired,
		Message: fmt.Sprintf("path param required: %s", param),
		User:    true,
	})
}

func ErrorAnyQueryParamRequired(param string, params ...string) error {
	allParams := append(params, param)
	return errors.WithStack(&errors.CortexError{
		Kind:    ErrAnyQueryParamRequired,
		Message: fmt.Sprintf("query params required: %s", s.UserStrsOr(allParams)),
		User:    true,
	})
}

func ErrorAnyPathParamRequired(param string, params ...string) error {
	allParams := append(params, param)
	return errors.WithStack(&errors.CortexError{
		Kind:    ErrAnyPathParamRequired,
		Message: fmt.Sprintf("path params required: %s", s.UserStrsOr(allParams)),
		User:    true,
	})
}
