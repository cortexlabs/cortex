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
	"time"

	"github.com/cortexlabs/cortex/cli/cluster"
	"github.com/cortexlabs/cortex/cli/types/cliconfig"
	"github.com/cortexlabs/cortex/cli/types/flags"
	"github.com/cortexlabs/cortex/pkg/lib/console"
	"github.com/cortexlabs/cortex/pkg/lib/errors"
	"github.com/cortexlabs/cortex/pkg/lib/exit"
	libjson "github.com/cortexlabs/cortex/pkg/lib/json"
	"github.com/cortexlabs/cortex/pkg/lib/pointer"
	s "github.com/cortexlabs/cortex/pkg/lib/strings"
	"github.com/cortexlabs/cortex/pkg/lib/table"
	"github.com/cortexlabs/cortex/pkg/lib/telemetry"
	libtime "github.com/cortexlabs/cortex/pkg/lib/time"
	"github.com/cortexlabs/cortex/pkg/operator/schema"
	"github.com/cortexlabs/cortex/pkg/types/userconfig"
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
	_getCmd.Flags().StringVarP(&_flagGetEnv, "env", "e", "", "environment to use")
	_getCmd.Flags().BoolVarP(&_flagWatch, "watch", "w", false, "re-run the command every 2 seconds")
	_getCmd.Flags().VarP(&_flagOutput, "output", "o", fmt.Sprintf("output format: one of %s", strings.Join(flags.UserOutputTypeStrings(), "|")))
	addVerboseFlag(_getCmd)
}

var _getCmd = &cobra.Command{
	Use:   "get [API_NAME] [JOB_ID]",
	Short: "get information about apis or jobs",
	Args:  cobra.RangeArgs(0, 2),
	Run: func(cmd *cobra.Command, args []string) {
		var envName string
		if wasEnvFlagProvided(cmd) {
			envName = _flagGetEnv
		} else if len(args) > 0 {
			var err error
			envName, err = getEnvFromFlag("")
			if err != nil {
				telemetry.Event("cli.get")
				exit.Error(err)
			}
		}

		// if API_NAME is specified or env name is provided then the provider is known, otherwise provider isn't because all apis from all environments will be fetched
		if len(args) == 1 || wasEnvFlagProvided(cmd) {
			env, err := ReadOrConfigureEnv(envName)
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
				env, err := ReadOrConfigureEnv(envName)
				if err != nil {
					exit.Error(err)
				}

				out, err := envStringIfNotSpecified(envName, cmd)
				if err != nil {
					return "", err
				}
				apiTable, err := getAPI(env, args[0])
				if err != nil {
					return "", err
				}

				if _flagOutput == flags.JSONOutputType {
					return apiTable, nil
				}

				return out + apiTable, nil
			} else if len(args) == 2 {
				env, err := ReadOrConfigureEnv(envName)
				if err != nil {
					exit.Error(err)
				}

				out, err := envStringIfNotSpecified(envName, cmd)
				if err != nil {
					return "", err
				}

				apisRes, err := cluster.GetAPI(MustGetOperatorConfig(envName), args[0])
				if err != nil {
					return "", err
				}

				var jobTable string
				if apisRes[0].Spec.Kind == userconfig.BatchAPIKind {
					jobTable, err = getBatchJob(env, args[0], args[1])
				} else {
					jobTable, err = getTaskJob(env, args[0], args[1])
				}
				if err != nil {
					return "", err
				}
				if _flagOutput == flags.JSONOutputType {
					return jobTable, nil
				}

				return out + jobTable, nil
			} else {
				envs, err := listConfiguredEnvs()
				if err != nil {
					return "", err
				}
				if len(envs) == 0 {
					return "", ErrorNoAvailableEnvironment()
				}

				if wasEnvFlagProvided(cmd) {
					env, err := ReadOrConfigureEnv(envName)
					if err != nil {
						exit.Error(err)
					}

					out, err := envStringIfNotSpecified(envName, cmd)
					if err != nil {
						return "", err
					}

					apiTable, err := getAPIsByEnv(env, false)
					if err != nil {
						return "", err
					}

					if _flagOutput == flags.JSONOutputType {
						return apiTable, nil
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

	var allRealtimeAPIs []schema.APIResponse
	var allRealtimeAPIEnvs []string
	var allAsyncAPIs []schema.APIResponse
	var allAsyncAPIEnvs []string
	var allBatchAPIs []schema.APIResponse
	var allBatchAPIEnvs []string
	var allTaskAPIs []schema.APIResponse
	var allTaskAPIEnvs []string
	var allTrafficSplitters []schema.APIResponse
	var allTrafficSplitterEnvs []string

	type getAPIsOutput struct {
		EnvName string               `json:"env_name"`
		APIs    []schema.APIResponse `json:"apis"`
		Error   string               `json:"error"`
	}

	allAPIsOutput := []getAPIsOutput{}

	errorsMap := map[string]error{}
	// get apis from both environments
	for _, env := range cliConfig.Environments {
		apisRes, err := cluster.GetAPIs(MustGetOperatorConfig(env.Name))

		apisOutput := getAPIsOutput{
			EnvName: env.Name,
			APIs:    apisRes,
		}

		if err == nil {
			for _, api := range apisRes {
				switch api.Spec.Kind {
				case userconfig.BatchAPIKind:
					allBatchAPIEnvs = append(allBatchAPIEnvs, env.Name)
					allBatchAPIs = append(allBatchAPIs, api)
				case userconfig.RealtimeAPIKind:
					allRealtimeAPIEnvs = append(allRealtimeAPIEnvs, env.Name)
					allRealtimeAPIs = append(allRealtimeAPIs, api)
				case userconfig.AsyncAPIKind:
					allAsyncAPIEnvs = append(allAsyncAPIEnvs, env.Name)
					allAsyncAPIs = append(allAsyncAPIs, api)
				case userconfig.TaskAPIKind:
					allTaskAPIEnvs = append(allTaskAPIEnvs, env.Name)
					allTaskAPIs = append(allTaskAPIs, api)
				case userconfig.TrafficSplitterKind:
					allTrafficSplitterEnvs = append(allTrafficSplitterEnvs, env.Name)
					allTrafficSplitters = append(allTrafficSplitters, api)
				}
			}
		} else {
			apisOutput.Error = err.Error()
			errorsMap[env.Name] = err
		}

		allAPIsOutput = append(allAPIsOutput, apisOutput)
	}

	if _flagOutput == flags.JSONOutputType {
		bytes, err := libjson.Marshal(allAPIsOutput)
		if err != nil {
			return "", err
		}

		return string(bytes), nil
	}

	out := ""

	if len(allRealtimeAPIs) == 0 && len(allAsyncAPIs) == 0 && len(allBatchAPIs) == 0 && len(allTrafficSplitters) == 0 && len(allTaskAPIs) == 0 {
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

		if len(allTaskAPIs) > 0 {
			t := taskAPIsTable(allTaskAPIs, allTaskAPIEnvs)
			if len(allBatchAPIs) > 0 {
				out += "\n"
			}
			out += t.MustFormat()
		}

		if len(allRealtimeAPIs) > 0 {
			t := realtimeAPIsTable(allRealtimeAPIs, allRealtimeAPIEnvs)
			if len(allBatchAPIs) > 0 || len(allTaskAPIs) > 0 {
				out += "\n"
			}
			out += t.MustFormat()
		}
		if len(allAsyncAPIs) > 0 {
			t := asyncAPIsTable(allAsyncAPIs, allAsyncAPIEnvs)
			if len(allBatchAPIs) > 0 || len(allTaskAPIs) > 0 || len(allRealtimeAPIs) > 0 {
				out += "\n"
			}
			out += t.MustFormat()
		}

		if len(allTrafficSplitters) > 0 {
			t := trafficSplitterListTable(allTrafficSplitters, allTrafficSplitterEnvs)

			if len(allBatchAPIs) > 0 || len(allTaskAPIs) > 0 || len(allRealtimeAPIs) > 0 || len(allAsyncAPIs) > 0 {
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

	return out, nil
}

func getAPIsByEnv(env cliconfig.Environment, printEnv bool) (string, error) {
	apisRes, err := cluster.GetAPIs(MustGetOperatorConfig(env.Name))
	if err != nil {
		return "", err
	}

	if _flagOutput == flags.JSONOutputType {
		bytes, err := libjson.Marshal(apisRes)
		if err != nil {
			return "", err
		}
		return string(bytes), nil
	}

	var allRealtimeAPIs []schema.APIResponse
	var allAsyncAPIs []schema.APIResponse
	var allBatchAPIs []schema.APIResponse
	var allTaskAPIs []schema.APIResponse
	var allTrafficSplitters []schema.APIResponse

	for _, api := range apisRes {
		switch api.Spec.Kind {
		case userconfig.BatchAPIKind:
			allBatchAPIs = append(allBatchAPIs, api)
		case userconfig.TaskAPIKind:
			allTaskAPIs = append(allTaskAPIs, api)
		case userconfig.RealtimeAPIKind:
			allRealtimeAPIs = append(allRealtimeAPIs, api)
		case userconfig.AsyncAPIKind:
			allAsyncAPIs = append(allAsyncAPIs, api)
		case userconfig.TrafficSplitterKind:
			allTrafficSplitters = append(allTrafficSplitters, api)
		}
	}

	if len(allRealtimeAPIs) == 0 && len(allBatchAPIs) == 0 && len(allTaskAPIs) == 0 && len(allTrafficSplitters) == 0 {
		return console.Bold("no apis are deployed"), nil
	}

	out := ""

	if len(allBatchAPIs) > 0 {
		envNames := []string{}
		for range allBatchAPIs {
			envNames = append(envNames, env.Name)
		}

		t := batchAPIsTable(allBatchAPIs, envNames)
		t.FindHeaderByTitle(_titleEnvironment).Hidden = true

		out += t.MustFormat()
	}

	if len(allTaskAPIs) > 0 {
		envNames := []string{}
		for range allTaskAPIs {
			envNames = append(envNames, env.Name)
		}

		t := taskAPIsTable(allTaskAPIs, envNames)
		t.FindHeaderByTitle(_titleEnvironment).Hidden = true

		if len(allBatchAPIs) > 0 {
			out += "\n"
		}

		out += t.MustFormat()
	}

	if len(allRealtimeAPIs) > 0 {
		envNames := []string{}
		for range allRealtimeAPIs {
			envNames = append(envNames, env.Name)
		}

		t := realtimeAPIsTable(allRealtimeAPIs, envNames)
		t.FindHeaderByTitle(_titleEnvironment).Hidden = true

		if len(allBatchAPIs) > 0 || len(allTaskAPIs) > 0 {
			out += "\n"
		}

		out += t.MustFormat()
	}

	if len(allAsyncAPIs) > 0 {
		envNames := []string{}
		for range allRealtimeAPIs {
			envNames = append(envNames, env.Name)
		}

		t := asyncAPIsTable(allAsyncAPIs, envNames)
		t.FindHeaderByTitle(_titleEnvironment).Hidden = true

		if len(allBatchAPIs) > 0 || len(allTaskAPIs) > 0 || len(allRealtimeAPIs) > 0 {
			out += "\n"
		}

		out += t.MustFormat()
	}

	if len(allTrafficSplitters) > 0 {
		envNames := []string{}
		for range allTrafficSplitters {
			envNames = append(envNames, env.Name)
		}

		t := trafficSplitterListTable(allTrafficSplitters, envNames)
		t.FindHeaderByTitle(_titleEnvironment).Hidden = true

		if len(allBatchAPIs) > 0 || len(allTaskAPIs) > 0 || len(allRealtimeAPIs) > 0 || len(allAsyncAPIs) > 0 {
			out += "\n"
		}

		out += t.MustFormat()
	}

	return out, nil
}

func getAPI(env cliconfig.Environment, apiName string) (string, error) {
	apisRes, err := cluster.GetAPI(MustGetOperatorConfig(env.Name), apiName)
	if err != nil {
		return "", err
	}

	if _flagOutput == flags.JSONOutputType {
		bytes, err := libjson.Marshal(apisRes)
		if err != nil {
			return "", err
		}
		return string(bytes), nil
	}

	if len(apisRes) == 0 {
		exit.Error(errors.ErrorUnexpected(fmt.Sprintf("unable to find API %s", apiName)))
	}

	apiRes := apisRes[0]

	switch apiRes.Spec.Kind {
	case userconfig.RealtimeAPIKind:
		return realtimeAPITable(apiRes, env)
	case userconfig.AsyncAPIKind:
		return asyncAPITable(apiRes, env)
	case userconfig.TrafficSplitterKind:
		return trafficSplitterTable(apiRes, env)
	case userconfig.BatchAPIKind:
		return batchAPITable(apiRes), nil
	case userconfig.TaskAPIKind:
		return taskAPITable(apiRes), nil
	default:
		return "", errors.ErrorUnexpected(fmt.Sprintf("encountered unexpected kind %s for api %s", apiRes.Spec.Kind, apiRes.Spec.Name))
	}
}

func apiHistoryTable(apiVersions []schema.APIVersion) string {
	t := table.Table{
		Headers: []table.Header{
			{Title: "api id"},
			{Title: "last deployed"},
		},
	}

	t.Rows = make([][]interface{}, len(apiVersions))
	for i, apiVersion := range apiVersions {
		lastUpdated := time.Unix(apiVersion.LastUpdated, 0)
		t.Rows[i] = []interface{}{apiVersion.APIID, libtime.SinceStr(&lastUpdated)}
	}

	return t.MustFormat(&table.Opts{Sort: pointer.Bool(false)})
}

func titleStr(title string) string {
	return "\n" + console.Bold(title) + "\n"
}
