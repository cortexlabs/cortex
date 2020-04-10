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
	"github.com/cortexlabs/cortex/pkg/lib/json"
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

var _flagWatch bool

func init() {
	addEnvFlag(_getCmd, types.LocalProviderType.String())
	_getCmd.PersistentFlags().BoolVarP(&_flagWatch, "watch", "w", false, "re-run the command every second")
}

var _getCmd = &cobra.Command{
	Use:   "get [API_NAME]",
	Short: "get information about apis",
	Args:  cobra.RangeArgs(0, 1),
	Run: func(cmd *cobra.Command, args []string) {
		telemetry.Event("cli.get")

		env := MustReadOrConfigureEnv(_flagEnv)
		rerun(func() (string, error) {
			return get(args, env)
		})
	},
}

func get(args []string, env cliconfig.Environment) (string, error) {
	if len(args) == 0 {
		return getAPIs(env)
	}
	return getAPI(env, args[0])
}

func getAPIs(env cliconfig.Environment) (string, error) {
	var apisRes schema.GetAPIsResponse
	var err error

	if env.Provider == types.AWSProviderType {
		apisRes, err = cluster.GetAPIs(MustGetOperatorConfig(env.Name))
		if err != nil {
			// TODO
		}
	} else {
		apisRes, err = local.GetAPIs()
		if err != nil {
			// TODO
		}
	}

	t := apiTable(apisRes.APIs, apisRes.Statuses, apisRes.AllMetrics, true)
	return t.MustFormat(), nil
}

func getAPI(env cliconfig.Environment, apiName string) (string, error) {
	var apiRes schema.GetAPIResponse
	var err error
	if env.Provider == types.AWSProviderType {
		apiRes, err = cluster.GetAPI(MustGetOperatorConfig(env.Name), apiName)
		if err != nil {
			// TODO
		}
	} else {
		apiRes, err = local.GetAPI(apiName)
		if err != nil {
			// TODO
		}
	}

	var out string

	t := apiTable([]spec.API{apiRes.API}, []status.Status{apiRes.Status}, []metrics.Metrics{apiRes.Metrics}, false)
	out += t.MustFormat()

	api := apiRes.API

	if api.Tracker != nil {
		switch api.Tracker.ModelType {
		case userconfig.ClassificationModelType:
			out += "\n" + classificationMetricsStr(&apiRes.Metrics)
		case userconfig.RegressionModelType:
			out += "\n" + regressionMetricsStr(&apiRes.Metrics)
		}
	}

	apiEndpoint := urls.Join(apiRes.BaseURL, "/predict")
	if env.Provider == types.AWSProviderType {
		apiEndpoint = urls.Join(apiRes.BaseURL, *api.Endpoint)
	}
	out += "\n" + console.Bold("endpoint: ") + apiEndpoint

	out += fmt.Sprintf("\n%s curl %s?debug=true -X POST -H \"Content-Type: application/json\" -d @sample.json\n", console.Bold("curl:"), apiEndpoint)

	if api.Predictor.Type == userconfig.TensorFlowPredictorType || api.Predictor.Type == userconfig.ONNXPredictorType {
		out += "\n" + describeModelInput(&apiRes.Status, apiEndpoint)
	}

	out += titleStr("configuration") + strings.TrimSpace(api.UserStr())

	return out, nil
}

func apiTable(apis []spec.API, statuses []status.Status, allMetrics []metrics.Metrics, includeAPIName bool) table.Table {
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
			{Title: "api", Hidden: !includeAPIName},
			{Title: "status"},
			{Title: "up-to-date"},
			{Title: "stale", Hidden: totalStale == 0},
			{Title: "requested"},
			{Title: "failed", Hidden: totalFailed == 0},
			{Title: "last update"},
			{Title: "avg request"},
			{Title: "2XX"},
			{Title: "4XX", Hidden: total4XX == 0},
			{Title: "5XX", Hidden: total5XX == 0},
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

func MakeRequest(request *http.Request) ([]byte, error) {
	client := http.Client{
		Timeout: 600 * time.Second,
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
		},
	}

	response, err := client.Do(request)
	if err != nil {
		return nil, errors.Wrap(err, errStrFailedToConnect(*request.URL))
	}
	defer response.Body.Close()

	if response.StatusCode != 200 {
		bodyBytes, err := ioutil.ReadAll(response.Body)
		if err != nil {
			return nil, errors.Wrap(err, _errStrRead)
		}
		return nil, ErrorResponseUnknown(string(bodyBytes), response.StatusCode)
	}

	bodyBytes, err := ioutil.ReadAll(response.Body)
	if err != nil {
		return nil, errors.Wrap(err, _errStrRead)
	}
	return bodyBytes, nil
}

func getAPISummary(apiEndpoint string) (*schema.APISummary, error) {
	httpsAPIEndpoint := strings.Replace(apiEndpoint, "http://", "https://", 1)
	req, err := http.NewRequest("GET", httpsAPIEndpoint, nil)
	if err != nil {
		return nil, errors.Wrap(err, "unable to request api summary")
	}
	req.Header.Set("Content-Type", "application/json")
	response, err := MakeRequest(req)
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
