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

package cmd

import (
	"fmt"
	"net/url"

	"github.com/cortexlabs/cortex/pkg/lib/errors"
	s "github.com/cortexlabs/cortex/pkg/lib/strings"
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
	ErrCLINotConfigured              = "cli.cli_not_configured"
	ErrCortexYAMLNotFound            = "cli.cortex_yaml_not_found"
	ErrDockerDaemon                  = "cli.docker_daemon"
	ErrDockerCtrlC                   = "cli.docker_ctrl_c"
	ErrAPINotReady                   = "cli.api_not_ready"
	ErrFailedToConnectOperator       = "cli.failed_to_connect_operator"
	ErrOperatorSocketRead            = "cli.operator_socket_read"
	ErrResponseUnknown               = "cli.response_unknown"
	ErrOperatorResponseUnknown       = "cli.operator_response_unknown"
	ErrOperatorStreamResponseUnknown = "cli.operator_stream_response_unknown"
	ErrOneAWSEnvVarSet               = "cli.one_aws_env_var_set"
	ErrOneAWSConfigFieldSet          = "cli.one_aws_config_field_set"
	ErrClusterUp                     = "cli.cluster_up"
	ErrClusterUpdate                 = "cli.cluster_update"
	ErrClusterInfo                   = "cli.cluster_info"
	ErrClusterDebug                  = "cli.cluster_debug"
	ErrClusterRefresh                = "cli.cluster_refresh"
	ErrClusterDown                   = "cli.cluster_down"
	ErrDuplicateCLIEnvNames          = "cli.duplicate_cli_env_names"
)

func ErrorCLINotConfigured(env string) error {
	msg := "your cli is not configured; run `cortex configure`"
	if env != "default" {
		msg = fmt.Sprintf("your cli is not configured; run `cortex configure --env=%s`", env)
	}

	return errors.WithStack(&errors.Error{
		Kind:    ErrCLINotConfigured,
		Message: msg,
	})
}

func ErrorCortexYAMLNotFound() error {
	return errors.WithStack(&errors.Error{
		Kind:    ErrCortexYAMLNotFound,
		Message: "no api config file was specified, and ./cortex.yaml does not exist; create cortex.yaml, or reference an existing config file by running `cortex deploy <config_file_path>`",
	})
}

func ErrorDockerDaemon() error {
	return errors.WithStack(&errors.Error{
		Kind:    ErrDockerDaemon,
		Message: "unable to connect to the Docker daemon, please confirm Docker is running",
	})
}

func ErrorDockerCtrlC() error {
	return errors.WithStack(&errors.Error{
		Kind:        ErrDockerCtrlC,
		NoPrint:     true,
		NoTelemetry: true,
	})
}

func ErrorAPINotReady(apiName string, status string) error {
	return errors.WithStack(&errors.Error{
		Kind:    ErrAPINotReady,
		Message: fmt.Sprintf("%s is %s", s.UserStr(apiName), status),
	})
}

func ErrorFailedToConnectOperator(originalError error, operatorURL string) error {
	operatorURLMsg := ""
	if operatorURL != "" {
		operatorURLMsg = fmt.Sprintf(" (%s)", operatorURL)
	}

	originalErrMsg := ""
	if originalError != nil {
		originalErrMsg = urls.TrimQueryParamsStr(errors.Message(originalError)) + "\n\n"
	}

	return errors.WithStack(&errors.Error{
		Kind:    ErrFailedToConnectOperator,
		Message: fmt.Sprintf("%sfailed to connect to the operator%s, run `cortex configure` if you need to update the operator endpoint, run `cortex cluster info` to show your operator endpoint", originalErrMsg, operatorURLMsg),
	})
}

func ErrorOperatorSocketRead(err error) error {
	return errors.WithStack(&errors.Error{
		Kind:    ErrOperatorSocketRead,
		Message: err.Error(),
		NoPrint: true,
	})
}

func ErrorResponseUnknown(body string) error {
	return errors.WithStack(&errors.Error{
		Kind:    ErrResponseUnknown,
		Message: body,
	})
}

func ErrorOperatorResponseUnknown(body string) error {
	return errors.WithStack(&errors.Error{
		Kind:    ErrOperatorResponseUnknown,
		Message: body,
	})
}

func ErrorOperatorStreamResponseUnknown(body string) error {
	return errors.WithStack(&errors.Error{
		Kind:    ErrOperatorStreamResponseUnknown,
		Message: body,
	})
}

func ErrorOneAWSEnvVarSet(setEnvVar string, missingEnvVar string) error {
	return errors.WithStack(&errors.Error{
		Kind:    ErrOneAWSEnvVarSet,
		Message: fmt.Sprintf("only $%s is set; please run `export %s=***`", setEnvVar, missingEnvVar),
	})
}

func ErrorOneAWSConfigFieldSet(setConfigField string, missingConfigField string, configPath string) error {
	return errors.WithStack(&errors.Error{
		Kind:    ErrOneAWSConfigFieldSet,
		Message: fmt.Sprintf("only %s is set in %s; please set %s as well", setConfigField, _flagClusterConfig, missingConfigField),
	})
}

func ErrorClusterUp(out string) error {
	return errors.WithStack(&errors.Error{
		Kind:    ErrClusterUp,
		Message: out,
		NoPrint: true,
	})
}

func ErrorClusterUpdate(out string) error {
	return errors.WithStack(&errors.Error{
		Kind:    ErrClusterUpdate,
		Message: out,
		NoPrint: true,
	})
}

func ErrorClusterInfo(out string) error {
	return errors.WithStack(&errors.Error{
		Kind:    ErrClusterInfo,
		Message: out,
		NoPrint: true,
	})
}

func ErrorClusterDebug(out string) error {
	return errors.WithStack(&errors.Error{
		Kind:    ErrClusterDebug,
		Message: out,
		NoPrint: true,
	})
}

func ErrorClusterRefresh(out string) error {
	return errors.WithStack(&errors.Error{
		Kind:    ErrClusterRefresh,
		Message: out,
		NoPrint: true,
	})
}

func ErrorClusterDown(out string) error {
	return errors.WithStack(&errors.Error{
		Kind:    ErrClusterDown,
		Message: out,
		NoPrint: true,
	})
}

func ErrorDuplicateCLIEnvNames(environment string) error {
	return errors.WithStack(&errors.Error{
		Kind:    ErrDuplicateCLIEnvNames,
		Message: fmt.Sprintf("duplicate environment names: %s is defined more than once", s.UserStr(environment)),
	})
}
