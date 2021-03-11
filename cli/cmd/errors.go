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

package cmd

import (
	"fmt"
	"net/url"
	"runtime"
	"strings"

	"github.com/cortexlabs/cortex/pkg/consts"
	"github.com/cortexlabs/cortex/pkg/lib/errors"
	s "github.com/cortexlabs/cortex/pkg/lib/strings"
	"github.com/cortexlabs/cortex/pkg/lib/urls"
	"github.com/cortexlabs/cortex/pkg/types"
	"github.com/cortexlabs/cortex/pkg/types/clusterconfig"
	"github.com/cortexlabs/cortex/pkg/types/userconfig"
)

const (
	_errStrCantMakeRequest = "unable to make request"
	_errStrRead            = "unable to read"
)

func errStrFailedToConnect(u url.URL) string {
	return "failed to connect to " + urls.TrimQueryParamsURL(u)
}

const (
	ErrInvalidProvider                         = "cli.invalid_provider"
	ErrInvalidLegacyProvider                   = "cli.invalid_legacy_provider"
	ErrCommandNotSupportedForKind              = "cli.command_not_supported_for_kind"
	ErrNoAvailableEnvironment                  = "cli.no_available_environment"
	ErrEnvironmentNotSet                       = "cli.environment_not_set"
	ErrEnvironmentNotFound                     = "cli.environment_not_found"
	ErrFieldNotFoundInEnvironment              = "cli.field_not_found_in_environment"
	ErrInvalidOperatorEndpoint                 = "cli.invalid_operator_endpoint"
	ErrNoOperatorLoadBalancer                  = "cli.no_operator_load_balancer"
	ErrCortexYAMLNotFound                      = "cli.cortex_yaml_not_found"
	ErrConnectToDockerDaemon                   = "cli.connect_to_docker_daemon"
	ErrDockerPermissions                       = "cli.docker_permissions"
	ErrDockerCtrlC                             = "cli.docker_ctrl_c"
	ErrResponseUnknown                         = "cli.response_unknown"
	ErrAPINotReady                             = "cli.api_not_ready"
	ErrOneAWSEnvVarSet                         = "cli.one_aws_env_var_set"
	ErrOneAWSFlagSet                           = "cli.one_aws_flag_set"
	ErrOnlyAWSClusterEnvVarSet                 = "cli.only_aws_cluster_env_var_set"
	ErrOnlyAWSClusterFlagSet                   = "cli.only_aws_cluster_flag_set"
	ErrMissingAWSCredentials                   = "cli.missing_aws_credentials"
	ErrCredentialsInClusterConfig              = "cli.credentials_in_cluster_config"
	ErrClusterUp                               = "cli.cluster_up"
	ErrClusterConfigure                        = "cli.cluster_configure"
	ErrClusterInfo                             = "cli.cluster_info"
	ErrClusterDebug                            = "cli.cluster_debug"
	ErrClusterRefresh                          = "cli.cluster_refresh"
	ErrClusterDown                             = "cli.cluster_down"
	ErrDuplicateCLIEnvNames                    = "cli.duplicate_cli_env_names"
	ErrClusterAccessConfigRequired             = "cli.cluster_access_config_or_prompts_required"
	ErrGCPClusterAccessConfigRequired          = "cli.gcp_cluster_access_config_or_prompts_required"
	ErrGCPClusterAccessConfigOrPromptsRequired = "cli.gcp_cluster_access_config_or_prompts_required"
	ErrShellCompletionNotSupported             = "cli.shell_completion_not_supported"
	ErrNoTerminalWidth                         = "cli.no_terminal_width"
	ErrDeployFromTopLevelDir                   = "cli.deploy_from_top_level_dir"
	ErrGCPClusterAlreadyExists                 = "cli.gcp_cluster_already_exists"
	ErrGCPClusterDoesntExist                   = "cli.gcp_cluster_doesnt_exist"
)

func ErrorInvalidProvider(providerStr string) error {
	return errors.WithStack(&errors.Error{
		Kind:    ErrInvalidProvider,
		Message: fmt.Sprintf("%s is not a valid provider (%s are supported)", providerStr, s.UserStrsAnd(types.ProviderTypeStrings())),
	})
}

func ErrorInvalidLegacyProvider(providerStr, cliConfigPath string) error {
	return errors.WithStack(&errors.Error{
		Kind:    ErrInvalidLegacyProvider,
		Message: fmt.Sprintf("the %s provider is no longer supported on cortex v%s; remove the environment(s) which use the %s provider from %s, or delete %s (it will be recreated on subsequent CLI commands)", providerStr, consts.CortexVersionMinor, providerStr, cliConfigPath, cliConfigPath),
	})
}

func ErrorCommandNotSupportedForKind(kind userconfig.Kind, command string) error {
	return errors.WithStack(&errors.Error{
		Kind:    ErrCommandNotSupportedForKind,
		Message: fmt.Sprintf("the `%s` command is not supported for %s kind", command, kind),
	})
}

func ErrorNoAvailableEnvironment() error {
	return errors.WithStack(&errors.Error{
		Kind:    ErrNoAvailableEnvironment,
		Message: "no environments are configured; run `cortex cluster up` or `cortex cluster-gcp up` to create a cluster, or run `cortex env configure` to connect to an existing cluster",
	})
}

func ErrorEnvironmentNotSet() error {
	return errors.WithStack(&errors.Error{
		Kind:    ErrEnvironmentNotSet,
		Message: "no environment was provided and the default environment is not set; specify the environment to use via the `-e/--env` flag, or run `cortex env default` to set the default environment",
	})
}

func ErrorEnvironmentNotFound(envName string) error {
	return errors.WithStack(&errors.Error{
		Kind:    ErrEnvironmentNotFound,
		Message: fmt.Sprintf("unable to find environment named \"%s\"", envName),
	})
}

func ErrorFieldNotFoundInEnvironment(fieldName string, envName string) error {
	return errors.WithStack(&errors.Error{
		Kind:    ErrFieldNotFoundInEnvironment,
		Message: fmt.Sprintf("%s was not found in %s environment", fieldName, envName),
	})
}

func ErrorInvalidOperatorEndpoint(endpoint string) error {
	return errors.WithStack(&errors.Error{
		Kind:    ErrInvalidOperatorEndpoint,
		Message: fmt.Sprintf("%s is not a cortex operator endpoint; run `cortex cluster info` to show your operator endpoint or run `cortex cluster up` to spin up a new cluster", endpoint),
	})
}

func ErrorNoOperatorLoadBalancer() error {
	return errors.WithStack(&errors.Error{
		Kind:    ErrNoOperatorLoadBalancer,
		Message: "unable to locate operator load balancer",
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

func ErrorOneAWSFlagSet(setFlag string, missingFlag string) error {
	return errors.WithStack(&errors.Error{
		Kind:    ErrOneAWSFlagSet,
		Message: fmt.Sprintf("only flag %s was provided; please provide %s as well", setFlag, missingFlag),
	})
}

func ErrorOnlyAWSClusterEnvVarSet() error {
	return errors.WithStack(&errors.Error{
		Kind:    ErrOnlyAWSClusterEnvVarSet,
		Message: "when specifying $CLUSTER_AWS_ACCESS_KEY_ID and $CLUSTER_AWS_SECRET_ACCESS_KEY, please also specify $AWS_ACCESS_KEY_ID and $AWS_SECRET_ACCESS_KEY",
	})
}

func ErrorClusterUp(out string) error {
	return errors.WithStack(&errors.Error{
		Kind:    ErrClusterUp,
		Message: out,
		NoPrint: true,
	})
}

func ErrorClusterConfigure(out string) error {
	return errors.WithStack(&errors.Error{
		Kind:    ErrClusterConfigure,
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

func ErrorClusterAccessConfigRequired() error {
	return errors.WithStack(&errors.Error{
		Kind:    ErrClusterAccessConfigRequired,
		Message: fmt.Sprintf("please provide a cluster configuration file which specifies `%s` and `%s` (e.g. via `--config cluster.yaml`) or use the CLI flags to specify the cluster (e.g. via `--name` and `--region`)", clusterconfig.ClusterNameKey, clusterconfig.RegionKey),
	})
}

func ErrorGCPClusterAccessConfigRequired() error {
	return errors.WithStack(&errors.Error{
		Kind:    ErrGCPClusterAccessConfigRequired,
		Message: fmt.Sprintf("please provide a cluster configuration file which specifies `%s` and `%s` (e.g. via `--config cluster.yaml`) or use the CLI flags to specify the cluster (e.g. via `--name`, `--project` and `--zone`)", clusterconfig.ClusterNameKey, clusterconfig.RegionKey),
	})
}

func ErrorShellCompletionNotSupported(shell string) error {
	return errors.WithStack(&errors.Error{
		Kind:    ErrShellCompletionNotSupported,
		Message: fmt.Sprintf("shell completion for %s is not supported (only bash and zsh are supported)", shell),
	})
}

func ErrorNoTerminalWidth() error {
	return errors.WithStack(&errors.Error{
		Kind:    ErrNoTerminalWidth,
		Message: "unable to determine terminal width; please re-run the command without the `--watch` flag",
	})
}

func ErrorDeployFromTopLevelDir(genericDirName string) error {
	return errors.WithStack(&errors.Error{
		Kind:    ErrDeployFromTopLevelDir,
		Message: fmt.Sprintf("cannot deploy from your %s directory - when deploying your API, cortex sends all files in your project directory (i.e. the directory which contains cortex.yaml) to your cluster (see https://docs.cortex.dev/v/%s/); therefore it is recommended to create a subdirectory for your project files", genericDirName, consts.CortexVersionMinor),
	})
}

func ErrorGCPClusterAlreadyExists(clusterName string, zone string, project string) error {
	return errors.WithStack(&errors.Error{
		Kind:    ErrGCPClusterAlreadyExists,
		Message: fmt.Sprintf("there is already a cluster named \"%s\" in %s in the %s project", clusterName, zone, project),
	})
}

func ErrorGCPClusterDoesntExist(clusterName string, zone string, project string) error {
	return errors.WithStack(&errors.Error{
		Kind:    ErrGCPClusterDoesntExist,
		Message: fmt.Sprintf("there is no cluster named \"%s\" in %s in the %s project", clusterName, zone, project),
	})
}
