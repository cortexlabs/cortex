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
	"strings"

	"github.com/cortexlabs/cortex/cli/types/cliconfig"
	"github.com/cortexlabs/cortex/cli/types/flags"
	"github.com/cortexlabs/cortex/pkg/lib/exit"
	libjson "github.com/cortexlabs/cortex/pkg/lib/json"
	"github.com/cortexlabs/cortex/pkg/lib/print"
	"github.com/cortexlabs/cortex/pkg/lib/telemetry"
	"github.com/spf13/cobra"
)

var (
	_flagEnvOperatorEndpoint string
)

func envInit() {
	_envConfigureCmd.Flags().SortFlags = false
	_envConfigureCmd.Flags().StringVarP(&_flagEnvOperatorEndpoint, "operator-endpoint", "o", "", "set the operator endpoint without prompting")
	_envCmd.AddCommand(_envConfigureCmd)

	_envListCmd.Flags().SortFlags = false
	_envListCmd.Flags().VarP(&_flagOutput, "output", "o", fmt.Sprintf("output format: one of %s", strings.Join(flags.UserOutputTypeStrings(), "|")))
	_envCmd.AddCommand(_envListCmd)

	_envDefaultCmd.Flags().SortFlags = false
	_envCmd.AddCommand(_envDefaultCmd)

	_envDeleteCmd.Flags().SortFlags = false
	_envCmd.AddCommand(_envDeleteCmd)
}

var _envCmd = &cobra.Command{
	Use:   "env",
	Short: "manage cli environments (contains subcommands)",
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

		fieldsToSkipPrompt := cliconfig.Environment{}
		if _flagEnvOperatorEndpoint != "" {
			operatorEndpoint, provider, err := validateOperatorEndpoint(_flagEnvOperatorEndpoint)
			if err != nil {
				exit.Error(err)
			}
			fieldsToSkipPrompt.OperatorEndpoint = operatorEndpoint
			fieldsToSkipPrompt.Provider = provider
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

		if _flagOutput == flags.JSONOutputType {
			if len(cliConfig.Environments) == 0 {
				fmt.Print("[]")
				return
			}
			bytes, err := libjson.Marshal(cliConfig.ConvertToUserFacingCLIConfig())
			if err != nil {
				exit.Error(err)
			}
			fmt.Print(string(bytes))
			return
		}

		if len(cliConfig.Environments) == 0 {
			fmt.Println("no environments are configured")
			return
		}

		defaultEnv, err := getDefaultEnv()
		if err != nil {
			exit.Error(err)
		}

		for i, env := range cliConfig.Environments {
			fmt.Print(env.String(defaultEnv != nil && *defaultEnv == env.Name))
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

		defaultEnv, err := getDefaultEnv()
		if err != nil {
			exit.Error(err)
		}

		var envName string
		if len(args) == 0 {
			if defaultEnv != nil {
				fmt.Printf("current default environment: %s\n\n", *defaultEnv)
			}
			envName = promptForExistingEnvName("name of environment to set as default")
		} else {
			envName = args[0]
		}

		if defaultEnv != nil && *defaultEnv == envName {
			print.BoldFirstLine(fmt.Sprintf("%s is already the default environment", envName))
			exit.Ok()
		}

		if err := setDefaultEnv(envName); err != nil {
			exit.Error(err)
		}

		print.BoldFirstLine(fmt.Sprintf("set %s as the default environment", envName))
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
			envName = promptForExistingEnvName("name of environment to delete")
		}

		prevDefault, err := getDefaultEnv()
		if err != nil {
			exit.Error(err)
		}

		if err := removeEnvFromCLIConfig(envName); err != nil {
			exit.Error(err)
		}

		newDefault, err := getDefaultEnv()
		if err != nil {
			exit.Error(err)
		}

		print.BoldFirstLine(fmt.Sprintf("deleted the %s environment configuration", envName))
		if prevDefault != nil && newDefault == nil {
			print.BoldFirstLine("unset the default environment")
		} else if newDefault != nil && (prevDefault == nil || *prevDefault != *newDefault) {
			print.BoldFirstLine(fmt.Sprintf("set the default environment to %s", *newDefault))
		}
	},
}
