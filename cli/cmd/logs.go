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
	"net/http"
	"net/url"

	"github.com/cortexlabs/cortex/cli/cluster"
	"github.com/cortexlabs/cortex/cli/local"
	"github.com/cortexlabs/cortex/pkg/lib/console"
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
	Use:   "logs API_NAME [JOB_ID]",
	Short: "stream logs from an api",
	Args:  cobra.RangeArgs(1, 2),
	Run: func(cmd *cobra.Command, args []string) {
		env, err := ReadOrConfigureEnv(_flagLogsEnv)
		if err != nil {
			telemetry.Event("cli.logs")
			exit.Error(err)
		}
		telemetry.Event("cli.logs", map[string]interface{}{"provider": env.Provider.String(), "env_name": env.Name})

		err = printEnvIfNotSpecified(_flagLogsEnv, cmd)
		if err != nil {
			exit.Error(err)
		}

		apiName := args[0]
		if env.Provider == types.AWSProviderType {
			if len(args) == 1 {
				err := cluster.StreamLogs(MustGetOperatorConfig(env.Name), apiName)
				if err != nil {
					exit.Error(err)
				}
			}
			if len(args) == 2 {
				err := cluster.StreamJobLogs(MustGetOperatorConfig(env.Name), apiName, args[1])
				if err != nil {
					exit.Error(err)
				}
			}
		}
		if env.Provider == types.GCPProviderType {
			gcpLogsResponse, err := cluster.GetGCPLogsURL(MustGetOperatorConfig(env.Name), apiName)
			if err != nil {
				exit.Error(err)
			}

			gcpReq, err := http.NewRequest("GET", "https://console.cloud.google.com/logs/query", nil)
			if err != nil {
				exit.Error(err)
			}
			query := ""
			for q, v := range gcpLogsResponse.Query {
				query += fmt.Sprintf("%s=\"%s\"\n", q, v.Param)
			}
			queryValues := make(url.Values)
			queryValues.Add("query", query)
			gcpReq.URL.RawQuery = queryValues.Encode()

			gcpLogsURL := gcpReq.URL.String()
			consoleOutput := console.Bold(fmt.Sprintf("api %s logs: ", apiName)) + gcpLogsURL
			fmt.Println(consoleOutput)
		}
		if env.Provider == types.LocalProviderType {
			if len(args) == 2 {
				exit.Error(ErrorNotSupportedInLocalEnvironment(), fmt.Sprintf("cannot stream logs for job %s for api %s", args[1], args[0]))
			}
			err := local.StreamLogs(apiName)
			if err != nil {
				exit.Error(err)
			}
		}
	},
}
