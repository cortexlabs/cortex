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
	"strings"

	"github.com/cortexlabs/cortex/cli/cluster"
	"github.com/cortexlabs/cortex/cli/local"
	"github.com/cortexlabs/cortex/pkg/lib/console"
	"github.com/cortexlabs/cortex/pkg/lib/errors"
	"github.com/cortexlabs/cortex/pkg/lib/exit"
	"github.com/cortexlabs/cortex/pkg/lib/telemetry"
	"github.com/cortexlabs/cortex/pkg/types"
	"github.com/spf13/cobra"
)

var _flagLogsEnv string

func logsInit() {
	_logsCmd.Flags().SortFlags = false
	_logsCmd.Flags().StringVarP(&_flagLogsEnv, "env", "e", getDefaultEnv(_generalCommandType), "environment to use")
}

var _logsCmd = &cobra.Command{
	Use:   "logs API_NAME",
	Short: "stream logs from an api",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		telemetry.Event("cli.logs")
		printEnvIfNotSpecified(_flagLogsEnv)

		apiName := args[0]
		env := MustReadOrConfigureEnv(_flagLogsEnv)
		if env.Provider == types.AWSProviderType {
			err := cluster.StreamLogs(MustGetOperatorConfig(env.Name), apiName)
			if err != nil {
				// note: if modifying this string, search the codebase for it and change all occurrences
				if strings.HasSuffix(errors.Message(err), "is not deployed") {
					fmt.Println(console.Bold(errors.Message(err)))
					return
				}
				exit.Error(err)
			}
		} else {
			err := local.StreamLogs(apiName)
			if err != nil {
				exit.Error(err)
			}
		}

	},
}
