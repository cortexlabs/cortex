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
	"strings"

	"github.com/cortexlabs/cortex/cli/cluster"
	"github.com/cortexlabs/cortex/cli/local"
	"github.com/cortexlabs/cortex/cli/types/cliconfig"
	"github.com/cortexlabs/cortex/pkg/lib/console"
	"github.com/cortexlabs/cortex/pkg/lib/errors"
	"github.com/cortexlabs/cortex/pkg/lib/exit"
	"github.com/cortexlabs/cortex/pkg/lib/sets/strset"
	s "github.com/cortexlabs/cortex/pkg/lib/strings"
	"github.com/cortexlabs/cortex/pkg/lib/table"
	"github.com/cortexlabs/cortex/pkg/lib/telemetry"
	"github.com/cortexlabs/cortex/pkg/operator/schema"
	"github.com/cortexlabs/cortex/pkg/types"
	"github.com/spf13/cobra"
)

const (
	_titleEnvironment = "env"
	_titleRealtimeAPI = "realtime api"
	_titleStatus      = "status"
	_titleUpToDate    = "up-to-date"
	_titleStale       = "stale"
	_titleRequested   = "requested"
	_titleFailed      = "failed"
	_titleLastupdated = "last update"
	_titleAvgRequest  = "avg request"
	_title2XX         = "2XX"
	_title4XX         = "4XX"
	_title5XX         = "5XX"
)

var (
	_flagGetEnv string
	_flagWatch  bool
)

func getInit() {
	_getCmd.Flags().SortFlags = false
	_getCmd.Flags().StringVarP(&_flagGetEnv, "env", "e", getDefaultEnv(_generalCommandType), "environment to use")
	_getCmd.Flags().BoolVarP(&_flagWatch, "watch", "w", false, "re-run the command every 2 seconds")
}

var _getCmd = &cobra.Command{
	Use:   "get [API_NAME] [JOB_ID]",
	Short: "get information about apis or jobs",
	Args:  cobra.RangeArgs(0, 2),
	Run: func(cmd *cobra.Command, args []string) {
		// if API_NAME is specified or env name is provided then the provider is known, otherwise provider isn't because all apis from all environments will be fetched
		if len(args) == 1 || wasEnvFlagProvided(cmd) {
			env, err := ReadOrConfigureEnv(_flagGetEnv)
			if err != nil {
				telemetry.Event("cli.get")
				exit.Error(err)
			}
			telemetry.Event("cli.get", map[string]interface{}{"provider": env.Provider.String(), "env_name": env.Name})
		} else {
			telemetry.Event("cli.get")
		}

		rerun(func() (string, error) {
			if len(args) == 1 {
				env, err := ReadOrConfigureEnv(_flagGetEnv)
				if err != nil {
					exit.Error(err)
				}

				out, err := envStringIfNotSpecified(_flagGetEnv, cmd)
				if err != nil {
					return "", err
				}
				apiTable, err := getAPI(env, args[0])
				if err != nil {
					return "", err
				}
				return out + apiTable, nil
			} else if len(args) == 2 {
				env, err := ReadOrConfigureEnv(_flagGetEnv)
				if err != nil {
					exit.Error(err)
				}

				out, err := envStringIfNotSpecified(_flagGetEnv, cmd)
				if err != nil {
					return "", err
				}

				if env.Provider == types.LocalProviderType {
					return "", errors.Wrap(ErrorNotSupportedInLocalEnvironment(), fmt.Sprintf("cannot get status of job %s for api %s", args[1], args[0]))
				}

				apiTable, err := getJob(env, args[0], args[1])
				if err != nil {
					return "", err
				}
				return out + apiTable, nil
			} else {
				if wasEnvFlagProvided(cmd) {
					env, err := ReadOrConfigureEnv(_flagGetEnv)
					if err != nil {
						exit.Error(err)
					}

					out, err := envStringIfNotSpecified(_flagGetEnv, cmd)
					if err != nil {
						return "", err
					}

					apiTable, err := getAPIsByEnv(env, false)
					if err != nil {
						return "", err
					}
					return out + apiTable, nil
				}

				out, err := getAPIsInAllEnvironments()
				if err != nil {
					return "", err
				}

				return out, nil
			}
		})
	},
}

func getAPIsInAllEnvironments() (string, error) {
	cliConfig, err := readCLIConfig()
	if err != nil {
		return "", err
	}

	var allRealtimeAPIs []schema.RealtimeAPI
	var allRealtimeAPIEnvs []string
	var allBatchAPIs []schema.BatchAPI
	var allBatchAPIEnvs []string
	var allTrafficSplitters []schema.TrafficSplitter
	var allTrafficSplitterEnvs []string

	errorsMap := map[string]error{}
	// get apis from both environments
	for _, env := range cliConfig.Environments {
		var apisRes schema.GetAPIsResponse
		var err error
		if env.Provider == types.AWSProviderType {
			apisRes, err = cluster.GetAPIs(MustGetOperatorConfig(env.Name))
		} else {
			apisRes, err = local.GetAPIs()
		}

		if err == nil {
			for range apisRes.BatchAPIs {
				allBatchAPIEnvs = append(allBatchAPIEnvs, env.Name)
			}
			for range apisRes.RealtimeAPIs {
				allRealtimeAPIEnvs = append(allRealtimeAPIEnvs, env.Name)
			}
			for range apisRes.TrafficSplitters {
				allTrafficSplitterEnvs = append(allTrafficSplitterEnvs, env.Name)
			}
			allRealtimeAPIs = append(allRealtimeAPIs, apisRes.RealtimeAPIs...)
			allBatchAPIs = append(allBatchAPIs, apisRes.BatchAPIs...)
			allTrafficSplitters = append(allTrafficSplitters, apisRes.TrafficSplitters...)
		} else {
			errorsMap[env.Name] = err
		}
	}

	out := ""

	if len(allRealtimeAPIs) == 0 && len(allBatchAPIs) == 0 && len(allTrafficSplitters) == 0 {
		// check if any environments errorred
		if len(errorsMap) != len(cliConfig.Environments) {
			if len(errorsMap) == 0 {
				return console.Bold("no apis are deployed"), nil
			}

			var successfulEnvs []string
			for _, env := range cliConfig.Environments {
				if _, ok := errorsMap[env.Name]; !ok {
					successfulEnvs = append(successfulEnvs, env.Name)
				}
			}
			fmt.Println(console.Bold(fmt.Sprintf("no apis are deployed in %s: %s", s.PluralS("environment", len(successfulEnvs)), s.StrsAnd(successfulEnvs))) + "\n")
		}

		// Print the first error
		for name, err := range errorsMap {
			if err != nil {
				exit.Error(errors.Wrap(err, "env "+name))
			}
		}
	} else {
		if len(allBatchAPIs) > 0 {
			t := batchAPIsTable(allBatchAPIs, allBatchAPIEnvs)
			out += t.MustFormat()
		}

		if len(allRealtimeAPIs) > 0 {
			t := realtimeAPIsTable(allRealtimeAPIs, allRealtimeAPIEnvs)
			if strset.New(allRealtimeAPIEnvs...).IsEqual(strset.New(types.LocalProviderType.String())) {
				hideReplicaCountColumns(&t)
			}

			if len(allBatchAPIs) > 0 {
				out += "\n"
			}

			out += t.MustFormat()
		}

		if len(allTrafficSplitters) > 0 {
			t := trafficSplitterListTable(allTrafficSplitters, allTrafficSplitterEnvs)

			if len(allRealtimeAPIs) > 0 || len(allBatchAPIs) > 0 {
				out += "\n"
			}

			out += t.MustFormat()
		}
	}

	if len(errorsMap) == 1 {
		out = s.EnsureBlankLineIfNotEmpty(out)
		out += fmt.Sprintf("unable to detect apis from the %s environment; run `cortex get --env %s` if this is unexpected\n", errors.FirstKeyInErrorMap(errorsMap), errors.FirstKeyInErrorMap(errorsMap))
	} else if len(errorsMap) > 1 {
		out = s.EnsureBlankLineIfNotEmpty(out)
		out += fmt.Sprintf("unable to detect apis from the %s environments; run `cortex get --env ENV_NAME` if this is unexpected\n", s.StrsAnd(errors.NonNilErrorMapKeys(errorsMap)))
	}

	mismatchedAPIMessage, err := getLocalVersionMismatchedAPIsMessage()
	if err == nil {
		out = s.EnsureBlankLineIfNotEmpty(out)
		out += mismatchedAPIMessage
	}

	return out, nil
}

func hideReplicaCountColumns(t *table.Table) {
	t.FindHeaderByTitle(_titleUpToDate).Hidden = true
	t.FindHeaderByTitle(_titleStale).Hidden = true
	t.FindHeaderByTitle(_titleRequested).Hidden = true
	t.FindHeaderByTitle(_titleFailed).Hidden = true
}

func getAPIsByEnv(env cliconfig.Environment, printEnv bool) (string, error) {
	var apisRes schema.GetAPIsResponse
	var err error

	if env.Provider == types.AWSProviderType {
		apisRes, err = cluster.GetAPIs(MustGetOperatorConfig(env.Name))
		if err != nil {
			return "", err
		}
	} else {
		apisRes, err = local.GetAPIs()
		if err != nil {
			return "", err
		}
	}

	if len(apisRes.RealtimeAPIs) == 0 && len(apisRes.BatchAPIs) == 0 && len(apisRes.TrafficSplitters) == 0 {
		return console.Bold("no apis are deployed"), nil
	}

	out := ""

	if len(apisRes.BatchAPIs) > 0 {
		envNames := []string{}
		for range apisRes.BatchAPIs {
			envNames = append(envNames, env.Name)
		}

		t := batchAPIsTable(apisRes.BatchAPIs, envNames)
		t.FindHeaderByTitle(_titleEnvironment).Hidden = true

		out += t.MustFormat()
	}

	if len(apisRes.RealtimeAPIs) > 0 {
		envNames := []string{}
		for range apisRes.RealtimeAPIs {
			envNames = append(envNames, env.Name)
		}

		t := realtimeAPIsTable(apisRes.RealtimeAPIs, envNames)
		t.FindHeaderByTitle(_titleEnvironment).Hidden = true

		if len(apisRes.BatchAPIs) > 0 {
			out += "\n"
		}

		if env.Provider == types.LocalProviderType {
			hideReplicaCountColumns(&t)
		}

		out += t.MustFormat()
	}

	if len(apisRes.TrafficSplitters) > 0 {
		envNames := []string{}
		for range apisRes.TrafficSplitters {
			envNames = append(envNames, env.Name)
		}

		t := trafficSplitterListTable(apisRes.TrafficSplitters, envNames)
		t.FindHeaderByTitle(_titleEnvironment).Hidden = true

		if len(apisRes.BatchAPIs) > 0 || len(apisRes.RealtimeAPIs) > 0 {
			out += "\n"
		}

		out += t.MustFormat()
	}

	if env.Provider == types.LocalProviderType {
		mismatchedVersionAPIsErrorMessage, _ := getLocalVersionMismatchedAPIsMessage()
		if len(mismatchedVersionAPIsErrorMessage) > 0 {
			out += "\n" + mismatchedVersionAPIsErrorMessage
		}
	}

	return out, nil
}

func getLocalVersionMismatchedAPIsMessage() (string, error) {
	mismatchedAPINames, err := local.ListVersionMismatchedAPIs()
	if err != nil {
		return "", err
	}
	if len(mismatchedAPINames) == 0 {
		return "", nil
	}

	if len(mismatchedAPINames) == 1 {
		return fmt.Sprintf("an api named %s was deployed in your local environment using a different version of the cortex cli; please delete them using `cortex delete %s` and then redeploy them\n", s.UserStr(mismatchedAPINames[0]), mismatchedAPINames[0]), nil
	}
	return fmt.Sprintf("apis named %s were deployed in your local environment using a different version of the cortex cli; please delete them using `cortex delete API_NAME` and then redeploy them\n", s.UserStrsAnd(mismatchedAPINames)), nil
}

func getAPI(env cliconfig.Environment, apiName string) (string, error) {
	if env.Provider == types.AWSProviderType {
		apiRes, err := cluster.GetAPI(MustGetOperatorConfig(env.Name), apiName)
		if err != nil {
			// note: if modifying this string, search the codebase for it and change all occurrences
			if strings.HasSuffix(errors.Message(err), "is not deployed") {
				return console.Bold(errors.Message(err)), nil
			}
			return "", err
		}

		if apiRes.RealtimeAPI != nil {
			return realtimeAPITable(apiRes.RealtimeAPI, env)
		}
		if apiRes.TrafficSplitter != nil {
			return trafficSplitterTable(apiRes.TrafficSplitter, env)
		}
		return batchAPITable(*apiRes.BatchAPI), nil
	}

	apiRes, err := local.GetAPI(apiName)
	if err != nil {
		// note: if modifying this string, search the codebase for it and change all occurrences
		if strings.HasSuffix(errors.Message(err), "is not deployed") {
			return console.Bold(errors.Message(err)), nil
		}
		return "", err
	}

	return realtimeAPITable(apiRes.RealtimeAPI, env)
}

func titleStr(title string) string {
	return "\n" + console.Bold(title) + "\n"
}
