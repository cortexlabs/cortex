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
	"strconv"
	"strings"
	"time"

	"github.com/cortexlabs/cortex/cli/cluster"
	"github.com/cortexlabs/cortex/cli/local"
	"github.com/cortexlabs/cortex/cli/types/cliconfig"
	"github.com/cortexlabs/cortex/pkg/consts"
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
	"github.com/cortexlabs/cortex/pkg/types/status"
	"github.com/cortexlabs/cortex/pkg/types/userconfig"
	"github.com/spf13/cobra"
)

const (
	_titleEnvironment = "env"
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

	var allSyncAPIs []schema.SyncAPI
	var allEnvs []string
	errorsMap := map[string]error{}
	for _, env := range cliConfig.Environments {
		var apisRes schema.GetAPIsResponse
		var err error
		if env.Provider == types.AWSProviderType {
			apisRes, err = cluster.GetAPIs(MustGetOperatorConfig(env.Name))
		} else {
			apisRes, err = local.GetAPIs()
		}

		if err == nil {
			for range apisRes.SyncAPIs {
				allEnvs = append(allEnvs, env.Name)
			}

			allSyncAPIs = append(allSyncAPIs, apisRes.SyncAPIs...)
		} else {
			errorsMap[env.Name] = err
		}
	}

	out := ""

	if len(allSyncAPIs) == 0 {
		if len(errorsMap) == 1 {
			// Print the error if there is just one
			exit.Error(errors.FirstErrorInMap(errorsMap))
		}
		// if all envs errored, skip it "no apis are deployed" since it's misleading
		if len(errorsMap) != len(cliConfig.Environments) {
			out += console.Bold("no apis are deployed") + "\n"
		}
	} else {
		t := apiTable(allSyncAPIs, allEnvs)

		if strset.New(allEnvs...).IsEqual(strset.New(types.LocalProviderType.String())) {
			hideReplicaCountColumns(&t)
		}

		out += t.MustFormat()
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

	if len(apisRes.SyncAPIs) == 0 {
		return console.Bold("no apis are deployed"), nil
	}

	envNames := []string{}
	for range apisRes.SyncAPIs {
		envNames = append(envNames, env.Name)
	}

	t := apiTable(apisRes.SyncAPIs, envNames)

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
	return syncAPITable(apiRes.SyncAPI, env)
}

func syncAPITable(syncAPI *schema.SyncAPI, env cliconfig.Environment) (string, error) {
	var out string

	t := apiTable([]schema.SyncAPI{*syncAPI}, []string{env.Name})
	t.FindHeaderByTitle(_titleEnvironment).Hidden = true
	t.FindHeaderByTitle(_titleAPI).Hidden = true

	out += t.MustFormat()

	if env.Provider != types.LocalProviderType && syncAPI.Spec.Monitoring != nil {
		switch syncAPI.Spec.Monitoring.ModelType {
		case userconfig.ClassificationModelType:
			out += "\n" + classificationMetricsStr(&syncAPI.Metrics)
		case userconfig.RegressionModelType:
			out += "\n" + regressionMetricsStr(&syncAPI.Metrics)
		}
	}

	apiEndpoint := syncAPI.BaseURL
	if env.Provider == types.AWSProviderType {
		apiEndpoint = urls.Join(syncAPI.BaseURL, *syncAPI.Spec.Networking.Endpoint)
		if syncAPI.Spec.Networking.APIGateway == userconfig.NoneAPIGatewayType {
			apiEndpoint = strings.Replace(apiEndpoint, "https://", "http://", 1)
		}
	}

	if syncAPI.DashboardURL != "" {
		out += "\n" + console.Bold("metrics dashboard: ") + syncAPI.DashboardURL + "\n"
	}

	out += "\n" + console.Bold("endpoint: ") + apiEndpoint

	out += fmt.Sprintf("\n%s curl %s -X POST -H \"Content-Type: application/json\" -d @sample.json\n", console.Bold("curl:"), apiEndpoint)

	out += "\n" + describeModelInput(&syncAPI.Status, syncAPI.Spec.Predictor, apiEndpoint)

	out += titleStr("configuration") + strings.TrimSpace(syncAPI.Spec.UserStr(env.Provider))

	return out, nil
}

func apiTable(syncAPIs []schema.SyncAPI, envNames []string) table.Table {
	rows := make([][]interface{}, 0, len(syncAPIs))

	var totalFailed int32
	var totalStale int32
	var total4XX int
	var total5XX int

	for i, syncAPI := range syncAPIs {
		lastUpdated := time.Unix(syncAPI.Spec.LastUpdated, 0)
		rows = append(rows, []interface{}{
			envNames[i],
			syncAPI.Spec.Name,
			syncAPI.Status.Message(),
			syncAPI.Status.Updated.Ready,
			syncAPI.Status.Stale.Ready,
			syncAPI.Status.Requested,
			syncAPI.Status.Updated.TotalFailed(),
			libtime.SinceStr(&lastUpdated),
			latencyStr(&syncAPI.Metrics),
			code2XXStr(&syncAPI.Metrics),
			code4XXStr(&syncAPI.Metrics),
			code5XXStr(&syncAPI.Metrics),
		})

		totalFailed += syncAPI.Status.Updated.TotalFailed()
		totalStale += syncAPI.Status.Stale.Ready

		if syncAPI.Metrics.NetworkStats != nil {
			total4XX += syncAPI.Metrics.NetworkStats.Code4XX
			total5XX += syncAPI.Metrics.NetworkStats.Code5XX
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

	if len(classList) == consts.MaxClassesPerMonitoringRequest {
		out += fmt.Sprintf("\nlisting at most %d classes, the complete list can be found in your cloudwatch dashboard\n", consts.MaxClassesPerMonitoringRequest)
	}
	return out
}

func describeModelInput(status *status.Status, predictor *userconfig.Predictor, apiEndpoint string) string {
	if status.Updated.Ready+status.Stale.Ready == 0 {
		return "the models' metadata schema will be available when the api is live\n"
	}

	apiModelSummary, apiTFLiveReloadingSummary, err := getAPISummary(apiEndpoint, predictor)
	if err != nil {
		return "error retrieving the models' metadata schema: " + errors.Message(err) + "\n"
	}

	if apiModelSummary != nil {
		t, err := parseAPIModelSummary(apiModelSummary)
		if err != nil {
			return "error retrieving the models' metadata schema: " + errors.Message(err) + "\n"
		}
		return t
	}
	t, err := parseAPITFLiveReloadingSummary(apiTFLiveReloadingSummary)
	if err != nil {
		return "error retrieving the model's input schema: " + errors.Message(err) + "\n"
	}
	return t
}

func parseAPIModelSummary(summary *schema.APIModelSummary) (string, error) {
	numRows := 0
	tags := make(map[string][]int64)
	for modelName := range summary.ModelMetadata {
		numRows += len(summary.ModelMetadata[modelName].Versions)
		highestVersion := int64(0)
		latestTimestamp := int64(0)
		for i, version := range summary.ModelMetadata[modelName].Versions {
			v, err := strconv.ParseInt(version, 10, 64)
			if err != nil {
				return "", err
			}
			if v > highestVersion {
				highestVersion = v
			}
			timestamp := summary.ModelMetadata[modelName].Timestamps[i]
			if timestamp > latestTimestamp {
				latestTimestamp = timestamp
			}
		}
		tags[modelName] = []int64{highestVersion, latestTimestamp}
	}

	rows := make([][]interface{}, numRows)
	rowNum := 0
	for modelName := range summary.ModelMetadata {
		highestVersion := strconv.FormatInt(tags[modelName][0], 10)
		latestTimestamp := tags[modelName][1]

		for idx, version := range summary.ModelMetadata[modelName].Versions {
			timestamp := summary.ModelMetadata[modelName].Timestamps[idx]

			applicableTags := "-"
			if highestVersion == version && latestTimestamp == timestamp {
				applicableTags = "latest/highest"
			} else if highestVersion == version {
				applicableTags = "highest"
			} else if latestTimestamp == timestamp {
				applicableTags = "latest"
			}

			date := time.Unix(timestamp, 0)

			rows[rowNum] = []interface{}{
				modelName,
				summary.ModelMetadata[modelName].Versions[idx],
				applicableTags,
				date.Format("02 Jan 06 15:04:05 MST"),
			}
			rowNum++
		}
	}

	t := table.Table{
		Headers: []table.Header{
			{
				Title:    "model name",
				MaxWidth: 32,
			},
			{
				Title:    "model version",
				MaxWidth: 16,
			},
			{
				Title:    "version tags",
				MaxWidth: 14,
			},
			{
				Title:    "edit time",
				MaxWidth: 32,
			},
		},
		Rows: rows,
	}

	return t.MustFormat(), nil
}

func parseAPITFLiveReloadingSummary(summary *schema.APITFLiveReloadingSummary) (string, error) {
	highestVersions := make(map[string]int64)
	latestTimestamps := make(map[string]int64)

	numRows := 0
	models := make(map[string]schema.GenericModelMetadata, 0)
	for modelID := range summary.ModelMetadata {
		timestamp := summary.ModelMetadata[modelID].Timestamp
		modelName, modelVersion, err := getModelFromModelID(modelID)
		if err != nil {
			return "", err
		}
		if _, ok := models[modelName]; !ok {
			models[modelName] = schema.GenericModelMetadata{
				Versions:   []string{strconv.FormatInt(modelVersion, 10)},
				Timestamps: []int64{timestamp},
			}
		} else {
			model := models[modelName]
			model.Versions = append(model.Versions, strconv.FormatInt(modelVersion, 10))
			model.Timestamps = append(model.Timestamps, timestamp)
			models[modelName] = model
		}
		if _, ok := highestVersions[modelName]; !ok {
			highestVersions[modelName] = modelVersion
		} else {
			if modelVersion > highestVersions[modelName] {
				highestVersions[modelName] = modelVersion
			}
		}
		if _, ok := latestTimestamps[modelName]; !ok {
			latestTimestamps[modelName] = timestamp
		} else {
			if timestamp > latestTimestamps[modelName] {
				latestTimestamps[modelName] = timestamp
			}
		}
		numRows += len(summary.ModelMetadata[modelID].InputSignatures)
	}

	rows := make([][]interface{}, numRows)
	rowNum := 0
	for modelName := range models {
		highestVersion := highestVersions[modelName]
		latestTimestamp := latestTimestamps[modelName]

		for _, modelVersion := range models[modelName].Versions {
			modelID := fmt.Sprintf("%s-%s", modelName, modelVersion)

			inputSignatures := summary.ModelMetadata[modelID].InputSignatures
			timestamp := summary.ModelMetadata[modelID].Timestamp
			versionInt, err := strconv.ParseInt(modelVersion, 10, 64)
			if err != nil {
				return "", err
			}

			applicableTags := "-"
			if highestVersion == versionInt && latestTimestamp == timestamp {
				applicableTags = "latest/highest"
			} else if highestVersion == versionInt {
				applicableTags = "highest"
			} else if latestTimestamp == timestamp {
				applicableTags = "latest"
			}

			date := time.Unix(timestamp, 0)

			for inputName, inputSignature := range inputSignatures {
				shapeStr := make([]string, len(inputSignature.Shape))
				for idx, dim := range inputSignature.Shape {
					shapeStr[idx] = s.ObjFlatNoQuotes(dim)
				}
				rows[rowNum] = []interface{}{
					modelName,
					modelVersion,
					inputName,
					inputSignature.Type,
					"(" + strings.Join(shapeStr, ", ") + ")",
					applicableTags,
					date.Format("02 Jan 06 15:04:05 MST"),
				}
				rowNum++
			}
		}
	}

	t := table.Table{
		Headers: []table.Header{
			{
				Title:    "model name",
				MaxWidth: 32,
			},
			{
				Title:    "model version",
				MaxWidth: 16,
			},
			{
				Title:    "model input",
				MaxWidth: 32,
			},
			{
				Title:    "type",
				MaxWidth: 10,
			},
			{
				Title:    "shape",
				MaxWidth: 20,
			},
			{
				Title:    "version tags",
				MaxWidth: 14,
			},
			{
				Title:    "edit time",
				MaxWidth: 32,
			},
		},
		Rows: rows,
	}

	return t.MustFormat(), nil
}

func getModelFromModelID(modelID string) (modelName string, modelVersion int64, err error) {
	splitIndex := strings.LastIndex(modelID, "-")
	modelName = modelID[:splitIndex]
	modelVersion, err = strconv.ParseInt(modelID[splitIndex+1:], 10, 64)
	return
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

func getAPISummary(apiEndpoint string, predictor *userconfig.Predictor) (*schema.APIModelSummary, *schema.APITFLiveReloadingSummary, error) {
	req, err := http.NewRequest("GET", apiEndpoint, nil)
	if err != nil {
		return nil, nil, errors.Wrap(err, "unable to request api summary")
	}
	req.Header.Set("Content-Type", "application/json")
	_, response, err := makeRequest(req)
	if err != nil {
		return nil, nil, err
	}

	var apiModelSummary schema.APIModelSummary
	var apiTFLiveReloadingSummary schema.APITFLiveReloadingSummary

	cachingEnabled := predictor.Models != nil && predictor.Models.CacheSize != nil && predictor.Models.DiskCacheSize != nil
	if predictor.Type == userconfig.TensorFlowPredictorType && !cachingEnabled {
		err = json.DecodeWithNumber(response, &apiTFLiveReloadingSummary)
		if err != nil {
			return nil, nil, errors.Wrap(err, "unable to parse api summary response")
		}
		return nil, &apiTFLiveReloadingSummary, nil
	}

	err = json.DecodeWithNumber(response, &apiModelSummary)
	if err != nil {
		return nil, nil, errors.Wrap(err, "unable to parse api summary response")
	}
	return &apiModelSummary, nil, nil
}

func titleStr(title string) string {
	return "\n" + console.Bold(title) + "\n"
}
