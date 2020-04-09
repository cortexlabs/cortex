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

	"github.com/cortexlabs/cortex/pkg/consts"
	"github.com/cortexlabs/cortex/pkg/lib/exit"
	"github.com/cortexlabs/cortex/pkg/lib/json"
	"github.com/cortexlabs/cortex/pkg/lib/telemetry"
	"github.com/cortexlabs/cortex/pkg/operator/schema"
	"github.com/cortexlabs/cortex/pkg/types"
	"github.com/spf13/cobra"
)

func versionInit() {
	addEnvFlag(_versionCmd, _generalCommandType, _envToUseUsage)
}

var _versionCmd = &cobra.Command{
	Use:   "version",
	Short: "print the cli and cluster versions",
	Args:  cobra.NoArgs,
	Run: func(cmd *cobra.Command, args []string) {
		telemetry.Event("cli.version")

		env, err := readEnv(_flagEnv)
		if err != nil || env.Provider == types.LocalProviderType {
			fmt.Println("cli version: " + consts.CortexVersion)
			return
		}

		fmt.Println()

		httpResponse, err := HTTPGet("/info")
		if err != nil {
			fmt.Println("cli version: " + consts.CortexVersion + "\n")
			exit.Error(err)
		}
		var infoResponse schema.InfoResponse
		err = json.Unmarshal(httpResponse, &infoResponse)
		if err != nil {
			fmt.Println("cli version: " + consts.CortexVersion + "\n")
			exit.Error(err, "/info", string(httpResponse))
		}

		fmt.Println("cli version:     " + consts.CortexVersion)
		fmt.Println("cluster version: " + infoResponse.ClusterConfig.APIVersion)
	},
}
