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

package cluster

import (
	"fmt"
	"net/url"
	"strings"

	"github.com/cortexlabs/cortex/pkg/lib/errors"
	"github.com/cortexlabs/cortex/pkg/lib/urls"
)

const (
	_errStrCantMakeRequest = "unable to make request"
	_errStrRead            = "unable to read"
)

func errStrFailedToConnect(u url.URL) string {
	return "failed to connect to " + urls.TrimQueryParamsURL(u)
}

const (
	ErrFailedToConnectOperator       = "cli.failed_to_connect_operator"
	ErrOperatorSocketRead            = "cli.operator_socket_read"
	ErrResponseUnknown               = "cli.response_unknown"
	ErrOperatorResponseUnknown       = "cli.operator_response_unknown"
	ErrOperatorStreamResponseUnknown = "cli.operator_stream_response_unknown"
)

func ErrorFailedToConnectOperator(originalError error, envName string, operatorURL string) error {
	originalErrMsg := ""
	if originalError != nil {
		originalErrMsg = urls.TrimQueryParamsStr(errors.Message(originalError)) + "\n\n"
	}

	return errors.WithStack(&errors.Error{
		Kind:    ErrFailedToConnectOperator,
		Message: fmt.Sprintf("%sfailed to connect to the operator in the %s environment (operator endpoint: %s); run `cortex env configure %s` if you need to update the operator endpoint, `cortex cluster info` to show your operator endpoint, or `cortex cluster up` to create a new cluster", originalErrMsg, envName, operatorURL, envName),
	})
}

func ErrorOperatorSocketRead(err error) error {
	return errors.WithStack(&errors.Error{
		Kind:    ErrOperatorSocketRead,
		Message: err.Error(),
		NoPrint: true,
	})
}

func ErrorResponseUnknown(body string, statusCode int) error {
	msg := body
	if strings.TrimSpace(body) == "" {
		msg = fmt.Sprintf("empty response (status code %d)", statusCode)
	}

	return errors.WithStack(&errors.Error{
		Kind:    ErrResponseUnknown,
		Message: msg,
	})
}

func ErrorOperatorResponseUnknown(body string, statusCode int) error {
	return errors.WithStack(&errors.Error{
		Kind:    ErrOperatorResponseUnknown,
		Message: fmt.Sprintf("unexpected response from operator (status code %d): %s", statusCode, body),
	})
}

func ErrorOperatorStreamResponseUnknown(body string, statusCode int) error {
	return errors.WithStack(&errors.Error{
		Kind:    ErrOperatorStreamResponseUnknown,
		Message: fmt.Sprintf("unexpected response from operator (status code %d): %s", statusCode, body),
	})
}
