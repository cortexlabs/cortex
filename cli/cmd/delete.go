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
	"github.com/cortexlabs/cortex/cli/local"
	"github.com/cortexlabs/cortex/pkg/lib/exit"
	"github.com/cortexlabs/cortex/pkg/lib/print"
	"github.com/cortexlabs/cortex/pkg/lib/telemetry"
	"github.com/cortexlabs/cortex/pkg/operator/schema"
	"github.com/cortexlabs/cortex/pkg/types"
	"github.com/spf13/cobra"
)

var (
	_flagDeleteEnv       string
	_flagDeleteKeepCache bool
	_flagDeleteForce     bool
)

func deleteInit() {
	_deleteCmd.Flags().SortFlags = false
	_deleteCmd.Flags().StringVarP(&_flagDeleteEnv, "env", "e", getDefaultEnv(_generalCommandType), "environment to use")

	// only applies to aws provider because local doesn't support multiple replicas
	_deleteCmd.Flags().BoolVarP(&_flagDeleteForce, "force", "f", false, "delete the api without confirmation")
	_deleteCmd.Flags().BoolVarP(&_flagDeleteKeepCache, "keep-cache", "c", false, "keep cached data for the api")
}

var _deleteCmd = &cobra.Command{
	Use:   "delete API_NAME [JOB_ID]",
	Short: "delete any kind of api or stop a batch job",
	Args:  cobra.RangeArgs(1, 2),
	Run: func(cmd *cobra.Command, args []string) {
		env, err := ReadOrConfigureEnv(_flagDeleteEnv)
		if err != nil {
			telemetry.Event("cli.delete")
			exit.Error(err)
		}
		telemetry.Event("cli.delete", map[string]interface{}{"provider": env.Provider.String(), "env_name": env.Name})

		err = printEnvIfNotSpecified(_flagDeleteEnv, cmd)
		if err != nil {
			exit.Error(err)
		}

		var deleteResponse schema.DeleteResponse
		if env.Provider == types.AWSProviderType {
			if len(args) == 2 {
				deleteResponse, err = cluster.StopJob(MustGetOperatorConfig(env.Name), args[0], args[1])
				if err != nil {
					exit.Error(err)
				}
			} else {
				deleteResponse, err = cluster.Delete(MustGetOperatorConfig(env.Name), args[0], _flagDeleteKeepCache, _flagDeleteForce)
				if err != nil {
					exit.Error(err)
				}
			}
		} else {
			if len(args) == 2 {
				exit.Error(ErrorNotSupportedInLocalEnvironment(), fmt.Sprintf("cannot delete job %s for api %s", args[1], args[0]))
			}

			// local only supports deploying 1 replica at a time, so _flagDeleteForce is only useful when attempting to delete an API that has been deployed with different CLI version
			deleteResponse, err = local.Delete(args[0], _flagDeleteKeepCache, _flagDeleteForce)
			if err != nil {
				exit.Error(err)
			}
		}

		print.BoldFirstLine(deleteResponse.Message)
	},
}
