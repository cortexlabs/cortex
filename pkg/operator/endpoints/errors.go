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

const (
	ErrAPIVersionMismatch     = "endpoints.api_version_mismatch"
	ErrHeaderMissing          = "endpoints.header_missing"
	ErrHeaderMalformed        = "endpoints.header_malformed"
	ErrAuthAPIError           = "endpoints.auth_api_error"
	ErrAuthInvalid            = "endpoints.auth_invalid"
	ErrAuthOtherAccount       = "endpoints.auth_other_account"
	ErrFormFileMustBeProvided = "endpoints.form_file_must_be_provided"
	ErrQueryParamRequired     = "endpoints.query_param_required"
	ErrPathParamRequired      = "endpoints.path_param_required"
	ErrAnyQueryParamRequired  = "endpoints.any_query_param_required"
	ErrAnyPathParamRequired   = "endpoints.any_path_param_required"
)

func ErrorAPIVersionMismatch(operatorVersion string, clientVersion string) error {
	return errors.WithStack(&errors.Error{
		Kind:    ErrAPIVersionMismatch,
		Message: fmt.Sprintf("your CLI version (%s) doesn't match your Cortex operator version (%s); please update your cluster by following the instructions at https://www.cortex.dev/cluster-management/update, or update your CLI by following the instructions at https://www.cortex.dev/install", clientVersion, operatorVersion),
	})
}

func ErrorHeaderMissing(header string) error {
	return errors.WithStack(&errors.Error{
		Kind:    ErrHeaderMissing,
		Message: fmt.Sprintf("missing %s header", header),
	})
}

func ErrorHeaderMalformed(header string) error {
	return errors.WithStack(&errors.Error{
		Kind:    ErrHeaderMalformed,
		Message: fmt.Sprintf("malformed %s header", header),
	})
}

func ErrorAuthAPIError() error {
	return errors.WithStack(&errors.Error{
		Kind:    ErrAuthAPIError,
		Message: "the operator is unable to verify user's credentials using AWS STS; export AWS_ACCESS_KEY_ID and AWS_SECRET_ACCESS_KEY, and run `cortex cluster update` to update the operator's AWS credentials",
	})
}

func ErrorAuthInvalid() error {
	return errors.WithStack(&errors.Error{
		Kind:    ErrAuthInvalid,
		Message: "invalid AWS credentials; run `cortex configure` to configure your CLI with credentials for any IAM user in the same AWS account as the operator",
	})
}

func ErrorAuthOtherAccount() error {
	return errors.WithStack(&errors.Error{
		Kind:    ErrAuthOtherAccount,
		Message: "AWS account associated with CLI AWS credentials differs from account associated with cluster AWS credentials; run `cortex configure` to configure your CLI with credentials for any IAM user in the same AWS account as your cluster",
	})
}

func ErrorFormFileMustBeProvided(fileName string) error {
	return errors.WithStack(&errors.Error{
		Kind:    ErrFormFileMustBeProvided,
		Message: fmt.Sprintf("request form file %s must be provided", s.UserStr(fileName)),
	})
}
func ErrorQueryParamRequired(param string) error {
	return errors.WithStack(&errors.Error{
		Kind:    ErrQueryParamRequired,
		Message: fmt.Sprintf("query param required: %s", param),
	})
}

func ErrorPathParamRequired(param string) error {
	return errors.WithStack(&errors.Error{
		Kind:    ErrPathParamRequired,
		Message: fmt.Sprintf("path param required: %s", param),
	})
}

func ErrorAnyQueryParamRequired(param string, params ...string) error {
	allParams := append(params, param)
	return errors.WithStack(&errors.Error{
		Kind:    ErrAnyQueryParamRequired,
		Message: fmt.Sprintf("query params required: %s", s.UserStrsOr(allParams)),
	})
}

func ErrorAnyPathParamRequired(param string, params ...string) error {
	allParams := append(params, param)
	return errors.WithStack(&errors.Error{
		Kind:    ErrAnyPathParamRequired,
		Message: fmt.Sprintf("path params required: %s", s.UserStrsOr(allParams)),
	})
}
