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

	msg := fmt.Sprintf("%sunable to connect to your cluster in the %s environment (operator endpoint: %s)\n\n", originalErrMsg, envName, operatorURL)
	msg += "if you don't have a cluster running:\n"
	msg += fmt.Sprintf("    → if you'd like to create a cluster, run `cortex cluster up --env %s`\n", envName)
	msg += fmt.Sprintf("    → otherwise you can ignore this message, and prevent it in the future with `cortex env delete %s`\n", envName)
	msg += "\nif you have a cluster running:\n"
	msg += fmt.Sprintf("    → run `cortex cluster info --env %s` to update your environment (include `--config <cluster.yaml>` if you have a cluster configuration file)\n", envName)

	return errors.WithStack(&errors.Error{
		Kind:    ErrFailedToConnectOperator,
		Message: msg,
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
