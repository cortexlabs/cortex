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

	"github.com/cortexlabs/cortex/cli/types/cliconfig"
	"github.com/cortexlabs/cortex/pkg/lib/exit"
	"github.com/cortexlabs/cortex/pkg/lib/pointer"
	"github.com/cortexlabs/cortex/pkg/lib/print"
	"github.com/cortexlabs/cortex/pkg/lib/slices"
	s "github.com/cortexlabs/cortex/pkg/lib/strings"
	"github.com/cortexlabs/cortex/pkg/lib/table"
	"github.com/cortexlabs/cortex/pkg/lib/telemetry"
	"github.com/cortexlabs/cortex/pkg/types"
	"github.com/spf13/cobra"
)

var (
	_flagEnvProvider           string
	_flagEnvOperatorEndpoint   string
	_flagEnvAWSAccessKeyID     string
	_flagEnvAWSSecretAccessKey string
	_flagEnvAWSRegion          string
)

func envInit() {
	_envConfigureCmd.Flags().SortFlags = false
	_envConfigureCmd.Flags().StringVarP(&_flagEnvProvider, "provider", "p", "", "set the provider without prompting")
	_envConfigureCmd.Flags().StringVarP(&_flagEnvOperatorEndpoint, "operator-endpoint", "o", "", "set the operator endpoint without prompting")
	_envConfigureCmd.Flags().StringVarP(&_flagEnvAWSAccessKeyID, "aws-access-key-id", "k", "", "set the aws access key id without prompting")
	_envConfigureCmd.Flags().StringVarP(&_flagEnvAWSSecretAccessKey, "aws-secret-access-key", "s", "", "set the aws secret access key without prompting")
	_envConfigureCmd.Flags().StringVarP(&_flagEnvAWSRegion, "aws-region", "r", "", "set the aws region without prompting")
	_envCmd.AddCommand(_envConfigureCmd)

	_envListCmd.Flags().SortFlags = false
	_envCmd.AddCommand(_envListCmd)

	_envDefaultCmd.Flags().SortFlags = false
	_envCmd.AddCommand(_envDefaultCmd)

	_envDeleteCmd.Flags().SortFlags = false
	_envCmd.AddCommand(_envDeleteCmd)
}

var _envCmd = &cobra.Command{
	Use:   "env",
	Short: "manage environments",
}

var _envConfigureCmd = &cobra.Command{
	Use:   "configure [ENVIRONMENT_NAME]",
	Short: "configure an environment",
	Args:  cobra.RangeArgs(0, 1),
	Run: func(cmd *cobra.Command, args []string) {
		telemetry.Event("cli.env.configure")

		var envName string
		if len(args) == 1 {
			envName = args[0]
		}

		skipProvider := types.UnknownProviderType
		if _flagEnvProvider != "" {
			skipProvider = types.ProviderTypeFromString(_flagEnvProvider)
			if skipProvider == types.UnknownProviderType {
				exit.Error(ErrorInvalidProvider(_flagEnvProvider))
			}
		}

		var skipOperatorEndpoint *string
		if _flagEnvOperatorEndpoint != "" {
			skipOperatorEndpoint = &_flagEnvOperatorEndpoint
		}

		var skipAWSAccessKeyID *string
		if _flagEnvAWSAccessKeyID != "" {
			skipAWSAccessKeyID = &_flagEnvAWSAccessKeyID
		}

		var skipAWSSecretAccessKey *string
		if _flagEnvAWSSecretAccessKey != "" {
			skipAWSSecretAccessKey = &_flagEnvAWSSecretAccessKey
		}

		var skipAWSRegion *string
		if _flagEnvAWSRegion != "" {
			skipAWSRegion = &_flagEnvAWSRegion
		}

		fieldsToSkipPrompt := cliconfig.Environment{
			Provider:           skipProvider,
			OperatorEndpoint:   skipOperatorEndpoint,
			AWSAccessKeyID:     skipAWSAccessKeyID,
			AWSSecretAccessKey: skipAWSSecretAccessKey,
			AWSRegion:          skipAWSRegion,
		}

		if _, err := configureEnv(envName, fieldsToSkipPrompt); err != nil {
			exit.Error(err)
		}
	},
}

var _envListCmd = &cobra.Command{
	Use:   "list",
	Short: "list all configured environments",
	Args:  cobra.NoArgs,
	Run: func(cmd *cobra.Command, args []string) {
		telemetry.Event("cli.env.list")

		cliConfig, err := readCLIConfig()
		if err != nil {
			exit.Error(err)
		}

		defaultEnv := getDefaultEnv(_generalCommandType)

		for i, env := range cliConfig.Environments {
			var items table.KeyValuePairs

			if env.Name == defaultEnv {
				items.Add("name", env.Name+" (default)")
			} else {
				items.Add("name", env.Name)
			}

			items.Add("provider", env.Provider)

			if env.OperatorEndpoint != nil {
				items.Add("cortex operator endpoint", *env.OperatorEndpoint)
			}
			if env.AWSAccessKeyID != nil {
				items.Add("aws access key id", *env.AWSAccessKeyID)
			}
			if env.AWSSecretAccessKey != nil {
				items.Add("aws secret access key", s.MaskString(*env.AWSSecretAccessKey, 4))
			}
			if env.AWSRegion != nil {
				items.Add("aws region", *env.AWSRegion)
			}

			items.Print(&table.KeyValuePairOpts{
				BoldFirstLine: pointer.Bool(true),
			})
			if i+1 < len(cliConfig.Environments) {
				fmt.Println()
			}
		}
	},
}

var _envDefaultCmd = &cobra.Command{
	Use:   "default [ENVIRONMENT_NAME]",
	Short: "set the default environment",
	Args:  cobra.RangeArgs(0, 1),
	Run: func(cmd *cobra.Command, args []string) {
		telemetry.Event("cli.env.default")

		defaultEnv := getDefaultEnv(_generalCommandType)

		var envName string
		if len(args) == 1 {
			envName = args[0]

			configuredEnvNames, err := listConfiguredEnvNames()
			if err != nil {
				exit.Error(err)
			}
			if !slices.HasString(configuredEnvNames, envName) {
				exit.Error(cliconfig.ErrorEnvironmentNotConfigured(envName))
			}

		} else {
			fmt.Printf("current default environment: %s\n\n", defaultEnv)
			envName = promptEnvName("name of environment to set as default", true)
		}

		if envName == defaultEnv {
			print.BoldFirstLine(fmt.Sprintf("✓ %s is already the default environment", envName))
			exit.Ok()
		}

		if err := setDefaultEnv(envName); err != nil {
			exit.Error(err)
		}

		print.BoldFirstLine(fmt.Sprintf("✓ set %s as the default environment", envName)) // TODO we don't dispaly a check mark for other successful commands, I think we should delete this for consistency
	},
}

var _envDeleteCmd = &cobra.Command{
	Use:   "delete [ENVIRONMENT_NAME]",
	Short: "delete an environment configuration",
	Args:  cobra.RangeArgs(0, 1),
	Run: func(cmd *cobra.Command, args []string) {
		telemetry.Event("cli.env.delete")

		var envName string
		if len(args) == 1 {
			envName = args[0]
		} else {
			envName = promptEnvName("name of environment to delete", true)
		}

		if err := removeEnvFromCLIConfig(envName); err != nil {
			exit.Error(err)
		}

		if envName == types.LocalProviderType.String() {
			print.BoldFirstLine(fmt.Sprintf("✓ cleared %s environment configuration", envName))
		} else {
			print.BoldFirstLine(fmt.Sprintf("✓ deleted %s environment configuration", envName))
		}
	},
}
