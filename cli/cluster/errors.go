/*
Copyright 2021 Cortex Labs, Inc.

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

	"github.com/cortexlabs/cortex/pkg/consts"
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
	msg := ""
	if originalError != nil {
		msg += urls.TrimQueryParamsStr(errors.Message(originalError)) + "\n\n"
	}

	if envName == "" {
		msg += fmt.Sprintf("unable to connect to your cluster (operator endpoint: %s)\n\n", operatorURL)
		msg += "if you don't have a cluster running:\n"
		msg += "    → to create a cluster, run `cortex cluster up`\n"
		msg += "\nif you have a cluster running:\n"
		msg += "    → run `cortex cluster info --configure-env ENV_NAME` to update your environment (replace ENV_NAME with your desired environment name, and include `--config <cluster.yaml>` if you have a cluster configuration file)\n"
	} else {
		msg += fmt.Sprintf("unable to connect to your cluster in the %s environment (operator endpoint: %s)\n\n", envName, operatorURL)
		msg += "if you don't have a cluster running:\n"
		msg += fmt.Sprintf("    → if you'd like to create a cluster, run `cortex cluster up --configure-env %s`\n", envName)
		msg += fmt.Sprintf("    → otherwise you can ignore this message, and prevent it in the future with `cortex env delete %s`\n", envName)
		msg += "\nif you have a cluster running:\n"
		msg += fmt.Sprintf("    → run `cortex cluster info --configure-env %s` to update your environment (include `--config <cluster.yaml>` if you have a cluster configuration file)\n", envName)
		msg += fmt.Sprintf("    → if you set `operator_load_balancer_scheme: internal` in your cluster configuration file, your CLI must run from within a VPC that has access to your cluster's VPC (see https://docs.cortex.dev/v/%s/)\n", consts.CortexVersionMinor)
	}

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
