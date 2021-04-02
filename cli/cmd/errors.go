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
	"strings"

	"github.com/cortexlabs/cortex/pkg/consts"
	"github.com/cortexlabs/cortex/pkg/lib/errors"
	s "github.com/cortexlabs/cortex/pkg/lib/strings"
	"github.com/cortexlabs/cortex/pkg/lib/urls"
	"github.com/cortexlabs/cortex/pkg/types/clusterconfig"
)

const (
	_errStrCantMakeRequest = "unable to make request"
	_errStrRead            = "unable to read"
)

func errStrFailedToConnect(u url.URL) string {
	return "failed to connect to " + urls.TrimQueryParamsURL(u)
}

const (
	ErrInvalidProvider                     = "cli.invalid_provider"
	ErrInvalidLegacyProvider               = "cli.invalid_legacy_provider"
	ErrNoAvailableEnvironment              = "cli.no_available_environment"
	ErrEnvironmentNotSet                   = "cli.environment_not_set"
	ErrEnvironmentNotFound                 = "cli.environment_not_found"
	ErrFieldNotFoundInEnvironment          = "cli.field_not_found_in_environment"
	ErrInvalidOperatorEndpoint             = "cli.invalid_operator_endpoint"
	ErrNoOperatorLoadBalancer              = "cli.no_operator_load_balancer"
	ErrCortexYAMLNotFound                  = "cli.cortex_yaml_not_found"
	ErrDockerCtrlC                         = "cli.docker_ctrl_c"
	ErrResponseUnknown                     = "cli.response_unknown"
	ErrOnlyAWSClusterFlagSet               = "cli.only_aws_cluster_flag_set"
	ErrMissingAWSCredentials               = "cli.missing_aws_credentials"
	ErrCredentialsInClusterConfig          = "cli.credentials_in_cluster_config"
	ErrClusterUp                           = "cli.cluster_up"
	ErrClusterScale                        = "cli.cluster_scale"
	ErrClusterInfo                         = "cli.cluster_info"
	ErrClusterDebug                        = "cli.cluster_debug"
	ErrClusterRefresh                      = "cli.cluster_refresh"
	ErrClusterDown                         = "cli.cluster_down"
	ErrSpecifyAtLeastOneFlag               = "cli.specify_at_least_one_flag"
	ErrMinInstancesLowerThan               = "cli.min_instances_lower_than"
	ErrMaxInstancesLowerThan               = "cli.max_instances_lower_than"
	ErrMinInstancesGreaterThanMaxInstances = "cli.min_instances_greater_than_max_instances"
	ErrNodeGroupNotFound                   = "cli.nodegroup_not_found"
	ErrDuplicateCLIEnvNames                = "cli.duplicate_cli_env_names"
	ErrClusterAccessConfigRequired         = "cli.cluster_access_config_or_prompts_required"
	ErrShellCompletionNotSupported         = "cli.shell_completion_not_supported"
	ErrNoTerminalWidth                     = "cli.no_terminal_width"
	ErrDeployFromTopLevelDir               = "cli.deploy_from_top_level_dir"
)

func ErrorInvalidProvider(providerStr, cliConfigPath string) error {
	return errors.WithStack(&errors.Error{
		Kind:    ErrInvalidProvider,
		Message: fmt.Sprintf("\"%s\" is not a supported provider (only aws is supported); remove the environment(s) which use the %s provider from %s, or delete %s (it will be recreated on subsequent CLI commands)", providerStr, providerStr, cliConfigPath, cliConfigPath),
	})
}

func ErrorInvalidLegacyProvider(providerStr, cliConfigPath string) error {
	return errors.WithStack(&errors.Error{
		Kind:    ErrInvalidLegacyProvider,
		Message: fmt.Sprintf("the %s provider is no longer supported on cortex v%s; remove the environment(s) which use the %s provider from %s, or delete %s (it will be recreated on subsequent CLI commands)", providerStr, consts.CortexVersionMinor, providerStr, cliConfigPath, cliConfigPath),
	})
}

func ErrorNoAvailableEnvironment() error {
	return errors.WithStack(&errors.Error{
		Kind:    ErrNoAvailableEnvironment,
		Message: "no environments are configured; run `cortex cluster up` to create a cluster, or run `cortex env configure` to connect to an existing cluster",
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

func ErrorClusterUp(out string) error {
	return errors.WithStack(&errors.Error{
		Kind:    ErrClusterUp,
		Message: out,
		NoPrint: true,
	})
}

func ErrorClusterScale(out string) error {
	return errors.WithStack(&errors.Error{
		Kind:    ErrClusterScale,
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

func ErrorSpecifyAtLeastOneFlag(flagsToSpecify ...string) error {
	return errors.WithStack(&errors.Error{
		Kind:    ErrSpecifyAtLeastOneFlag,
		Message: fmt.Sprintf("must specify at least one of the following flags: %s", s.StrsOr(flagsToSpecify)),
	})
}

func ErrorMinInstancesLowerThan(minValue int64) error {
	return errors.WithStack(&errors.Error{
		Kind:    ErrMinInstancesLowerThan,
		Message: fmt.Sprintf("min instances cannot be set to a value lower than %d", minValue),
	})
}

func ErrorMaxInstancesLowerThan(minValue int64) error {
	return errors.WithStack(&errors.Error{
		Kind:    ErrMaxInstancesLowerThan,
		Message: fmt.Sprintf("max instances cannot be set to a value lower than %d", minValue),
	})
}

func ErrorMinInstancesGreaterThanMaxInstances(minInstances, maxInstances int64) error {
	return errors.WithStack(&errors.Error{
		Kind:    ErrMinInstancesGreaterThanMaxInstances,
		Message: "min instances (%d) cannot be set to a value higher than max instances (%d)",
	})
}

func ErrorNodeGroupNotFound(scalingNodeGroupName, clusterName, clusterRegion string, availableNodeGroups []string) error {
	return errors.WithStack(&errors.Error{
		Kind:    ErrNodeGroupNotFound,
		Message: fmt.Sprintf("nodegroup %s couldn't be found in the cluster named %s in region %s; the available nodegroups for this cluster are %s", scalingNodeGroupName, clusterName, clusterRegion, s.StrsAnd(availableNodeGroups)),
	})
}

func ErrorClusterAccessConfigRequired(cliFlagsOnly bool) error {
	message := ""
	if cliFlagsOnly {
		message = "please provide the name and region of the cluster using the CLI flags (e.g. via `--name` and `--region`)"
	} else {
		message = fmt.Sprintf("please provide a cluster configuration file which specifies `%s` and `%s` (e.g. via `--config cluster.yaml`) or use the CLI flags to specify the cluster (e.g. via `--name` and `--region`)", clusterconfig.ClusterNameKey, clusterconfig.RegionKey)
	}
	return errors.WithStack(&errors.Error{
		Kind:    ErrClusterAccessConfigRequired,
		Message: message,
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
