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

	"github.com/spf13/cobra"

	"github.com/cortexlabs/cortex/pkg/consts"
	"github.com/cortexlabs/cortex/pkg/lib/cast"
	"github.com/cortexlabs/cortex/pkg/lib/console"
	"github.com/cortexlabs/cortex/pkg/lib/errors"
	"github.com/cortexlabs/cortex/pkg/lib/exit"
	"github.com/cortexlabs/cortex/pkg/lib/json"
	s "github.com/cortexlabs/cortex/pkg/lib/strings"
	"github.com/cortexlabs/cortex/pkg/lib/table"
	"github.com/cortexlabs/cortex/pkg/lib/telemetry"
	libtime "github.com/cortexlabs/cortex/pkg/lib/time"
	"github.com/cortexlabs/cortex/pkg/lib/urls"
	"github.com/cortexlabs/cortex/pkg/operator/api/context"
	"github.com/cortexlabs/cortex/pkg/operator/api/resource"
	"github.com/cortexlabs/cortex/pkg/operator/api/schema"
	"github.com/cortexlabs/cortex/pkg/operator/api/userconfig"
)

var flagWatch bool
var flagVerbose bool
var flagSummary bool
var flagAllDeployments bool

func init() {
	addAppNameFlag(getCmd)
	addEnvFlag(getCmd)
	getCmd.PersistentFlags().BoolVarP(&flagWatch, "watch", "w", false, "re-run the command every second")
	getCmd.PersistentFlags().BoolVarP(&flagSummary, "summary", "s", false, "show summarized output")
	getCmd.PersistentFlags().BoolVarP(&flagVerbose, "verbose", "v", false, "show verbose output")
	getCmd.PersistentFlags().BoolVarP(&flagAllDeployments, "all-deployments", "a", false, "list all deployments")
}

var getCmd = &cobra.Command{
	Use:   "get [API_NAME]",
	Short: "get information about deployments",
	Args:  cobra.RangeArgs(0, 1),
	Run: func(cmd *cobra.Command, args []string) {
		telemetry.Event("cli.get")

		rerun(func() (string, error) {
			return runGet(cmd, args)
		})
	},
}

func runGet(cmd *cobra.Command, args []string) (string, error) {
	if flagAllDeployments || !IsAppNameSpecified() {
		return allDeploymentsStr()
	}

	appName, err := AppNameFromFlagOrConfig()
	if err != nil {
		exit.Error(err)
	}

	resourcesRes, err := getResourcesResponse(appName)
	if err != nil {
		// note: if modifying this string, search the codebase for it and change all occurrences
		if strings.HasSuffix(err.Error(), "is not deployed") {
			return console.Bold(err.Error()), nil
		}
		return "", err
	}

	switch len(args) {
	case 0:
		if flagSummary {
			return resourceStatusesStr(resourcesRes), nil
		}
		return allResourcesStr(resourcesRes), nil

	case 1:
		resourceName := args[0]
		if _, err = resourcesRes.Context.VisibleResourceByName(resourceName); err != nil {
			return "", err
		}

		return resourceByNameStr(resourceName, resourcesRes, flagVerbose)
	}

	return "", errors.New("too many args") // unexpected
}

func allDeploymentsStr() (string, error) {
	httpResponse, err := HTTPGet("/deployments", map[string]string{})
	if err != nil {
		return "", err
	}

	var deploymentsRes schema.GetDeploymentsResponse
	if err = json.Unmarshal(httpResponse, &deploymentsRes); err != nil {
		return "", err
	}

	if len(deploymentsRes.Deployments) == 0 {
		return console.Bold("no deployments found"), nil
	}

	fullTable := table.Table{}

	for deploymentIndex, deployment := range deploymentsRes.Deployments {
		resourcesRes, err := getResourcesResponse(deployment.Name)
		if err != nil {
			return "", err
		}

		t := apiResourceTable(resourcesRes.APIGroupStatuses)

		if deploymentIndex == 0 {
			deploymentHeader := table.Header{Title: "deployment"}
			fullTable.Headers = append([]table.Header{deploymentHeader}, t.Headers...)
		}

		for i := range t.Rows {
			t.Rows[i] = append([]interface{}{deployment.Name}, t.Rows[i]...)
		}

		fullTable.Rows = append(fullTable.Rows, t.Rows...)
	}

	return table.Format(fullTable)
}

func getResourcesResponse(appName string) (*schema.GetResourcesResponse, error) {
	params := map[string]string{"appName": appName}
	httpResponse, err := HTTPGet("/resources", params)
	if err != nil {
		return nil, err
	}

	var resourcesRes schema.GetResourcesResponse
	if err = json.Unmarshal(httpResponse, &resourcesRes); err != nil {
		return nil, err
	}

	return &resourcesRes, nil
}

func resourceByNameStr(resourceName string, resourcesRes *schema.GetResourcesResponse, flagVerbose bool) (string, error) {
	rs, err := resourcesRes.Context.VisibleResourceByName(resourceName)
	if err != nil {
		return "", err
	}
	switch resourceType := rs.GetResourceType(); resourceType {
	case resource.APIType:
		return describeAPI(resourceName, resourcesRes, flagVerbose)
	default:
		return "", resource.ErrorInvalidType(resourceType.String())
	}
}

func resourcesByTypeStr(resourceType resource.Type, resourcesRes *schema.GetResourcesResponse) (string, error) {
	switch resourceType {
	case resource.APIType:
		return apisStr(resourcesRes.APIGroupStatuses), nil
	default:
		return "", resource.ErrorInvalidType(resourceType.String())
	}
}

func resourceByNameAndTypeStr(resourceName string, resourceType resource.Type, resourcesRes *schema.GetResourcesResponse, flagVerbose bool) (string, error) {
	switch resourceType {
	case resource.APIType:
		return describeAPI(resourceName, resourcesRes, flagVerbose)
	default:
		return "", resource.ErrorInvalidType(resourceType.String())
	}
}

func allResourcesStr(resourcesRes *schema.GetResourcesResponse) string {
	out := ""
	out += apisStr(resourcesRes.APIGroupStatuses)
	return out
}

func apisStr(apiGroupStatuses map[string]*resource.APIGroupStatus) string {
	if len(apiGroupStatuses) == 0 {
		return ""
	}

	return table.MustFormat(apiResourceTable(apiGroupStatuses))
}

func describeAPI(name string, resourcesRes *schema.GetResourcesResponse, flagVerbose bool) (string, error) {
	groupStatus := resourcesRes.APIGroupStatuses[name]
	if groupStatus == nil {
		return "", userconfig.ErrorUndefinedResource(name, resource.APIType)
	}

	ctx := resourcesRes.Context
	api := ctx.APIs[name]

	var updatedAt *time.Time
	if groupStatus.ActiveStatus != nil {
		updatedAt = groupStatus.ActiveStatus.Start
	}

	row := []interface{}{
		groupStatus.Message(),
		groupStatus.ReadyUpdated,
		groupStatus.ReadyStale(),
		groupStatus.Requested,
		groupStatus.FailedUpdated,
		libtime.Since(updatedAt),
	}

	headers := []table.Header{
		{Title: "status"},
		{Title: "up-to-date"},
		{Title: "stale", Hidden: groupStatus.ReadyStale() == 0},
		{Title: "requested"},
		{Title: "failed", Hidden: groupStatus.FailedUpdated == 0},
		{Title: "last update"},
	}

	apiEndpoint := urls.Join(resourcesRes.APIsBaseURL, *api.Endpoint)

	statusTable := table.Table{
		Headers: headers,
		Rows:    [][]interface{}{row},
	}

	var out string
	apiMetrics, err := getAPIMetrics(ctx.App.Name, api.Name)
	statusTable = appendNetworkMetrics(statusTable, apiMetrics) // adds blank stats when there is an error
	out = table.MustFormat(statusTable) + "\n"

	var predictionMetrics string
	if err != nil {
		if !strings.Contains(err.Error(), "api is still initializing") {
			predictionMetrics = fmt.Sprintf("\nerror fetching metrics: %s\n", err.Error())
		}
	}

	if api.Tracker != nil && len(predictionMetrics) == 0 {
		predictionMetrics = "\n" + predictionMetricsTable(apiMetrics, api) + "\n"
	}

	out += predictionMetrics

	out += "\n" + console.Bold("endpoint: ") + apiEndpoint

	if !flagVerbose {
		return out, nil
	}

	out += fmt.Sprintf("\n%s curl %s?debug=true -X POST -H \"Content-Type: application/json\" -d @sample.json", console.Bold("curl:"), apiEndpoint)

	if api.TensorFlow != nil || api.ONNX != nil {
		out += "\n\n" + describeModelInput(groupStatus, apiEndpoint)
	}

	if api != nil {
		out += "\n" + titleStr("configuration") + strings.TrimSpace(api.UserConfigStr())
	}

	return out, nil
}

func getAPIMetrics(appName, apiName string) (schema.APIMetrics, error) {
	params := map[string]string{"appName": appName, "apiName": apiName}
	httpResponse, err := HTTPGet("/metrics", params)
	if err != nil {
		return schema.APIMetrics{}, err
	}

	var apiMetrics schema.APIMetrics
	err = json.Unmarshal(httpResponse, &apiMetrics)
	if err != nil {
		return schema.APIMetrics{}, err
	}

	return apiMetrics, nil
}

func appendNetworkMetrics(apiTable table.Table, apiMetrics schema.APIMetrics) table.Table {
	inferenceLatency := "-"
	code2XX := "-"
	code4XX := 0
	code5XX := 0

	if apiMetrics.NetworkStats != nil {
		code4XX = apiMetrics.NetworkStats.Code4XX
		code5XX = apiMetrics.NetworkStats.Code5XX
		if apiMetrics.NetworkStats.Latency != nil {
			if *apiMetrics.NetworkStats.Latency < 1000 {
				inferenceLatency = fmt.Sprintf("%.6g ms", *apiMetrics.NetworkStats.Latency)
			} else {
				inferenceLatency = fmt.Sprintf("%.6g s", (*apiMetrics.NetworkStats.Latency)/1000)
			}
		}
		if apiMetrics.NetworkStats.Code2XX != 0 {
			code2XX = s.Int(apiMetrics.NetworkStats.Code2XX)
		}
	}

	headers := []table.Header{
		{Title: "avg inference"},
		{Title: "2XX"},
		{Title: "4XX", Hidden: code4XX == 0},
		{Title: "5XX", Hidden: code5XX == 0},
	}

	row := []interface{}{
		inferenceLatency,
		code2XX,
		code4XX,
		code5XX,
	}

	apiTable.Headers = append(apiTable.Headers, headers...)
	apiTable.Rows[0] = append(apiTable.Rows[0], row...)

	return apiTable
}

func predictionMetricsTable(apiMetrics schema.APIMetrics, api *context.API) string {
	if api.Tracker == nil {
		return ""
	}

	if api.Tracker.ModelType == userconfig.ClassificationModelType {
		return classificationMetricsTable(apiMetrics)
	}
	return regressionMetricsTable(apiMetrics)
}

func regressionMetricsTable(apiMetrics schema.APIMetrics) string {
	minStr := "-"
	maxStr := "-"
	avgStr := "-"

	if apiMetrics.RegressionStats != nil {
		if apiMetrics.RegressionStats.Min != nil {
			minStr = fmt.Sprintf("%.9g", *apiMetrics.RegressionStats.Min)
		}

		if apiMetrics.RegressionStats.Max != nil {
			maxStr = fmt.Sprintf("%.9g", *apiMetrics.RegressionStats.Max)
		}

		if apiMetrics.RegressionStats.Avg != nil {
			avgStr = fmt.Sprintf("%.9g", *apiMetrics.RegressionStats.Avg)
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

	return table.MustFormat(t)
}

func classificationMetricsTable(apiMetrics schema.APIMetrics) string {
	classList := make([]string, len(apiMetrics.ClassDistribution))

	i := 0
	for inputName := range apiMetrics.ClassDistribution {
		classList[i] = inputName
		i++
	}
	sort.Strings(classList)

	rows := make([][]interface{}, len(classList))
	for rowNum, className := range classList {
		rows[rowNum] = []interface{}{
			className,
			apiMetrics.ClassDistribution[className],
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

	out := table.MustFormat(t)

	if len(classList) == consts.MaxClassesPerRequest {
		out += fmt.Sprintf("\n\nlisting at most %d classes, the complete list can be found in your cloudwatch dashboard", consts.MaxClassesPerRequest)
	}
	return out
}

func describeModelInput(groupStatus *resource.APIGroupStatus, apiEndpoint string) string {
	if groupStatus.ReadyUpdated+groupStatus.ReadyStaleCompute == 0 {
		return "the model's input schema will be available when the api is live"
	}

	apiSummary, err := getAPISummary(apiEndpoint)
	if err != nil {
		return "error retrieving the model's input schema: " + err.Error()
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

	return table.MustFormat(t)
}

func getAPISummary(apiEndpoint string) (*schema.APISummary, error) {
	httpsAPIEndpoint := strings.Replace(apiEndpoint, "http://", "https://", 1)
	req, err := http.NewRequest("GET", httpsAPIEndpoint, nil)
	if err != nil {
		return nil, errors.Wrap(err, "unable to request api summary")
	}
	req.Header.Set("Content-Type", "application/json")
	response, err := apiClient.MakeRequest(req)
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

func dataStatusSummary(dataStatus *resource.DataStatus) string {
	headers := []table.Header{
		{Title: "status"},
		{Title: "start"},
		{Title: "end"},
	}
	row := []interface{}{
		dataStatus.Message(),
		libtime.LocalTimestamp(dataStatus.Start),
		libtime.LocalTimestamp(dataStatus.End),
	}

	t := table.Table{
		Headers: headers,
		Rows:    [][]interface{}{row},
	}
	return table.MustFormat(t) + "\n"
}

func valueStr(value interface{}) string {
	return titleStr("value") + s.Obj(value) + "\n"
}

func dataResourceTable(resources []context.Resource, dataStatuses map[string]*resource.DataStatus, resourceType resource.Type) string {
	rows := make([][]interface{}, len(resources))
	for rowNum, res := range resources {
		dataStatus := dataStatuses[res.GetID()]
		rows[rowNum] = []interface{}{
			res.GetName(),
			dataStatus.Message(),
			libtime.Since(dataStatus.End),
		}
	}

	title := resourceType.UserFacing()
	t := table.Table{
		Headers: []table.Header{
			{Title: title, MaxWidth: 32},
			{Title: "status", MaxWidth: 21},
			{Title: "age"},
		},
		Rows: rows,
	}

	return table.MustFormat(t)
}

func apiResourceTable(apiGroupStatuses map[string]*resource.APIGroupStatus) table.Table {
	rows := make([][]interface{}, 0, len(apiGroupStatuses))

	var totalFailed int32
	var totalStale int32
	for name, groupStatus := range apiGroupStatuses {
		var updatedAt *time.Time
		if groupStatus.ActiveStatus != nil {
			updatedAt = groupStatus.ActiveStatus.Start
		}

		rows = append(rows, []interface{}{
			name,
			groupStatus.Message(),
			groupStatus.ReadyUpdated,
			groupStatus.ReadyStale(),
			groupStatus.Requested,
			groupStatus.FailedUpdated,
			libtime.Since(updatedAt),
		})

		totalFailed += groupStatus.FailedUpdated
		totalStale += groupStatus.ReadyStale()
	}

	t := table.Table{
		Headers: []table.Header{
			{Title: resource.APIType.UserFacing()},
			{Title: "status"},
			{Title: "up-to-date"},
			{Title: "stale", Hidden: totalStale == 0},
			{Title: "requested"},
			{Title: "failed", Hidden: totalFailed == 0},
			{Title: "last update"},
		},
		Rows: rows,
	}

	return t
}

func titleStr(title string) string {
	return "\n" + console.Bold(title) + "\n"
}

func resourceStatusesStr(resourcesRes *schema.GetResourcesResponse) string {
	var titles, values []string

	if len(resourcesRes.APIGroupStatuses) != 0 {
		titles = append(titles, resource.APIType.UserFacingPlural())
		values = append(values, apiStatusesStr(resourcesRes.APIGroupStatuses))
	}

	maxTitleLen := s.MaxLen(titles...)

	out := ""
	for i, title := range titles {
		paddingWidth := maxTitleLen - len(title) + 3
		padding := strings.Repeat(" ", paddingWidth)
		out += title + ":" + padding + values[i]
		if i != len(titles)-1 {
			out += "\n"
		}
	}

	return out
}

func apiStatusesStr(apiGroupStatuses map[string]*resource.APIGroupStatus) string {
	var statuses = make([]resource.Status, len(apiGroupStatuses))
	i := 0
	for _, apiGroupStatus := range apiGroupStatuses {
		statuses[i] = apiGroupStatus
		i++
	}
	return StatusStr(statuses)
}

func StatusStr(statuses []resource.Status) string {
	if len(statuses) == 0 {
		return "none"
	}

	messageBuckets := make(map[int][]string)
	for _, status := range statuses {
		bucketKey := status.GetCode().SortBucket()
		messageBuckets[bucketKey] = append(messageBuckets[bucketKey], status.Message())
	}

	var bucketKeys []int
	for bucketKey := range messageBuckets {
		bucketKeys = append(bucketKeys, bucketKey)
	}
	sort.Ints(bucketKeys)

	var messageItems []string

	for _, bucketKey := range bucketKeys {
		messageCounts := make(map[string]int)
		for _, message := range messageBuckets[bucketKey] {
			messageCounts[message]++
		}

		var messages []string
		for message := range messageCounts {
			messages = append(messages, message)
		}
		sort.Strings(messages)

		for _, message := range messages {
			messageItem := fmt.Sprintf("%d %s", messageCounts[message], message)
			messageItems = append(messageItems, messageItem)
		}
	}

	return strings.Join(messageItems, " | ")
}
