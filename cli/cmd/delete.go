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
	"github.com/cortexlabs/cortex/pkg/operator/schema"
	"github.com/spf13/cobra"
)

var (
	_flagDeleteEnv       string
	_flagDeleteKeepCache bool
	_flagDeleteForce     bool
)

func deleteInit() {
	_deleteCmd.Flags().SortFlags = false
	_deleteCmd.Flags().StringVarP(&_flagDeleteEnv, "env", "e", "", "environment to use")

	_deleteCmd.Flags().BoolVarP(&_flagDeleteForce, "force", "f", false, "delete the api without confirmation")
	_deleteCmd.Flags().BoolVarP(&_flagDeleteKeepCache, "keep-cache", "c", false, "keep cached data for the api")
	_deleteCmd.Flags().VarP(&_flagOutput, "output", "o", fmt.Sprintf("output format: one of %s", strings.Join(flags.UserOutputTypeStrings(), "|")))
}

var _deleteCmd = &cobra.Command{
	Use:   "delete API_NAME [JOB_ID]",
	Short: "delete any kind of api or stop a batch job",
	Args:  cobra.RangeArgs(1, 2),
	Run: func(cmd *cobra.Command, args []string) {
		envName, err := getEnvFromFlag(_flagDeleteEnv)
		if err != nil {
			telemetry.Event("cli.delete")
			exit.Error(err)
		}

		env, err := ReadOrConfigureEnv(envName)
		if err != nil {
			telemetry.Event("cli.delete")
			exit.Error(err)
		}
		telemetry.Event("cli.delete", map[string]interface{}{"provider": env.Provider.String(), "env_name": env.Name})

		err = printEnvIfNotSpecified(env.Name, cmd)
		if err != nil {
			exit.Error(err)
		}

		var deleteResponse schema.DeleteResponse
		if len(args) == 2 {
			apisRes, err := cluster.GetAPI(MustGetOperatorConfig(env.Name), args[0])
			if err != nil {
				exit.Error(err)
			}

			deleteResponse, err = cluster.StopJob(MustGetOperatorConfig(env.Name), apisRes[0].Spec.Kind, args[0], args[1])
			if err != nil {
				exit.Error(err)
			}
		} else {
			deleteResponse, err = cluster.Delete(MustGetOperatorConfig(env.Name), args[0], _flagDeleteKeepCache, _flagDeleteForce)
			if err != nil {
				exit.Error(err)
			}
		}

		if _flagOutput == flags.JSONOutputType {
			bytes, err := libjson.Marshal(deleteResponse)
			if err != nil {
				exit.Error(err)
			}
			fmt.Print(string(bytes))
			return
		}

		print.BoldFirstLine(deleteResponse.Message)
	},
}
