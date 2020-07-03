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

	"github.com/cortexlabs/cortex/cli/types/cliconfig"
	"github.com/cortexlabs/cortex/pkg/consts"
	"github.com/cortexlabs/cortex/pkg/lib/cast"
	"github.com/cortexlabs/cortex/pkg/lib/console"
	"github.com/cortexlabs/cortex/pkg/lib/errors"
	"github.com/cortexlabs/cortex/pkg/lib/json"
	s "github.com/cortexlabs/cortex/pkg/lib/strings"
	"github.com/cortexlabs/cortex/pkg/lib/table"
	libtime "github.com/cortexlabs/cortex/pkg/lib/time"
	"github.com/cortexlabs/cortex/pkg/lib/urls"
	"github.com/cortexlabs/cortex/pkg/operator/schema"
	"github.com/cortexlabs/cortex/pkg/types"
	"github.com/cortexlabs/cortex/pkg/types/metrics"
	"github.com/cortexlabs/cortex/pkg/types/status"
	"github.com/cortexlabs/cortex/pkg/types/userconfig"
)

func syncAPITable(syncAPI *schema.SyncAPI, env cliconfig.Environment) (string, error) {
	var out string

	t := syncAPIsTable([]schema.SyncAPI{*syncAPI}, []string{env.Name})
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

	if syncAPI.Spec.Predictor.Type == userconfig.TensorFlowPredictorType || syncAPI.Spec.Predictor.Type == userconfig.ONNXPredictorType {
		out += "\n" + describeModelInput(&syncAPI.Status, apiEndpoint)
	}

	out += titleStr("configuration") + strings.TrimSpace(syncAPI.Spec.UserStr(env.Provider))

	return out, nil
}

func syncAPIsTable(syncAPIs []schema.SyncAPI, envNames []string) table.Table {
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

func describeModelInput(status *status.Status, apiEndpoint string) string {
	if status.Updated.Ready+status.Stale.Ready == 0 {
		return "the model's input schema will be available when the api is live\n"
	}

	apiSummary, err := getAPISummary(apiEndpoint)
	if err != nil {
		return "error retrieving the model's input schema: " + errors.Message(err) + "\n"
	}

	numRows := 0
	for _, inputSignatures := range apiSummary.ModelSignatures {
		numRows += len(inputSignatures)
	}

	usesDefaultModel := false
	rows := make([][]interface{}, numRows)
	rowNum := 0
	for modelName, inputSignatures := range apiSummary.ModelSignatures {
		for inputName, inputSignature := range inputSignatures {
			shapeStr := make([]string, len(inputSignature.Shape))
			for idx, dim := range inputSignature.Shape {
				shapeStr[idx] = s.ObjFlatNoQuotes(dim)
			}
			rows[rowNum] = []interface{}{
				modelName,
				inputName,
				inputSignature.Type,
				"(" + strings.Join(shapeStr, ", ") + ")",
			}
			rowNum++
		}
		if modelName == consts.SingleModelName {
			usesDefaultModel = true
		}
	}

	inputTitle := "input"
	if usesDefaultModel {
		inputTitle = "model input"
	}
	t := table.Table{
		Headers: []table.Header{
			{Title: "model name", MaxWidth: 32, Hidden: usesDefaultModel},
			{Title: inputTitle, MaxWidth: 32},
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

	for _, inputSignatures := range apiSummary.ModelSignatures {
		for _, inputSignature := range inputSignatures {
			inputSignature.Shape = cast.JSONNumbers(inputSignature.Shape)
		}
	}

	return &apiSummary, nil
}
