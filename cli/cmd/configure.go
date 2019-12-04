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

	s "github.com/cortexlabs/cortex/pkg/lib/strings"
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
		telemetry.ReportEvent("cli.configure", nil)

		if flagPrint {
			cliEnvConfig, err := readCLIEnvConfig(flagEnv)
			if err != nil {
				telemetry.ExitErr(err)
			}

			if cliEnvConfig == nil {
				if flagEnv == "default" {
					telemetry.ExitErr("cli is not configured; run `cortex configure`")
				} else {
					telemetry.ExitErr(fmt.Sprintf("cli is not configured; run `cortex configure --env=%s`", flagEnv))
				}
			}

			if flagEnv != "default" {
				fmt.Printf("environment:              %s\n", flagEnv)
			}
			fmt.Printf("cortex operator endpoint: %s\n", cliEnvConfig.OperatorEndpoint)
			fmt.Printf("aws access key id:        %s\n", cliEnvConfig.AWSAccessKeyID)
			fmt.Printf("aws secret access key:    %s\n", s.MaskString(cliEnvConfig.AWSSecretAccessKey, 4))
			return
		}

		configureCLIEnv(flagEnv)
	},
}
