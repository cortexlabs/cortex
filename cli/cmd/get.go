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
	"github.com/cortexlabs/cortex/pkg/lib/json"
	s "github.com/cortexlabs/cortex/pkg/lib/strings"
	"github.com/cortexlabs/cortex/pkg/lib/table"
	libtime "github.com/cortexlabs/cortex/pkg/lib/time"
	"github.com/cortexlabs/cortex/pkg/lib/urls"
	"github.com/cortexlabs/cortex/pkg/operator/api/context"
	"github.com/cortexlabs/cortex/pkg/operator/api/resource"
	"github.com/cortexlabs/cortex/pkg/operator/api/schema"
	"github.com/cortexlabs/cortex/pkg/operator/api/userconfig"
)

func init() {
	addAppNameFlag(getCmd)
	addEnvFlag(getCmd)
	addWatchFlag(getCmd)
	addSummaryFlag(getCmd)
	addVerboseFlag(getCmd)
	addAllDeploymentsFlag(getCmd)
	// addResourceTypesToHelp(getCmd)
}

var getCmd = &cobra.Command{
	Use:   "get [RESOURCE_NAME]",
	Short: "get information about resources",
	Long:  "Get information about resources.",
	Args:  cobra.RangeArgs(0, 1),
	Run: func(cmd *cobra.Command, args []string) {
		rerun(func() (string, error) {
			return runGet(cmd, args)
		})
	},
}

func runGet(cmd *cobra.Command, args []string) (string, error) {
	if flagAllDeployments || !IsAppNameSpecified() {
		return allDeploymentsStr()
	}

	resourcesRes, err := getResourcesResponse()
	if err != nil {
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

	var resourcesRes schema.GetDeploymentsResponse
	if err = json.Unmarshal(httpResponse, &resourcesRes); err != nil {
		return "", err
	}

	if len(resourcesRes.Deployments) == 0 {
		return console.Bold("\nno deployments found"), nil
	}

	rows := make([][]interface{}, len(resourcesRes.Deployments))
	for idx, deployment := range resourcesRes.Deployments {
		rows[idx] = []interface{}{
			deployment.Name,
			deployment.Status.String(),
			libtime.Since(&deployment.LastUpdated),
		}
	}

	t := table.Table{
		Headers: []table.Header{
			{Title: "name", MaxWidth: 32},
			{Title: "status", MaxWidth: 21},
			{Title: "last update"},
		},
		Rows: rows,
	}

	return "\n" + table.MustFormat(t), nil
}

func getResourcesResponse() (*schema.GetResourcesResponse, error) {
	appName, err := AppNameFromFlagOrConfig()
	if err != nil {
		return nil, err
	}

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

	return "\n" + apiResourceTable(apiGroupStatuses)
}

func describeAPI(name string, resourcesRes *schema.GetResourcesResponse, flagVerbose bool) (string, error) {
	groupStatus := resourcesRes.APIGroupStatuses[name]
	if groupStatus == nil {
		return "", userconfig.ErrorUndefinedResource(name, resource.APIType)
	}

	ctx := resourcesRes.Context
	api := ctx.APIs[name]

	var anyAPIStatus *resource.APIStatus
	for _, apiStatus := range resourcesRes.APIStatuses {
		if apiStatus.APIName == name {
			anyAPIStatus = apiStatus
			break
		}
	}
	var updatedAt *time.Time
	if groupStatus.ActiveStatus != nil {
		updatedAt = groupStatus.ActiveStatus.Start
	}

	row := []interface{}{
		groupStatus.Message(),
		groupStatus.ReadyUpdated,
		groupStatus.Available(),
		groupStatus.Requested,
		groupStatus.ReadyStaleCompute,
		groupStatus.ReadyStaleModel,
		groupStatus.FailedUpdated,
		libtime.Since(updatedAt),
	}

	headers := []table.Header{
		{Title: "status"},
		{Title: "up-to-date"},
		{Title: "available"},
		{Title: "requested"},
		{Title: "stale compute", Hidden: groupStatus.ReadyStaleCompute == 0},
		{Title: "stale model", Hidden: groupStatus.ReadyStaleModel == 0},
		{Title: "failed", Hidden: groupStatus.FailedUpdated == 0},
		{Title: "last update"},
	}

	apiEndpoint := urls.Join(resourcesRes.APIsBaseURL, anyAPIStatus.Path)

	statusTable := table.Table{
		Headers: headers,
		Rows:    [][]interface{}{row},
	}

	var predictionMetrics string
	apiMetrics, err := getAPIMetrics(ctx.App.Name, api.Name)
	if err != nil || apiMetrics == nil {
		predictionMetrics = "\nmetrics are not available yet\n"
	} else {
		statusTable = appendNetworkMetrics(statusTable, apiMetrics)
		if api.Tracker != nil {
			predictionMetrics = "\n" + predictionMetricsTable(apiMetrics, api) + "\n"
		}
	}

	out := "\n" + table.MustFormat(statusTable) + "\n"
	out += predictionMetrics

	out += "\n" + console.Bold("url: ") + apiEndpoint

	if !flagVerbose {
		return out, nil
	}

	out += fmt.Sprintf("\n%s curl %s?debug=true -X POST -H \"Content-Type: application/json\" -d @sample.json", console.Bold("curl:"), apiEndpoint)

	out += "\n\n" + describeModelInput(groupStatus, apiEndpoint)

	if api != nil {
		out += "\n" + titleStr("configuration") + strings.TrimSpace(api.UserConfigStr())
	}

	return out, nil
}

func getAPIMetrics(appName, apiName string) (*schema.APIMetrics, error) {
	params := map[string]string{"appName": appName, "apiName": apiName}
	httpResponse, err := HTTPGet("/metrics", params)
	if err != nil {
		return nil, err
	}

	var apiMetrics schema.APIMetrics
	err = json.Unmarshal(httpResponse, &apiMetrics)
	if err != nil {
		return nil, err
	}

	return &apiMetrics, nil
}

func appendNetworkMetrics(apiTable table.Table, apiMetrics *schema.APIMetrics) table.Table {
	headers := []table.Header{
		{Title: "avg latency", Hidden: apiMetrics.NetworkStats.Latency == nil},
		{Title: "2XX", Hidden: apiMetrics.NetworkStats.Code2XX == 0},
		{Title: "4XX", Hidden: apiMetrics.NetworkStats.Code4XX == 0},
		{Title: "5XX", Hidden: apiMetrics.NetworkStats.Code5XX == 0},
	}

	latency := ""
	if apiMetrics.NetworkStats.Latency != nil {
		if *apiMetrics.NetworkStats.Latency < 1000 {
			latency = fmt.Sprintf("%.6g ms", *apiMetrics.NetworkStats.Latency)
		} else {
			latency = fmt.Sprintf("%.6g s", (*apiMetrics.NetworkStats.Latency)/1000)
		}
	}

	row := []interface{}{
		latency,
		apiMetrics.NetworkStats.Code2XX,
		apiMetrics.NetworkStats.Code4XX,
		apiMetrics.NetworkStats.Code5XX,
	}

	apiTable.Headers = append(apiTable.Headers, headers...)
	apiTable.Rows[0] = append(apiTable.Rows[0], row...)

	return apiTable
}

func predictionMetricsTable(apiMetrics *schema.APIMetrics, api *context.API) string {
	if api.Tracker == nil {
		return ""
	}

	if api.Tracker.ModelType == userconfig.ClassificationModelType {
		return classificationMetricsTable(apiMetrics)
	}
	return regressionMetricsTable(apiMetrics)
}

func regressionMetricsTable(apiMetrics *schema.APIMetrics) string {
	minStr := "-"
	if apiMetrics.RegressionStats.Min != nil {
		minStr = fmt.Sprintf("%.9g", *apiMetrics.RegressionStats.Min)
	}

	maxStr := "-"
	if apiMetrics.RegressionStats.Max != nil {
		maxStr = fmt.Sprintf("%.9g", *apiMetrics.RegressionStats.Max)
	}

	avgStr := "-"
	if apiMetrics.RegressionStats.Avg != nil {
		avgStr = fmt.Sprintf("%.9g", *apiMetrics.RegressionStats.Avg)
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

func classificationMetricsTable(apiMetrics *schema.APIMetrics) string {
	classList := make([]string, len(apiMetrics.ClassDistribution))

	i := 0
	for inputName := range apiMetrics.ClassDistribution {
		classList[i] = inputName
		i++
	}
	sort.Strings(classList)

	if len(classList) > 0 && len(classList) < 4 {
		row := []interface{}{}
		headers := []table.Header{}

		for _, className := range classList {
			headers = append(headers, table.Header{Title: s.TruncateEllipses(className, 20), MaxWidth: 20})
			row = append(row, apiMetrics.ClassDistribution[className])
		}

		t := table.Table{
			Headers: headers,
			Rows:    [][]interface{}{row},
		}

		return table.MustFormat(t)
	}

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
	if groupStatus.Available() == 0 {
		return "the model's input schema will be available when the API is live"
	}

	modelInput, err := getModelInput(urls.Join(apiEndpoint, "signature"))
	if err != nil {
		return "error retreiving the model's input schema: " + err.Error()
	}

	rows := make([][]interface{}, len(modelInput.Signature))
	rowNum := 0
	for inputName, featureSignature := range modelInput.Signature {
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

func getModelInput(infoAPIPath string) (*schema.ModelInput, error) {
	req, err := http.NewRequest("GET", infoAPIPath, nil)
	if err != nil {
		return nil, errors.Wrap(err, "unable to request model input")
	}
	req.Header.Set("Content-Type", "application/json")
	response, err := httpsNoVerifyClient.makeRequest(req)
	if err != nil {
		return nil, err
	}

	var modelInput schema.ModelInput
	err = json.DecodeWithNumber(response, &modelInput)
	if err != nil {
		return nil, errors.Wrap(err, "unable to parse model input response")
	}

	for _, featureSignature := range modelInput.Signature {
		featureSignature.Shape = cast.JSONNumbers(featureSignature.Shape)
	}

	return &modelInput, nil
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
	return "\n" + table.MustFormat(t)
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

func apiResourceTable(apiGroupStatuses map[string]*resource.APIGroupStatus) string {
	rows := make([][]interface{}, 0, len(apiGroupStatuses))

	totalFailed := 0
	for name, groupStatus := range apiGroupStatuses {
		var updatedAt *time.Time
		if groupStatus.ActiveStatus != nil {
			updatedAt = groupStatus.ActiveStatus.Start
		}

		rows = append(rows, []interface{}{
			name,
			groupStatus.Message(),
			groupStatus.ReadyUpdated,
			groupStatus.Available(),
			groupStatus.Requested,
			groupStatus.FailedUpdated,
			libtime.Since(updatedAt),
		})

		totalFailed += int(groupStatus.FailedUpdated)
	}

	t := table.Table{
		Headers: []table.Header{
			{Title: resource.APIType.UserFacing()},
			{Title: "status"},
			{Title: "up-to-date"},
			{Title: "available"},
			{Title: "requested"},
			{Title: "failed", Hidden: totalFailed == 0},
			{Title: "last update"},
		},
		Rows: rows,
	}

	return table.MustFormat(t)
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

	out := "\n"
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
