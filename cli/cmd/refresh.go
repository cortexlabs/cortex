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

	"github.com/cortexlabs/cortex/cli/cluster"
	"github.com/cortexlabs/cortex/cli/types/flags"
	"github.com/cortexlabs/cortex/pkg/lib/exit"
	libjson "github.com/cortexlabs/cortex/pkg/lib/json"
	"github.com/cortexlabs/cortex/pkg/lib/print"
	"github.com/cortexlabs/cortex/pkg/lib/telemetry"
	"github.com/spf13/cobra"
)

var (
	_flagRefreshEnv   string
	_flagRefreshForce bool
)

func refreshInit() {
	_refreshCmd.Flags().SortFlags = false
	_refreshCmd.Flags().StringVarP(&_flagRefreshEnv, "env", "e", "", "environment to use")
	_refreshCmd.Flags().BoolVarP(&_flagRefreshForce, "force", "f", false, "override the in-progress api update")
	_refreshCmd.Flags().VarP(&_flagOutput, "output", "o", fmt.Sprintf("output format: one of %s", strings.Join(flags.UserOutputTypeStrings(), "|")))
}

var _refreshCmd = &cobra.Command{
	Use:   "refresh API_NAME",
	Short: "restart all replicas for an api (without downtime)",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		envName, err := getEnvFromFlag(_flagRefreshEnv)
		if err != nil {
			telemetry.Event("cli.refresh")
			exit.Error(err)
		}

		env, err := ReadOrConfigureEnv(envName)
		if err != nil {
			telemetry.Event("cli.refresh")
			exit.Error(err)
		}
		telemetry.Event("cli.refresh", map[string]interface{}{"provider": env.Provider.String(), "env_name": env.Name})

		err = printEnvIfNotSpecified(env.Name, cmd)
		if err != nil {
			exit.Error(err)
		}

		refreshResponse, err := cluster.Refresh(MustGetOperatorConfig(env.Name), args[0], _flagRefreshForce)
		if err != nil {
			exit.Error(err)
		}

		if _flagOutput == flags.JSONOutputType {
			bytes, err := libjson.Marshal(refreshResponse)
			if err != nil {
				exit.Error(err)
			}
			fmt.Print(string(bytes))
			return
		}

		print.BoldFirstLine(refreshResponse.Message)
	},
}
