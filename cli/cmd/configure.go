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
	s "github.com/cortexlabs/cortex/pkg/lib/strings"
	"github.com/cortexlabs/cortex/pkg/lib/table"
	"github.com/cortexlabs/cortex/pkg/lib/telemetry"
	"github.com/spf13/cobra"
)

var _flagPrint bool
var _flagOperatorEndpoint string
var _flagAWSAccessKeyID string
var _flagAWSSecretAccessKey string

func init() {
	addEnvFlag(_configureCmd)
	_configureCmd.PersistentFlags().BoolVarP(&_flagPrint, "print", "p", false, "print the configuration")
	_configureCmd.PersistentFlags().StringVarP(&_flagOperatorEndpoint, "operator-endpoint", "o", "", "set the operator endpoint without prompting")
	_configureCmd.PersistentFlags().StringVarP(&_flagAWSAccessKeyID, "aws-access-key-id", "k", "", "set the aws access key id without prompting")
	_configureCmd.PersistentFlags().StringVarP(&_flagAWSSecretAccessKey, "aws-secret-access-key", "s", "", "set the aws secret access key without prompting")
}

var _configureCmd = &cobra.Command{
	Use:   "configure",
	Short: "configure the cli",
	Args:  cobra.NoArgs,
	Run: func(cmd *cobra.Command, args []string) {
		telemetry.Event("cli.configure")

		if _flagPrint {
			printConfiguration()
			return
		}

		fieldsToSkipPrompt := CLIEnvConfig{
			OperatorEndpoint:   _flagOperatorEndpoint,
			AWSAccessKeyID:     _flagAWSAccessKeyID,
			AWSSecretAccessKey: _flagAWSSecretAccessKey,
		}

		configureCLIEnv(_flagEnv, fieldsToSkipPrompt)
	},
}

func printConfiguration() {
	cliEnvConfig, err := readCLIEnvConfig(_flagEnv)
	if err != nil {
		exit.Error(err)
	}

	if cliEnvConfig == nil {
		if _flagEnv == "default" {
			exit.Error("cli is not configured; run `cortex configure`")
		} else {
			exit.Error(fmt.Sprintf("cli is not configured; run `cortex configure --env=%s`", _flagEnv))
		}
	}

	var items table.KeyValuePairs

	if _flagEnv != "default" {
		items.Add("environment", _flagEnv)
	}
	items.Add("cortex operator endpoint", cliEnvConfig.OperatorEndpoint)
	items.Add("aws access key id", cliEnvConfig.AWSAccessKeyID)
	items.Add("aws secret access key", s.MaskString(cliEnvConfig.AWSSecretAccessKey, 4))

	items.Print()
}
