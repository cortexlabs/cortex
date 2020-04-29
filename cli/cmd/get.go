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
	"crypto/tls"
	"fmt"
	"io/ioutil"
	"net/http"
	"sort"
	"strings"
	"time"

	"github.com/cortexlabs/cortex/cli/cluster"
	"github.com/cortexlabs/cortex/cli/local"
	"github.com/cortexlabs/cortex/cli/types/cliconfig"
	"github.com/cortexlabs/cortex/pkg/consts"
	"github.com/cortexlabs/cortex/pkg/lib/cast"
	"github.com/cortexlabs/cortex/pkg/lib/console"
	"github.com/cortexlabs/cortex/pkg/lib/errors"
	"github.com/cortexlabs/cortex/pkg/lib/exit"
	"github.com/cortexlabs/cortex/pkg/lib/json"
	"github.com/cortexlabs/cortex/pkg/lib/sets/strset"
	s "github.com/cortexlabs/cortex/pkg/lib/strings"
	"github.com/cortexlabs/cortex/pkg/lib/table"
	"github.com/cortexlabs/cortex/pkg/lib/telemetry"
	libtime "github.com/cortexlabs/cortex/pkg/lib/time"
	"github.com/cortexlabs/cortex/pkg/lib/urls"
	"github.com/cortexlabs/cortex/pkg/operator/schema"
	"github.com/cortexlabs/cortex/pkg/types"
	"github.com/cortexlabs/cortex/pkg/types/metrics"
	"github.com/cortexlabs/cortex/pkg/types/spec"
	"github.com/cortexlabs/cortex/pkg/types/status"
	"github.com/cortexlabs/cortex/pkg/types/userconfig"
	"github.com/spf13/cobra"
)

const (
	_titleEnvironment = "environment"
	_titleAPI         = "api"
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
	_getCmd.Flags().BoolVarP(&_flagWatch, "watch", "w", false, "re-run the command every second")
}

var _getCmd = &cobra.Command{
	Use:   "get [API_NAME]",
	Short: "get information about apis",
	Args:  cobra.RangeArgs(0, 1),
	Run: func(cmd *cobra.Command, args []string) {
		// if API_NAME is specified or env name is provided then the provider is known, otherwise provider isn't because all apis from all environments will be fetched
		if len(args) == 1 || wasEnvFlagProvided() {
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

				out, err := envStringIfNotSpecified(_flagGetEnv)
				if err != nil {
					return "", err
				}

				apiTable, err := getAPI(env, args[0])
				if err != nil {
					return "", err
				}
				return out + apiTable, nil
			}

			if wasEnvFlagProvided() {
				env, err := ReadOrConfigureEnv(_flagGetEnv)
				if err != nil {
					exit.Error(err)
				}

				out, err := envStringIfNotSpecified(_flagGetEnv)
				if err != nil {
					return "", err
				}

				apiTable, err := getAPIs(env, false)
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
		})
	},
}

func getAPIsInAllEnvironments() (string, error) {
	cliConfig, err := readCLIConfig()
	if err != nil {
		return "", err
	}

	var errorEnvNames []string
	var allAPIs []spec.API
	var allAPIStatuses []status.Status
	var allMetrics []metrics.Metrics
	var allEnvs []string
	for _, env := range cliConfig.Environments {
		var apisRes schema.GetAPIsResponse
		var err error
		if env.Provider == types.AWSProviderType {
			apisRes, err = cluster.GetAPIs(MustGetOperatorConfig(env.Name))
		} else {
			apisRes, err = local.GetAPIs()
		}

		if err == nil {
			for range apisRes.APIs {
				allEnvs = append(allEnvs, env.Name)
			}

			allAPIs = append(allAPIs, apisRes.APIs...)
			allAPIStatuses = append(allAPIStatuses, apisRes.Statuses...)
			allMetrics = append(allMetrics, apisRes.AllMetrics...)
		} else {
			errorEnvNames = append(errorEnvNames, env.Name)
		}
	}

	t := apiTable(allAPIs, allAPIStatuses, allMetrics, allEnvs)

	if strset.New(allEnvs...).IsEqual(strset.New(types.LocalProviderType.String())) {
		hideReplicaCountColumns(&t)
	}

	out := t.MustFormat()

	if len(errorEnvNames) == 1 {
		out += "\n" + fmt.Sprintf("failed to fetch apis from %s environment; run `cortex get --env %s` to get the complete error message\n", s.UserStr(errorEnvNames[0]), errorEnvNames[0])
	} else if len(errorEnvNames) > 1 {
		out += "\n" + fmt.Sprintf("failed to fetch apis from %s environments; run `cortex get --env ENV_NAME` to get the complete error message\n", s.UserStrsAnd(errorEnvNames))
	}

	mismatchedAPIMessage, err := getLocalVersionMismatchedAPIsMessage()
	if err == nil {
		out += "\n" + mismatchedAPIMessage
	}

	return out, nil
}

func hideReplicaCountColumns(t *table.Table) {
	t.FindHeaderByTitle(_titleUpToDate).Hidden = true
	t.FindHeaderByTitle(_titleStale).Hidden = true
	t.FindHeaderByTitle(_titleRequested).Hidden = true
	t.FindHeaderByTitle(_titleFailed).Hidden = true
}

func getAPIs(env cliconfig.Environment, printEnv bool) (string, error) {
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

	if len(apisRes.APIs) == 0 {
		return console.Bold("no apis are deployed"), nil
	}

	envNames := []string{}
	for range apisRes.APIs {
		envNames = append(envNames, env.Name)
	}

	t := apiTable(apisRes.APIs, apisRes.Statuses, apisRes.AllMetrics, envNames)

	t.FindHeaderByTitle(_titleEnvironment).Hidden = true

	out := t.MustFormat()

	if env.Provider == types.LocalProviderType {
		hideReplicaCountColumns(&t)
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
	var apiRes schema.GetAPIResponse
	var err error
	if env.Provider == types.AWSProviderType {
		apiRes, err = cluster.GetAPI(MustGetOperatorConfig(env.Name), apiName)
		if err != nil {
			// note: if modifying this string, search the codebase for it and change all occurrences
			if strings.HasSuffix(errors.Message(err), "is not deployed") {
				return console.Bold(errors.Message(err)), nil
			}
			return "", err
		}
	} else {
		apiRes, err = local.GetAPI(apiName)
		if err != nil {
			// note: if modifying this string, search the codebase for it and change all occurrences
			if strings.HasSuffix(errors.Message(err), "is not deployed") {
				return console.Bold(errors.Message(err)), nil
			}
			return "", err
		}
	}

	var out string

	t := apiTable([]spec.API{apiRes.API}, []status.Status{apiRes.Status}, []metrics.Metrics{apiRes.Metrics}, []string{env.Name})
	t.FindHeaderByTitle(_titleEnvironment).Hidden = true
	t.FindHeaderByTitle(_titleAPI).Hidden = true

	out += t.MustFormat()

	api := apiRes.API

	if env.Provider != types.LocalProviderType && api.Tracker != nil {
		switch api.Tracker.ModelType {
		case userconfig.ClassificationModelType:
			out += "\n" + classificationMetricsStr(&apiRes.Metrics)
		case userconfig.RegressionModelType:
			out += "\n" + regressionMetricsStr(&apiRes.Metrics)
		}
	}

	apiEndpoint := apiRes.BaseURL
	if env.Provider == types.AWSProviderType {
		apiEndpoint = strings.Replace(urls.Join(apiRes.BaseURL, *api.Endpoint), "https://", "http://", 1)
	}
	out += "\n" + console.Bold("endpoint: ") + apiEndpoint

	out += fmt.Sprintf("\n%s curl %s -X POST -H \"Content-Type: application/json\" -d @sample.json\n", console.Bold("curl:"), apiEndpoint)

	if api.Predictor.Type == userconfig.TensorFlowPredictorType || api.Predictor.Type == userconfig.ONNXPredictorType {
		out += "\n" + describeModelInput(&apiRes.Status, apiEndpoint)
	}

	out += titleStr("configuration") + strings.TrimSpace(api.UserStr(env.Provider))

	return out, nil
}

func apiTable(apis []spec.API, statuses []status.Status, allMetrics []metrics.Metrics, envNames []string) table.Table {
	rows := make([][]interface{}, 0, len(apis))

	var totalFailed int32
	var totalStale int32
	var total4XX int
	var total5XX int

	for i, api := range apis {
		metrics := allMetrics[i]
		status := statuses[i]
		lastUpdated := time.Unix(api.LastUpdated, 0)
		rows = append(rows, []interface{}{
			envNames[i],
			api.Name,
			status.Message(),
			status.Updated.Ready,
			status.Stale.Ready,
			status.Requested,
			status.Updated.TotalFailed(),
			libtime.SinceStr(&lastUpdated),
			latencyStr(&metrics),
			code2XXStr(&metrics),
			code4XXStr(&metrics),
			code5XXStr(&metrics),
		})

		totalFailed += status.Updated.TotalFailed()
		totalStale += status.Stale.Ready

		if metrics.NetworkStats != nil {
			total4XX += metrics.NetworkStats.Code4XX
			total5XX += metrics.NetworkStats.Code5XX
		}
	}

	return table.Table{
		Headers: []table.Header{
			{Title: _titleEnvironment},
			{Title: _titleAPI},
			{Title: _titleStatus},
			{Title: _titleUpToDate},
			{Title: _titleStale, Hidden: totalStale == 0},
			{Title: _titleRequested},
			{Title: _titleFailed, Hidden: totalFailed == 0},
			{Title: _titleLastupdated},
			{Title: _titleAvgRequest},
			{Title: _title2XX},
			{Title: _title4XX, Hidden: total4XX == 0},
			{Title: _title5XX, Hidden: total5XX == 0},
		},
		Rows: rows,
	}
}

func latencyStr(metrics *metrics.Metrics) string {
	if metrics.NetworkStats == nil || metrics.NetworkStats.Latency == nil {
		return "-"
	}
	if *metrics.NetworkStats.Latency < 1000 {
		return fmt.Sprintf("%.6g ms", *metrics.NetworkStats.Latency)
	}
	return fmt.Sprintf("%.6g s", (*metrics.NetworkStats.Latency)/1000)
}

func code2XXStr(metrics *metrics.Metrics) string {
	if metrics.NetworkStats == nil || metrics.NetworkStats.Code2XX == 0 {
		return "-"
	}
	return s.Int(metrics.NetworkStats.Code2XX)
}

func code4XXStr(metrics *metrics.Metrics) string {
	if metrics.NetworkStats == nil || metrics.NetworkStats.Code4XX == 0 {
		return "-"
	}
	return s.Int(metrics.NetworkStats.Code4XX)
}

func code5XXStr(metrics *metrics.Metrics) string {
	if metrics.NetworkStats == nil || metrics.NetworkStats.Code5XX == 0 {
		return "-"
	}
	return s.Int(metrics.NetworkStats.Code5XX)
}

func regressionMetricsStr(metrics *metrics.Metrics) string {
	minStr := "-"
	maxStr := "-"
	avgStr := "-"

	if metrics.RegressionStats != nil {
		if metrics.RegressionStats.Min != nil {
			minStr = fmt.Sprintf("%.9g", *metrics.RegressionStats.Min)
		}

		if metrics.RegressionStats.Max != nil {
			maxStr = fmt.Sprintf("%.9g", *metrics.RegressionStats.Max)
		}

		if metrics.RegressionStats.Avg != nil {
			avgStr = fmt.Sprintf("%.9g", *metrics.RegressionStats.Avg)
		}
	}

	t := table.Table{
		Headers: []table.Header{
			{Title: "min", MaxWidth: 10},
			{Title: "max", MaxWidth: 10},
			{Title: "avg", MaxWidth: 10},
		},
		Rows: [][]interface{}{{minStr, maxStr, avgStr}},
	}

	return t.MustFormat()
}

func classificationMetricsStr(metrics *metrics.Metrics) string {
	classList := make([]string, 0, len(metrics.ClassDistribution))
	for inputName := range metrics.ClassDistribution {
		classList = append(classList, inputName)
	}
	sort.Strings(classList)

	rows := make([][]interface{}, len(classList))
	for rowNum, className := range classList {
		rows[rowNum] = []interface{}{
			className,
			metrics.ClassDistribution[className],
		}
	}

	if len(classList) == 0 {
		rows = append(rows, []interface{}{
			"-",
			"-",
		})
	}

	t := table.Table{
		Headers: []table.Header{
			{Title: "class", MaxWidth: 40},
			{Title: "count", MaxWidth: 20},
		},
		Rows: rows,
	}

	out := t.MustFormat()

	if len(classList) == consts.MaxClassesPerTrackerRequest {
		out += fmt.Sprintf("\nlisting at most %d classes, the complete list can be found in your cloudwatch dashboard\n", consts.MaxClassesPerTrackerRequest)
	}
	return out
}

func describeModelInput(status *status.Status, apiEndpoint string) string {
	if status.Updated.Ready+status.Stale.Ready == 0 {
		return "the model's input schema will be available when the api is live\n"
	}

	apiSummary, err := getAPISummary(apiEndpoint)
	if err != nil {
		return "error retrieving the model's input schema: " + errors.Message(err) + "\n"
	}

	rows := make([][]interface{}, len(apiSummary.ModelSignature))
	rowNum := 0
	for inputName, featureSignature := range apiSummary.ModelSignature {
		shapeStr := make([]string, len(featureSignature.Shape))
		for idx, dim := range featureSignature.Shape {
			shapeStr[idx] = s.ObjFlatNoQuotes(dim)
		}
		rows[rowNum] = []interface{}{
			inputName,
			featureSignature.Type,
			"(" + strings.Join(shapeStr, ", ") + ")",
		}
		rowNum++
	}

	t := table.Table{
		Headers: []table.Header{
			{Title: "model input", MaxWidth: 32},
			{Title: "type", MaxWidth: 10},
			{Title: "shape", MaxWidth: 20},
		},
		Rows: rows,
	}

	return t.MustFormat()
}

func makeRequest(request *http.Request) (http.Header, []byte, error) {
	client := http.Client{
		Timeout: 600 * time.Second,
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
		},
	}

	response, err := client.Do(request)
	if err != nil {
		return nil, nil, errors.Wrap(err, errStrFailedToConnect(*request.URL))
	}
	defer response.Body.Close()

	if response.StatusCode != 200 {
		bodyBytes, err := ioutil.ReadAll(response.Body)
		if err != nil {
			return nil, nil, errors.Wrap(err, _errStrRead)
		}
		return nil, nil, ErrorResponseUnknown(string(bodyBytes), response.StatusCode)
	}

	bodyBytes, err := ioutil.ReadAll(response.Body)
	if err != nil {
		return nil, nil, errors.Wrap(err, _errStrRead)
	}
	return response.Header, bodyBytes, nil
}

func getAPISummary(apiEndpoint string) (*schema.APISummary, error) {
	req, err := http.NewRequest("GET", apiEndpoint, nil)
	if err != nil {
		return nil, errors.Wrap(err, "unable to request api summary")
	}
	req.Header.Set("Content-Type", "application/json")
	_, response, err := makeRequest(req)
	if err != nil {
		return nil, err
	}

	var apiSummary schema.APISummary
	err = json.DecodeWithNumber(response, &apiSummary)
	if err != nil {
		return nil, errors.Wrap(err, "unable to parse api summary response")
	}

	for _, featureSignature := range apiSummary.ModelSignature {
		featureSignature.Shape = cast.JSONNumbers(featureSignature.Shape)
	}

	return &apiSummary, nil
}

func titleStr(title string) string {
	return "\n" + console.Bold(title) + "\n"
}
