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
	"crypto/tls"
	"fmt"
	"io/ioutil"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/cortexlabs/cortex/cli/types/cliconfig"
	"github.com/cortexlabs/cortex/pkg/consts"
	"github.com/cortexlabs/cortex/pkg/lib/console"
	"github.com/cortexlabs/cortex/pkg/lib/errors"
	"github.com/cortexlabs/cortex/pkg/lib/json"
	s "github.com/cortexlabs/cortex/pkg/lib/strings"
	"github.com/cortexlabs/cortex/pkg/lib/table"
	libtime "github.com/cortexlabs/cortex/pkg/lib/time"
	"github.com/cortexlabs/cortex/pkg/operator/schema"
	"github.com/cortexlabs/cortex/pkg/types/metrics"
	"github.com/cortexlabs/cortex/pkg/types/status"
	"github.com/cortexlabs/cortex/pkg/types/userconfig"
)

func realtimeAPITable(realtimeAPI schema.APIResponse, env cliconfig.Environment) (string, error) {
	var out string

	t := realtimeAPIsTable([]schema.APIResponse{realtimeAPI}, []string{env.Name})
	t.FindHeaderByTitle(_titleEnvironment).Hidden = true
	t.FindHeaderByTitle(_titleRealtimeAPI).Hidden = true

	out += t.MustFormat()

	if realtimeAPI.DashboardURL != nil && *realtimeAPI.DashboardURL != "" {
		out += "\n" + console.Bold("metrics dashboard: ") + *realtimeAPI.DashboardURL + "\n"
	}

	out += "\n" + console.Bold("endpoint: ") + realtimeAPI.Endpoint + "\n"

	if !(realtimeAPI.Spec.Predictor.Type == userconfig.PythonPredictorType && realtimeAPI.Spec.Predictor.MultiModelReloading == nil) {
		out += "\n" + describeModelInput(realtimeAPI.Status, realtimeAPI.Spec.Predictor, realtimeAPI.Endpoint)
	}

	out += "\n" + apiHistoryTable(realtimeAPI.APIVersions)

	if !_flagVerbose {
		return out, nil
	}

	out += titleStr("configuration") + strings.TrimSpace(realtimeAPI.Spec.UserStr(env.Provider))

	return out, nil
}

func realtimeAPIsTable(realtimeAPIs []schema.APIResponse, envNames []string) table.Table {
	rows := make([][]interface{}, 0, len(realtimeAPIs))

	var totalFailed int32
	var totalStale int32
	var total4XX int
	var total5XX int

	for i, realtimeAPI := range realtimeAPIs {
		lastUpdated := time.Unix(realtimeAPI.Spec.LastUpdated, 0)
		rows = append(rows, []interface{}{
			envNames[i],
			realtimeAPI.Spec.Name,
			realtimeAPI.Status.Message(),
			realtimeAPI.Status.Updated.Ready,
			realtimeAPI.Status.Stale.Ready,
			realtimeAPI.Status.Requested,
			realtimeAPI.Status.Updated.TotalFailed(),
			libtime.SinceStr(&lastUpdated),
			latencyStr(realtimeAPI.Metrics),
			code2XXStr(realtimeAPI.Metrics),
			code4XXStr(realtimeAPI.Metrics),
			code5XXStr(realtimeAPI.Metrics),
		})

		totalFailed += realtimeAPI.Status.Updated.TotalFailed()
		totalStale += realtimeAPI.Status.Stale.Ready

		if realtimeAPI.Metrics.NetworkStats != nil {
			total4XX += realtimeAPI.Metrics.NetworkStats.Code4XX
			total5XX += realtimeAPI.Metrics.NetworkStats.Code5XX
		}
	}

	return table.Table{
		Headers: []table.Header{
			{Title: _titleEnvironment},
			{Title: _titleRealtimeAPI},
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

func describeModelInput(status *status.Status, predictor *userconfig.Predictor, apiEndpoint string) string {
	if status.Updated.Ready+status.Stale.Ready == 0 {
		return "the models' metadata schema will be available when the api is live\n"
	}

	cachingEnabled := predictor.Models != nil && predictor.Models.CacheSize != nil && predictor.Models.DiskCacheSize != nil
	if predictor.Type == userconfig.TensorFlowPredictorType && !cachingEnabled {
		apiTFLiveReloadingSummary, err := getAPITFLiveReloadingSummary(apiEndpoint)
		if err != nil {
			return "error retrieving the models' metadata schema: " + errors.Message(err) + "\n"
		}
		t, err := parseAPITFLiveReloadingSummary(apiTFLiveReloadingSummary)
		if err != nil {
			return "error retrieving the model's input schema: " + errors.Message(err) + "\n"
		}
		return t
	}

	apiModelSummary, err := getAPIModelSummary(apiEndpoint)
	if err != nil {
		return "error retrieving the models' metadata schema: " + errors.Message(err) + "\n"
	}
	t, err := parseAPIModelSummary(apiModelSummary)
	if err != nil {
		return "error retrieving the models' metadata schema: " + errors.Message(err) + "\n"
	}
	return t
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

func getAPIModelSummary(apiEndpoint string) (*schema.APIModelSummary, error) {
	req, err := http.NewRequest("GET", apiEndpoint, nil)
	if err != nil {
		return nil, errors.Wrap(err, "unable to request api summary")
	}
	req.Header.Set("Content-Type", "application/json")
	_, response, err := makeRequest(req)
	if err != nil {
		return nil, err
	}

	var apiModelSummary schema.APIModelSummary
	err = json.DecodeWithNumber(response, &apiModelSummary)
	if err != nil {
		return nil, errors.Wrap(err, "unable to parse api summary response")
	}
	return &apiModelSummary, nil
}

func getAPITFLiveReloadingSummary(apiEndpoint string) (*schema.APITFLiveReloadingSummary, error) {
	req, err := http.NewRequest("GET", apiEndpoint, nil)
	if err != nil {
		return nil, errors.Wrap(err, "unable to request api summary")
	}
	req.Header.Set("Content-Type", "application/json")
	_, response, err := makeRequest(req)
	if err != nil {
		return nil, err
	}

	var apiTFLiveReloadingSummary schema.APITFLiveReloadingSummary
	err = json.DecodeWithNumber(response, &apiTFLiveReloadingSummary)
	if err != nil {
		return nil, errors.Wrap(err, "unable to parse api summary response")
	}
	return &apiTFLiveReloadingSummary, nil
}

func parseAPIModelSummary(summary *schema.APIModelSummary) (string, error) {
	rows := make([][]interface{}, 0)

	for modelName, modelMetadata := range summary.ModelMetadata {
		latestVersion := int64(0)
		for _, version := range modelMetadata.Versions {
			v, err := strconv.ParseInt(version, 10, 64)
			if err != nil {
				return "", err
			}
			if v > latestVersion {
				latestVersion = v
			}
		}
		latestStrVersion := strconv.FormatInt(latestVersion, 10)

		for idx, version := range modelMetadata.Versions {
			var latestTag string
			if latestStrVersion == version {
				latestTag = " (latest)"
			}

			timestamp := modelMetadata.Timestamps[idx]
			date := time.Unix(timestamp, 0)

			rows = append(rows, []interface{}{
				modelName,
				version + latestTag,
				date.Format(_timeFormat),
			})
		}
	}

	_, usesCortexDefaultModelName := summary.ModelMetadata[consts.SingleModelName]

	t := table.Table{
		Headers: []table.Header{
			{
				Title:    "model name",
				MaxWidth: 32,
				Hidden:   usesCortexDefaultModelName,
			},
			{
				Title:    "model version",
				MaxWidth: 25,
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
	latestVersions := make(map[string]int64)

	numRows := 0
	models := make(map[string]schema.GenericModelMetadata, 0)
	for modelID, modelMetadata := range summary.ModelMetadata {
		timestamp := modelMetadata.Timestamp
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
		if _, ok := latestVersions[modelName]; !ok {
			latestVersions[modelName] = modelVersion
		} else if modelVersion > latestVersions[modelName] {
			latestVersions[modelName] = modelVersion
		}
		numRows += len(modelMetadata.InputSignatures)
	}

	rows := make([][]interface{}, 0, numRows)
	for modelName, model := range models {
		latestVersion := latestVersions[modelName]

		for _, modelVersion := range model.Versions {
			modelID := fmt.Sprintf("%s-%s", modelName, modelVersion)

			inputSignatures := summary.ModelMetadata[modelID].InputSignatures
			timestamp := summary.ModelMetadata[modelID].Timestamp
			versionInt, err := strconv.ParseInt(modelVersion, 10, 64)
			if err != nil {
				return "", err
			}

			var applicableTags string
			if versionInt == latestVersion {
				applicableTags = " (latest)"
			}

			date := time.Unix(timestamp, 0)

			for inputName, inputSignature := range inputSignatures {
				shapeStr := make([]string, len(inputSignature.Shape))
				for idx, dim := range inputSignature.Shape {
					shapeStr[idx] = s.ObjFlatNoQuotes(dim)
				}
				shapeRowEntry := ""
				if len(shapeStr) == 1 && shapeStr[0] == "scalar" {
					shapeRowEntry = "scalar"
				} else if len(shapeStr) == 1 && shapeStr[0] == "unknown" {
					shapeRowEntry = "unknown"
				} else {
					shapeRowEntry = "(" + strings.Join(shapeStr, ", ") + ")"
				}
				rows = append(rows, []interface{}{
					modelName,
					modelVersion + applicableTags,
					inputName,
					inputSignature.Type,
					shapeRowEntry,
					date.Format(_timeFormat),
				})
			}
		}
	}

	usesCortexDefaultModelName := false
	for modelID := range summary.ModelMetadata {
		modelName, _, err := getModelFromModelID(modelID)
		if err != nil {
			return "", err
		}
		if modelName == consts.SingleModelName {
			usesCortexDefaultModelName = true
			break
		}
	}

	t := table.Table{
		Headers: []table.Header{
			{
				Title:    "model name",
				MaxWidth: 32,
				Hidden:   usesCortexDefaultModelName,
			},
			{
				Title:    "model version",
				MaxWidth: 25,
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
				Title:    "edit time",
				MaxWidth: 32,
			},
		},
		Rows: rows,
	}

	return t.MustFormat(), nil
}
