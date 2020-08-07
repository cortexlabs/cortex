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
	"github.com/cortexlabs/cortex/cli/cluster"
	"github.com/cortexlabs/cortex/pkg/lib/exit"
	"github.com/cortexlabs/cortex/pkg/lib/print"
	"github.com/cortexlabs/cortex/pkg/lib/telemetry"
	"github.com/cortexlabs/cortex/pkg/types"
	"github.com/spf13/cobra"
)

var (
	_flagRefreshEnv   string
	_flagRefreshForce bool
)

func refreshInit() {
	_refreshCmd.Flags().SortFlags = false
	_refreshCmd.Flags().StringVarP(&_flagRefreshEnv, "env", "e", getDefaultEnv(_generalCommandType), "environment to use")
	_refreshCmd.Flags().BoolVarP(&_flagRefreshForce, "force", "f", false, "override the in-progress api update")
}

var _refreshCmd = &cobra.Command{
	Use:   "refresh API_NAME",
	Short: "restart all replicas for an api (without downtime)",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		env, err := ReadOrConfigureEnv(_flagRefreshEnv)
		if err != nil {
			telemetry.Event("cli.refresh")
			exit.Error(err)
		}
		telemetry.Event("cli.refresh", map[string]interface{}{"provider": env.Provider.String(), "env_name": env.Name})

		err = printEnvIfNotSpecified(_flagRefreshEnv, cmd)
		if err != nil {
			exit.Error(err)
		}

		if env.Provider == types.LocalProviderType {
			print.BoldFirstLine("`cortex refresh` is not supported in the local environment; use `cortex deploy` instead")
			return
		}
		refreshResponse, err := cluster.Refresh(MustGetOperatorConfig(env.Name), args[0], _flagRefreshForce)
		if err != nil {
			exit.Error(err)
		}
		print.BoldFirstLine(refreshResponse.Message)
	},
}
