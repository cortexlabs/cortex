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
	"github.com/cortexlabs/cortex/cli/cluster"
	"github.com/cortexlabs/cortex/pkg/lib/exit"
	"github.com/cortexlabs/cortex/pkg/lib/prompt"
	"github.com/cortexlabs/cortex/pkg/lib/telemetry"
	"github.com/cortexlabs/cortex/pkg/types/userconfig"
	"github.com/spf13/cobra"
)

var (
	_flagLogsEnv            string
	_flagLogsDisallowPrompt bool
)

func logsInit() {
	_logsCmd.Flags().SortFlags = false
	_logsCmd.Flags().StringVarP(&_flagLogsEnv, "env", "e", "", "environment to use")
	_logsCmd.Flags().BoolVarP(&_flagLogsDisallowPrompt, "yes", "y", false, "skip prompts")
}

var _logsCmd = &cobra.Command{
	Use:   "logs API_NAME [JOB_ID]",
	Short: "stream logs from a single replica of an api or a single worker for a job",
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
		telemetry.Event("cli.logs", map[string]interface{}{"provider": env.Provider.String(), "env_name": env.Name})

		err = printEnvIfNotSpecified(env.Name, cmd)
		if err != nil {
			exit.Error(err)
		}

		operatorConfig := MustGetOperatorConfig(env.Name)
		apiName := args[0]

		apiResponse, err := cluster.GetAPI(operatorConfig, apiName)
		if err != nil {
			exit.Error(err)
		}

		if len(args) == 1 {
			if apiResponse[0].Spec.Kind == userconfig.RealtimeAPIKind && apiResponse[0].Status.Requested > 1 && !_flagLogsDisallowPrompt {
				prompt.YesOrExit("logs from a single random replica will be streamed\n\nfor aggregated logs please visit your cloud provider's logging dashboard; see https://docs.cortex.dev for details", "", "")
			}

			err = cluster.StreamLogs(operatorConfig, apiName)
			if err != nil {
				exit.Error(err)
			}
		}

		if len(args) == 2 {
			if apiResponse[0].Spec.Kind == userconfig.BatchAPIKind {
				jobResponse, err := cluster.GetBatchJob(operatorConfig, apiName, args[1])
				if err != nil {
					exit.Error(err)
				}

				if jobResponse.JobStatus.Workers > 1 && !_flagLogsDisallowPrompt {
					prompt.YesOrExit("logs from a single random worker will be streamed\n\nfor aggregated logs please visit your cloud provider's logging dashboard; see https://docs.cortex.dev for details", "", "")
				}
			}

			err = cluster.StreamJobLogs(operatorConfig, apiName, args[1])
			if err != nil {
				exit.Error(err)
			}
		}
	},
}
