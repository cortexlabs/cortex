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
	"github.com/cortexlabs/cortex/cli/types/cliconfig"
	"github.com/cortexlabs/cortex/pkg/lib/errors"
	"github.com/cortexlabs/cortex/pkg/lib/exit"
	"github.com/cortexlabs/cortex/pkg/lib/telemetry"
	"github.com/cortexlabs/cortex/pkg/types/userconfig"
	"github.com/spf13/cobra"
)

const (
	_titleReplicaStatus = "replica status"
	_titleReplicaCount  = "replica count"
)

var (
	_flagDescribeEnv   string
	_flagDescribeWatch bool
)

func describeInit() {
	_describeCmd.Flags().SortFlags = false
	_describeCmd.Flags().StringVarP(&_flagDescribeEnv, "env", "e", "", "environment to use")
	_describeCmd.Flags().BoolVarP(&_flagDescribeWatch, "watch", "w", false, "re-run the command every 2 seconds")
}

var _describeCmd = &cobra.Command{
	Use:   "describe [API_NAME]",
	Short: "describe an api",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		apiName := args[0]

		var envName string
		if wasFlagProvided(cmd, "env") {
			envName = _flagDescribeEnv
		} else {
			var err error
			envName, err = getEnvFromFlag("")
			if err != nil {
				telemetry.Event("cli.describe")
				exit.Error(err)
			}
		}

		env, err := ReadOrConfigureEnv(envName)
		if err != nil {
			telemetry.Event("cli.describe")
			exit.Error(err)
		}
		telemetry.Event("cli.describe", map[string]interface{}{"env_name": env.Name})

		rerun(_flagDescribeWatch, func() (string, error) {
			env, err := ReadOrConfigureEnv(envName)
			if err != nil {
				exit.Error(err)
			}

			out, err := envStringIfNotSpecified(envName, cmd)
			if err != nil {
				return "", err
			}
			apiTable, err := describeAPI(env, apiName)
			if err != nil {
				return "", err
			}

			return out + apiTable, nil
		})
	},
}

func describeAPI(env cliconfig.Environment, apiName string) (string, error) {
	apisRes, err := cluster.DescribeAPI(MustGetOperatorConfig(env.Name), apiName)
	if err != nil {
		return "", err
	}

	if len(apisRes) == 0 {
		exit.Error(errors.ErrorUnexpected(fmt.Sprintf("unable to find api %s", apiName)))
	}

	apiRes := apisRes[0]

	switch apiRes.Metadata.Kind {
	case userconfig.RealtimeAPIKind:
		return realtimeDescribeAPITable(apiRes, env)
	case userconfig.AsyncAPIKind:
		return asyncDescribeAPITable(apiRes, env)
	default:
		return "", errors.ErrorUnexpected(fmt.Sprintf("encountered unexpected kind %s for api %s", apiRes.Metadata.Kind, apiRes.Metadata.Name))
	}
}
