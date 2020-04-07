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

	"github.com/cortexlabs/cortex/pkg/lib/exit"
	"github.com/cortexlabs/cortex/pkg/lib/pointer"
	s "github.com/cortexlabs/cortex/pkg/lib/strings"
	"github.com/cortexlabs/cortex/pkg/lib/table"
	"github.com/cortexlabs/cortex/pkg/lib/telemetry"
	"github.com/spf13/cobra"
)

var _flagProvider string
var _flagOperatorEndpoint string
var _flagAWSAccessKeyID string
var _flagAWSSecretAccessKey string

func init() {
	addEnvFlag(_configureCmd, Local.String())
	_configureCmd.Flags().StringVarP(&_flagProvider, "provider", "v", "", "set the provider without prompting")
	_configureCmd.Flags().StringVarP(&_flagOperatorEndpoint, "operator-endpoint", "o", "", "set the operator endpoint without prompting")
	_configureCmd.Flags().StringVarP(&_flagAWSAccessKeyID, "aws-access-key-id", "k", "", "set the aws access key id without prompting")
	_configureCmd.Flags().StringVarP(&_flagAWSSecretAccessKey, "aws-secret-access-key", "s", "", "set the aws secret access key without prompting")

	_configureCmd.AddCommand(_configureListCmd)
	_configureCmd.AddCommand(_configureRemoveCmd)

}

var _configureCmd = &cobra.Command{
	Use:   "configure [--env=ENVIRONMENT_NAME]",
	Short: "configure an environment",
	Args:  cobra.NoArgs,
	Run: func(cmd *cobra.Command, args []string) {
		telemetry.Event("cli.configure")

		skipProvider := UnknownProvider
		if _flagProvider != "" {
			skipProvider = ProviderFromString(_flagProvider)
			if skipProvider == UnknownProvider {
				exit.Error(ErrorInvalidProvider(_flagProvider))
			}
		}

		var skipOperatorEndpoint *string
		if _flagOperatorEndpoint != "" {
			skipOperatorEndpoint = &_flagOperatorEndpoint
		}

		var skipAWSAccessKeyID *string
		if _flagAWSAccessKeyID != "" {
			skipAWSAccessKeyID = &_flagAWSAccessKeyID
		}

		var skipAWSSecretAccessKey *string
		if _flagAWSSecretAccessKey != "" {
			skipAWSSecretAccessKey = &_flagAWSSecretAccessKey
		}

		fieldsToSkipPrompt := Environment{
			Provider:           skipProvider,
			OperatorEndpoint:   skipOperatorEndpoint,
			AWSAccessKeyID:     skipAWSAccessKeyID,
			AWSSecretAccessKey: skipAWSSecretAccessKey,
		}

		configureEnv(_flagEnv, fieldsToSkipPrompt)
	},
}

var _configureListCmd = &cobra.Command{
	Use:   "list",
	Short: "list all configured environments",
	Args:  cobra.NoArgs,
	Run: func(cmd *cobra.Command, args []string) {
		telemetry.Event("cli.configure.list")

		cliConfig, err := readCLIConfig()
		if err != nil {
			exit.Error(err)
		}

		for i, env := range cliConfig.Environments {
			var items table.KeyValuePairs
			items.Add("environment name", env.Name)
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

			items.Print(&table.KeyValuePairOpts{
				BoldFirstLine: pointer.Bool(true),
			})
			if i+1 < len(cliConfig.Environments) {
				fmt.Println()
			}
		}
	},
}

var _configureRemoveCmd = &cobra.Command{
	Use:   "remove ENVIRONMENT_NAME",
	Short: "remove a configured environment",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		telemetry.Event("cli.configure.remove")

		envName := args[0]

		if err := removeEnvFromCLIConfig(envName); err != nil {
			exit.Error(err)
		}

		if envName == Local.String() {
			fmt.Printf("✓ cleared %s environment\n", envName)
		} else {
			fmt.Printf("✓ removed %s environment\n", envName)
		}
	},
}
