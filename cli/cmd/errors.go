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
	"runtime"
	"strings"

	"github.com/cortexlabs/cortex/cli/types/cliconfig"
	"github.com/cortexlabs/cortex/pkg/lib/errors"
	s "github.com/cortexlabs/cortex/pkg/lib/strings"
	"github.com/cortexlabs/cortex/pkg/lib/urls"
	"github.com/cortexlabs/cortex/pkg/types"
	"github.com/cortexlabs/cortex/pkg/types/clusterstate"
)

const (
	_errStrCantMakeRequest = "unable to make request"
	_errStrRead            = "unable to read"
)

func errStrFailedToConnect(u url.URL) string {
	return "failed to connect to " + urls.TrimQueryParamsURL(u)
}

func getCloudFormationURL(clusterName, region string) string {
	return fmt.Sprintf("https://console.aws.amazon.com/cloudformation/home?region=%s#/stacks?filteringText=-%s-", region, clusterName)
}

const (
	ErrInvalidProvider                    = "cli.invalid_provider"
	ErrLocalProviderNotSupported          = "cli.local_provider_not_supported"
	ErrEnvironmentProviderNameConflict    = "cli.environment_provider_name_conflict"
	ErrOperatorEndpointInLocalEnvironment = "cli.operator_endpoint_in_local_environment"
	ErrOperatorConfigFromLocalEnvironment = "cli.operater_config_from_local_environment"
	ErrInvalidOperatorEndpoint            = "cli.invalid_operator_endpoint"
	ErrCortexYAMLNotFound                 = "cli.cortex_yaml_not_found"
	ErrConnectToDockerDaemon              = "cli.connect_to_docker_daemon"
	ErrDockerPermissions                  = "cli.docker_permissions"
	ErrDockerCtrlC                        = "cli.docker_ctrl_c"
	ErrResponseUnknown                    = "cli.response_unknown"
	ErrAPINotReady                        = "cli.api_not_ready"
	ErrOneAWSEnvVarSet                    = "cli.one_aws_env_var_set"
	ErrOneAWSConfigFieldSet               = "cli.one_aws_config_field_set"
	ErrClusterUp                          = "cli.cluster_up"
	ErrClusterUpdate                      = "cli.cluster_update"
	ErrClusterInfo                        = "cli.cluster_info"
	ErrClusterDebug                       = "cli.cluster_debug"
	ErrClusterRefresh                     = "cli.cluster_refresh"
	ErrClusterDown                        = "cli.cluster_down"
	ErrDuplicateCLIEnvNames               = "cli.duplicate_cli_env_names"
	ErrClusterUpInProgress                = "cli.cluster_up_in_progress"
	ErrClusterAlreadyCreated              = "cli.cluster_already_created"
	ErrClusterDownInProgress              = "cli.cluster_down_in_progress"
	ErrClusterAlreadyDeleted              = "cli.cluster_already_deleted"
	ErrFailedClusterStatus                = "cli.failed_cluster_status"
	ErrClusterDoesNotExist                = "cli.cluster_does_not_exist"
)

func ErrorInvalidProvider(providerStr string) error {
	return errors.WithStack(&errors.Error{
		Kind:    ErrInvalidProvider,
		Message: fmt.Sprintf("%s is not a valid provider (%s are supported)", providerStr, s.UserStrsAnd(types.ProviderTypeStrings())),
	})
}

func ErrorLocalProviderNotSupported(environment cliconfig.Environment) error {
	msg := "this command cannot run locally; please specify an existing environment (via --env=<name>) which points to an existing cluster, create/update an environment for an existing cluster (`cortex env configure`), or create a cortex cluster (`cortex cluster up`)"
	if environment.Name != types.LocalProviderType.String() {
		msg = fmt.Sprintf("the %s environment uses the local provider, but ", environment.Name) + msg
	}

	return errors.WithStack(&errors.Error{
		Kind:    ErrLocalProviderNotSupported,
		Message: msg,
	})
}

func ErrorEnvironmentProviderNameConflict(envName string, provider types.ProviderType) error {
	return errors.WithStack(&errors.Error{
		Kind:    ErrEnvironmentProviderNameConflict,
		Message: fmt.Sprintf("the %s environment cannot use the %s provider", envName, provider.String()),
	})
}

func ErrorOperatorConfigFromLocalEnvironment() error {
	return errors.WithStack(&errors.Error{
		Kind:    ErrOperatorConfigFromLocalEnvironment,
		Message: "attempting to retrieve cluster operator config from local environment",
	})
}

func ErrorOperatorEndpointInLocalEnvironment() error {
	return errors.WithStack(&errors.Error{
		Kind:    ErrOperatorEndpointInLocalEnvironment,
		Message: fmt.Sprintf("operator_endpoint should not be specified (it's not used in local providers)"),
	})
}

func ErrorInvalidOperatorEndpoint(endpoint string) error {
	return errors.WithStack(&errors.Error{
		Kind:    ErrInvalidOperatorEndpoint,
		Message: fmt.Sprintf("%s is not a cortex operator endpoint; run `cortex cluster info` to show your operator endpoint or run `cortex cluster up` to spin up a new cluster", endpoint),
	})
}

func ErrorCortexYAMLNotFound() error {
	return errors.WithStack(&errors.Error{
		Kind:    ErrCortexYAMLNotFound,
		Message: "no api config file was specified, and ./cortex.yaml does not exist; create cortex.yaml, or reference an existing config file by running `cortex deploy <config_file_path>`",
	})
}

func ErrorConnectToDockerDaemon() error {
	installMsg := "install it by following the instructions for your operating system: https://docs.docker.com/install"
	if strings.HasPrefix(runtime.GOOS, "darwin") {
		installMsg = "install it here: https://docs.docker.com/docker-for-mac/install"
	}

	return errors.WithStack(&errors.Error{
		Kind:    ErrConnectToDockerDaemon,
		Message: fmt.Sprintf("unable to connect to the Docker daemon\n\nplease confirm Docker is running, or if Docker is not installed, %s", installMsg),
	})
}

func ErrorDockerPermissions(err error) error {
	errStr := errors.Message(err)

	var groupAddStr string
	if strings.HasPrefix(runtime.GOOS, "linux") {
		groupAddStr = " (e.g. by running `sudo groupadd docker && sudo gpasswd -a $USER docker`)"
	}

	return errors.WithStack(&errors.Error{
		Kind:    ErrDockerPermissions,
		Message: errStr + "\n\nyou can re-run this command with `sudo`, or grant your current user access to docker" + groupAddStr,
	})
}

func ErrorDockerCtrlC() error {
	return errors.WithStack(&errors.Error{
		Kind:        ErrDockerCtrlC,
		NoPrint:     true,
		NoTelemetry: true,
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

func ErrorAPINotReady(apiName string, status string) error {
	return errors.WithStack(&errors.Error{
		Kind:    ErrAPINotReady,
		Message: fmt.Sprintf("%s is %s", apiName, status),
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

func ErrorClusterDoesNotExist(clusterName string, region string) error {
	return errors.WithStack(&errors.Error{
		Kind:    ErrClusterDoesNotExist,
		Message: fmt.Sprintf("there is no cluster named \"%s\" in %s", clusterName, region),
	})
}

func ErrorClusterUpInProgress(clusterName string, region string) error {
	return errors.WithStack(&errors.Error{
		Kind:    ErrClusterUpInProgress,
		Message: fmt.Sprintf("creation of cluster \"%s\" in %s is currently in progress", clusterName, region),
	})
}

func ErrorClusterAlreadyCreated(clusterName string, region string) error {
	return errors.WithStack(&errors.Error{
		Kind:    ErrClusterAlreadyCreated,
		Message: fmt.Sprintf("a cluster named \"%s\" already exists in %s", clusterName, region),
	})
}

func ErrorClusterDownInProgress(clusterName string, region string) error {
	return errors.WithStack(&errors.Error{
		Kind:    ErrClusterDownInProgress,
		Message: fmt.Sprintf("deletion of cluster \"%s\" in %s is currently in progress", clusterName, region),
	})
}

func ErrorClusterAlreadyDeleted(clusterName string, region string) error {
	return errors.WithStack(&errors.Error{
		Kind:    ErrClusterAlreadyDeleted,
		Message: fmt.Sprintf("cluster \"%s\" in %s has already been deleted (or does not exist)", clusterName, region),
	})
}

func ErrorFailedClusterStatus(status clusterstate.Status, clusterName string, region string) error {
	return errors.WithStack(&errors.Error{
		Kind:    ErrFailedClusterStatus,
		Message: fmt.Sprintf("cluster \"%s\" in %s encountered an unexpected status %s, please try to delete the cluster with `cortex cluster down` or delete the cloudformation stacks manually in your AWS console %s", clusterName, region, string(status), getCloudFormationURL(clusterName, region)),
	})
}
