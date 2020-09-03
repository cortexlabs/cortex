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
	"github.com/cortexlabs/cortex/pkg/operator/schema"
	"github.com/cortexlabs/cortex/pkg/types"
	"github.com/cortexlabs/cortex/pkg/types/metrics"
	"github.com/cortexlabs/cortex/pkg/types/status"
	"github.com/cortexlabs/cortex/pkg/types/userconfig"
)

func realtimeAPITable(realtimeAPI *schema.RealtimeAPI, env cliconfig.Environment) (string, error) {
	var out string

	t := realtimeAPIsTable([]schema.RealtimeAPI{*realtimeAPI}, []string{env.Name})
	t.FindHeaderByTitle(_titleEnvironment).Hidden = true
	t.FindHeaderByTitle(_titleRealtimeAPI).Hidden = true
	if env.Provider == types.LocalProviderType {
		hideReplicaCountColumns(&t)
	}

	out += t.MustFormat()

	if env.Provider != types.LocalProviderType && realtimeAPI.Spec.Monitoring != nil {
		switch realtimeAPI.Spec.Monitoring.ModelType {
		case userconfig.ClassificationModelType:
			out += "\n" + classificationMetricsStr(&realtimeAPI.Metrics)
		case userconfig.RegressionModelType:
			out += "\n" + regressionMetricsStr(&realtimeAPI.Metrics)
		}
	}

	if realtimeAPI.DashboardURL != "" {
		out += "\n" + console.Bold("metrics dashboard: ") + realtimeAPI.DashboardURL + "\n"
	}

	out += "\n" + console.Bold("endpoint: ") + realtimeAPI.Endpoint

	out += fmt.Sprintf("\n%s curl %s -X POST -H \"Content-Type: application/json\" -d @sample.json\n", console.Bold("curl:"), realtimeAPI.Endpoint)

	if realtimeAPI.Spec.Predictor.Type == userconfig.TensorFlowPredictorType || realtimeAPI.Spec.Predictor.Type == userconfig.ONNXPredictorType {
		out += "\n" + describeModelInput(&realtimeAPI.Status, realtimeAPI.Endpoint)
	}

	out += titleStr("configuration") + strings.TrimSpace(realtimeAPI.Spec.UserStr(env.Provider))

	return out, nil
}

func realtimeAPIsTable(realtimeAPIs []schema.RealtimeAPI, envNames []string) table.Table {
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
			latencyStr(&realtimeAPI.Metrics),
			code2XXStr(&realtimeAPI.Metrics),
			code4XXStr(&realtimeAPI.Metrics),
			code5XXStr(&realtimeAPI.Metrics),
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

			shapeRowEntry := ""
			if len(shapeStr) == 1 && shapeStr[0] == "scalar" {
				shapeRowEntry = "scalar"
			} else if len(shapeStr) == 1 && shapeStr[0] == "unknown" {
				shapeRowEntry = "unknown"
			} else {
				shapeRowEntry = "(" + strings.Join(shapeStr, ", ") + ")"
			}
			rows[rowNum] = []interface{}{
				modelName,
				inputName,
				inputSignature.Type,
				shapeRowEntry,
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
