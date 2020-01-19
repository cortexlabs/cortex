/*
Copyright 2019 Cortex Labs, Inc.

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
	"net/http"
	"sort"
	"strings"
	"time"

	"github.com/cortexlabs/cortex/pkg/consts"
	"github.com/cortexlabs/cortex/pkg/lib/cast"
	"github.com/cortexlabs/cortex/pkg/lib/console"
	"github.com/cortexlabs/cortex/pkg/lib/errors"
	"github.com/cortexlabs/cortex/pkg/lib/json"
	s "github.com/cortexlabs/cortex/pkg/lib/strings"
	"github.com/cortexlabs/cortex/pkg/lib/table"
	"github.com/cortexlabs/cortex/pkg/lib/telemetry"
	libtime "github.com/cortexlabs/cortex/pkg/lib/time"
	"github.com/cortexlabs/cortex/pkg/lib/urls"
	"github.com/cortexlabs/cortex/pkg/operator/schema"
	"github.com/cortexlabs/cortex/pkg/types/metrics"
	"github.com/cortexlabs/cortex/pkg/types/spec"
	"github.com/cortexlabs/cortex/pkg/types/status"
	"github.com/cortexlabs/cortex/pkg/types/userconfig"
	"github.com/spf13/cobra"
)

var _flagWatch bool

func init() {
	addEnvFlag(_getCmd)
	_getCmd.PersistentFlags().BoolVarP(&_flagWatch, "watch", "w", false, "re-run the command every second")
}

var _getCmd = &cobra.Command{
	Use:   "get [API_NAME]",
	Short: "get information about apis",
	Args:  cobra.RangeArgs(0, 1),
	Run: func(cmd *cobra.Command, args []string) {
		telemetry.Event("cli.get")

		rerun(func() (string, error) {
			return get(args)
		})
	},
}

func get(args []string) (string, error) {
	switch len(args) {
	case 0:
		return getAPIs()

	case 1:
		return getAPI(args[0])
	}

	return "", errors.New("too many args") // unexpected
}

func getAPIs() (string, error) {
	httpRes, err := HTTPGet("/get")
	if err != nil {
		return "", err
	}

	var apisRes schema.GetAPIsResponse
	if err = json.Unmarshal(httpRes, &apisRes); err != nil {
		return "", err
	}

	if len(apisRes.APIs) == 0 {
		return console.Bold("no apis are deployed, see https://cortex.dev for guides and examples"), nil
	}

	t := apiTable(apisRes.APIs, apisRes.Statuses, apisRes.AllMetrics)
	return t.MustFormat(), nil
}

func getAPI(apiName string) (string, error) {
	httpRes, err := HTTPGet("/get/" + apiName)
	if err != nil {
		// note: if modifying this string, search the codebase for it and change all occurrences
		if strings.HasSuffix(errors.Message(err), "is not deployed") {
			return console.Bold(errors.Message(err)), nil
		}
		return "", err
	}

	var apiRes schema.GetAPIResponse
	if err = json.Unmarshal(httpRes, &apiRes); err != nil {
		return "", err
	}

	var out string

	t := apiTable([]spec.API{*apiRes.API}, []status.Status{*apiRes.Status}, []metrics.Metrics{*apiRes.Metrics})
	out += t.MustFormat()

	api := apiRes.API

	if api.Tracker != nil {
		switch api.Tracker.ModelType {
		case userconfig.ClassificationModelType:
			out += "\n" + classificationMetricsStr(apiRes.Metrics)
		case userconfig.RegressionModelType:
			out += "\n" + regressionMetricsStr(apiRes.Metrics)
		}
	}

	apiEndpoint := urls.Join(apiRes.BaseURL, *api.Endpoint)
	out += "\n" + console.Bold("endpoint: ") + apiEndpoint

	out += fmt.Sprintf("\n%s curl %s?debug=true -X POST -H \"Content-Type: application/json\" -d @sample.json", console.Bold("curl:"), apiEndpoint)

	if api.Predictor.Type == userconfig.TensorFlowPredictorType || api.Predictor.Type == userconfig.ONNXPredictorType {
		out += "\n\n" + describeModelInput(apiRes.Status, apiEndpoint)
	}

	out += titleStr("configuration") + strings.TrimSpace(api.UserStr())

	return out, nil
}

func apiTable(apis []spec.API, statuses []status.Status, allMetrics []metrics.Metrics) table.Table {
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
			{Title: "api"},
			{Title: "status"},
			{Title: "up-to-date"},
			{Title: "stale", Hidden: totalStale == 0},
			{Title: "requested"},
			{Title: "failed", Hidden: totalFailed == 0},
			{Title: "last update"},
			{Title: "avg inference"},
			{Title: "2XX"},
			{Title: "4XX", Hidden: total4XX == 0},
			{Title: "5XX", Hidden: total5XX == 0},
		},
		Rows: rows,
	}
}

func metricsStrs(metrics *metrics.Metrics) (string, string, string, string) {
	inferenceLatency := "-"
	code2XX := "-"
	code4XX := "-"
	code5XX := "-"

	if metrics.NetworkStats != nil {
		if metrics.NetworkStats.Latency != nil {
			if *metrics.NetworkStats.Latency < 1000 {
				inferenceLatency = fmt.Sprintf("%.6g ms", *metrics.NetworkStats.Latency)
			} else {
				inferenceLatency = fmt.Sprintf("%.6g s", (*metrics.NetworkStats.Latency)/1000)
			}
		}

		if metrics.NetworkStats.Code2XX != 0 {
			code2XX = s.Int(metrics.NetworkStats.Code2XX)
		}
		if metrics.NetworkStats.Code4XX != 0 {
			code4XX = s.Int(metrics.NetworkStats.Code4XX)
		}
		if metrics.NetworkStats.Code5XX != 0 {
			code5XX = s.Int(metrics.NetworkStats.Code5XX)
		}
	}

	return inferenceLatency, code2XX, code4XX, code5XX
}

func latencyStr(metrics *metrics.Metrics) string {
	if metrics.NetworkStats != nil && metrics.NetworkStats.Latency != nil {
		if *metrics.NetworkStats.Latency < 1000 {
			return fmt.Sprintf("%.6g ms", *metrics.NetworkStats.Latency)
		}
		return fmt.Sprintf("%.6g s", (*metrics.NetworkStats.Latency)/1000)
	}
	return "-"
}

func code2XXStr(metrics *metrics.Metrics) string {
	if metrics.NetworkStats != nil && metrics.NetworkStats.Code2XX != 0 {
		return s.Int(metrics.NetworkStats.Code2XX)
	}
	return "-"
}

func code4XXStr(metrics *metrics.Metrics) string {
	if metrics.NetworkStats != nil && metrics.NetworkStats.Code4XX != 0 {
		return s.Int(metrics.NetworkStats.Code4XX)
	}
	return "-"
}

func code5XXStr(metrics *metrics.Metrics) string {
	if metrics.NetworkStats != nil && metrics.NetworkStats.Code5XX != 0 {
		return s.Int(metrics.NetworkStats.Code5XX)
	}
	return "-"
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
	classList := make([]string, len(metrics.ClassDistribution))

	i := 0
	for inputName := range metrics.ClassDistribution {
		classList[i] = inputName
		i++
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

func getAPISummary(apiEndpoint string) (*schema.APISummary, error) {
	httpsAPIEndpoint := strings.Replace(apiEndpoint, "http://", "https://", 1)
	req, err := http.NewRequest("GET", httpsAPIEndpoint, nil)
	if err != nil {
		return nil, errors.Wrap(err, "unable to request api summary")
	}
	req.Header.Set("Content-Type", "application/json")
	response, err := _apiClient.MakeRequest(req)
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
