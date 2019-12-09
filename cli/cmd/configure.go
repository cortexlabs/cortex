/*
Copyright 2019 Cortex Labs, Inc.

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

	"github.com/spf13/cobra"

	"github.com/cortexlabs/cortex/pkg/lib/exit"
	s "github.com/cortexlabs/cortex/pkg/lib/strings"
	"github.com/cortexlabs/cortex/pkg/lib/table"
	"github.com/cortexlabs/cortex/pkg/lib/telemetry"
)

var flagPrint bool

func init() {
	addEnvFlag(configureCmd)
	configureCmd.PersistentFlags().BoolVarP(&flagPrint, "print", "p", false, "print the configuration")
}

var configureCmd = &cobra.Command{
	Use:   "configure",
	Short: "configure the cli",
	Args:  cobra.NoArgs,
	Run: func(cmd *cobra.Command, args []string) {
		telemetry.Event("cli.configure")

		if flagPrint {
			cliEnvConfig, err := readCLIEnvConfig(flagEnv)
			if err != nil {
				exit.Error(err)
			}

			if cliEnvConfig == nil {
				if flagEnv == "default" {
					exit.Error("cli is not configured; run `cortex configure`")
				} else {
					exit.Error(fmt.Sprintf("cli is not configured; run `cortex configure --env=%s`", flagEnv))
				}
			}

			var items table.KeyValuePairs

			if flagEnv != "default" {
				items.Add("environment", flagEnv)
			}
			items.Add("cortex operator endpoint", cliEnvConfig.OperatorEndpoint)
			items.Add("aws access key id", cliEnvConfig.AWSAccessKeyID)
			items.Add("aws secret access key", s.MaskString(cliEnvConfig.AWSSecretAccessKey, 4))

			items.Print()
			return
		}

		configureCLIEnv(flagEnv)
	},
}
