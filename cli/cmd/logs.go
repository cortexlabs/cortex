/*
Copyright 2022 Cortex Labs, Inc.

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

	"github.com/cortexlabs/cortex/cli/cluster"
	"github.com/cortexlabs/cortex/pkg/lib/exit"
	"github.com/cortexlabs/cortex/pkg/lib/telemetry"
	"github.com/spf13/cobra"
)

var (
	_flagLogsEnv            string
	_flagLogsDisallowPrompt bool
	_flagRandomPod          bool
	_logsOutput             = `Navigate to the link below and click "Run Query":

%s

NOTE: there may be 1-2 minutes of delay for the logs to show up in the results of CloudWatch Insight queries
`
)

func logsInit() {
	_logsCmd.Flags().SortFlags = false
	_logsCmd.Flags().StringVarP(&_flagLogsEnv, "env", "e", "", "environment to use")
	_logsCmd.Flags().BoolVarP(&_flagLogsDisallowPrompt, "yes", "y", false, "skip prompts")
	_logsCmd.Flags().BoolVarP(&_flagRandomPod, "random-pod", "", false, "stream logs from a random pod")
}

var _logsCmd = &cobra.Command{
	Use:   "logs API_NAME [JOB_ID]",
	Short: "get the logs for a workload",
	Args:  cobra.RangeArgs(1, 2),
	Run: func(cmd *cobra.Command, args []string) {
		envName, err := getEnvFromFlag(_flagLogsEnv)
		if err != nil {
			telemetry.Event("cli.logs")
			exit.Error(err)
		}

		env, err := ReadOrConfigureEnv(envName)
		if err != nil {
			telemetry.Event("cli.logs")
			exit.Error(err)
		}
		telemetry.Event("cli.logs", map[string]interface{}{"env_name": env.Name, "random_pod": _flagRandomPod})

		err = printEnvIfNotSpecified(env.Name, cmd)
		if err != nil {
			exit.Error(err)
		}

		operatorConfig := MustGetOperatorConfig(env.Name)
		apiName := args[0]

		if len(args) == 1 {
			if _flagRandomPod {
				err := cluster.StreamLogs(operatorConfig, apiName)
				if err != nil {
					exit.Error(err)
				}
				return
			}

			logResponse, err := cluster.GetLogs(operatorConfig, apiName)
			if err != nil {
				exit.Error(err)
			}
			fmt.Printf(_logsOutput, logResponse.LogURL)
			return
		}

		jobID := args[1]
		if _flagRandomPod {
			err := cluster.StreamJobLogs(operatorConfig, apiName, jobID)
			if err != nil {
				exit.Error(err)
			}
			return
		}

		logResponse, err := cluster.GetJobLogs(operatorConfig, apiName, jobID)
		if err != nil {
			exit.Error(err)
		}
		fmt.Printf(_logsOutput, logResponse.LogURL)
	},
}
