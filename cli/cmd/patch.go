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
	_flagPatchEnv   string
	_flagPatchForce bool
)

func patchInit() {
	_patchCmd.Flags().SortFlags = false
	_patchCmd.Flags().StringVarP(&_flagPatchEnv, "env", "e", "", "environment to use")
	_patchCmd.Flags().BoolVarP(&_flagPatchForce, "force", "f", false, "override the in-progress api update")
	_patchCmd.Flags().VarP(&_flagOutput, "output", "o", fmt.Sprintf("output format: one of %s", strings.Join(flags.UserOutputTypeStrings(), "|")))
}

var _patchCmd = &cobra.Command{
	Use:   "patch [CONFIG_FILE]",
	Short: "update API configuration for a deployed API",
	Args:  cobra.RangeArgs(0, 1),
	Run: func(cmd *cobra.Command, args []string) {
		envName, err := getEnvFromFlag(_flagPatchEnv)
		if err != nil {
			telemetry.Event("cli.patch")
			exit.Error(err)
		}

		env, err := ReadOrConfigureEnv(envName)
		if err != nil {
			telemetry.Event("cli.patch")
			exit.Error(err)
		}
		telemetry.Event("cli.patch", map[string]interface{}{"provider": env.Provider.String(), "env_name": env.Name})

		err = printEnvIfNotSpecified(env.Name, cmd)
		if err != nil {
			exit.Error(err)
		}

		configPath := getConfigPath(args)

		deployResults, err := cluster.Patch(MustGetOperatorConfig(env.Name), configPath, _flagPatchForce)
		if err != nil {
			exit.Error(err)
		}

		switch _flagOutput {
		case flags.JSONOutputType:
			bytes, err := libjson.Marshal(deployResults)
			if err != nil {
				exit.Error(err)
			}
			fmt.Print(string(bytes))
		case flags.PrettyOutputType:
			message, err := deployMessage(deployResults, env.Name)
			if err != nil {
				exit.Error(err)
			}
			if didAnyResultsError(deployResults) {
				print.StderrBoldFirstBlock(message)
			} else {
				print.BoldFirstBlock(message)
			}
		}
	},
}
