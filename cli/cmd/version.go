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

	"github.com/cortexlabs/cortex/cli/cluster"
	"github.com/cortexlabs/cortex/pkg/consts"
	"github.com/cortexlabs/cortex/pkg/lib/exit"
	"github.com/cortexlabs/cortex/pkg/lib/telemetry"
	"github.com/cortexlabs/cortex/pkg/types"
	"github.com/spf13/cobra"
)

var _flagVersionEnv string

func versionInit() {
	_versionCmd.Flags().SortFlags = false
	_versionCmd.Flags().StringVarP(&_flagVersionEnv, "env", "e", getDefaultEnv(_generalCommandType), "environment to use")
}

var _versionCmd = &cobra.Command{
	Use:   "version",
	Short: "print the cli and cluster versions",
	Args:  cobra.NoArgs,
	Run: func(cmd *cobra.Command, args []string) {
		env, err := ReadOrConfigureEnv(_flagVersionEnv)
		if err != nil {
			telemetry.Event("cli.version")
			exit.Error(err)
		}
		telemetry.Event("cli.version", map[string]interface{}{"provider": env.Provider.String(), "env_name": env.Name})

		err = printEnvIfNotSpecified(_flagVersionEnv, cmd)
		if err != nil {
			exit.Error(err)
		}

		fmt.Println("cli version: " + consts.CortexVersion)

		if env.Provider == types.LocalProviderType {
			return
		}

		infoResponse, err := cluster.Info(MustGetOperatorConfig(env.Name))
		if err != nil {
			exit.Error(err)
		}

		fmt.Println("cluster version: " + infoResponse.ClusterConfig.APIVersion)
	},
}
