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
	"github.com/cortexlabs/cortex/pkg/types"
	"github.com/spf13/cobra"
)

var _flagProvider string
var _flagOperatorEndpoint string
var _flagAWSAccessKeyID string
var _flagAWSSecretAccessKey string

func init() {
	_envConfigureCmd.Flags().StringVarP(&_flagProvider, "provider", "v", "", "set the provider without prompting")
	_envConfigureCmd.Flags().StringVarP(&_flagOperatorEndpoint, "operator-endpoint", "o", "", "set the operator endpoint without prompting")
	_envConfigureCmd.Flags().StringVarP(&_flagAWSAccessKeyID, "aws-access-key-id", "k", "", "set the aws access key id without prompting")
	_envConfigureCmd.Flags().StringVarP(&_flagAWSSecretAccessKey, "aws-secret-access-key", "s", "", "set the aws secret access key without prompting")
	_envCmd.AddCommand(_envConfigureCmd)

	_envCmd.AddCommand(_envListCmd)

	_envCmd.AddCommand(_envRemoveCmd)
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
		telemetry.Event("cli.configure")

		var envName string
		if len(args) == 1 {
			envName = args[0]
		}

		skipProvider := types.UnknownProviderType
		if _flagProvider != "" {
			skipProvider = types.ProviderTypeFromString(_flagProvider)
			if skipProvider == types.UnknownProviderType {
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

		configureEnv(envName, fieldsToSkipPrompt)
	},
}

var _envListCmd = &cobra.Command{
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
			items.Add("name", env.Name)
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

var _envRemoveCmd = &cobra.Command{
	Use:   "remove ENVIRONMENT_NAME",
	Short: "remove an environment configuration",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		telemetry.Event("cli.configure.remove")

		envName := args[0]

		if err := removeEnvFromCLIConfig(envName); err != nil {
			exit.Error(err)
		}

		if envName == types.LocalProviderType.String() {
			fmt.Printf("✓ cleared %s environment\n", envName)
		} else {
			fmt.Printf("✓ removed %s environment\n", envName)
		}
	},
}
