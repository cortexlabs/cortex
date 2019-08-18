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

	"github.com/cortexlabs/yaml"
	"github.com/spf13/cobra"

	"github.com/cortexlabs/cortex/pkg/lib/console"
	"github.com/cortexlabs/cortex/pkg/lib/errors"
	"github.com/cortexlabs/cortex/pkg/lib/json"
	"github.com/cortexlabs/cortex/pkg/lib/msgpack"
	"github.com/cortexlabs/cortex/pkg/lib/sets/strset"
	s "github.com/cortexlabs/cortex/pkg/lib/strings"
	"github.com/cortexlabs/cortex/pkg/lib/table"
	libtime "github.com/cortexlabs/cortex/pkg/lib/time"
	"github.com/cortexlabs/cortex/pkg/lib/urls"
	"github.com/cortexlabs/cortex/pkg/operator/api/context"
	"github.com/cortexlabs/cortex/pkg/operator/api/resource"
	"github.com/cortexlabs/cortex/pkg/operator/api/schema"
	"github.com/cortexlabs/cortex/pkg/operator/api/userconfig"
)

const (
	maxClassesToDisplay = 20
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
	Use:   "get [RESOURCE_TYPE] [RESOURCE_NAME]",
	Short: "get information about resources",
	Long:  "Get information about resources.",
	Args:  cobra.RangeArgs(0, 2),
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
		resourceNameOrType := args[0]

		resourceType, err := resource.VisibleResourceTypeFromPrefix(resourceNameOrType)
		if err != nil {
			if rerr, ok := err.(resource.Error); ok && rerr.Kind != resource.ErrInvalidType {
				return "", err
			}
		} else {
			return resourcesByTypeStr(resourceType, resourcesRes)
		}

		if _, err = resourcesRes.Context.VisibleResourceByName(resourceNameOrType); err != nil {
			if rerr, ok := err.(resource.Error); ok && rerr.Kind == resource.ErrNameNotFound {
				return "", resource.ErrorNameOrTypeNotFound(resourceNameOrType)
			}
			return "", err
		}

		return resourceByNameStr(resourceNameOrType, resourcesRes, flagVerbose)

	case 2:
		userResourceType := args[0]
		resourceName := args[1]
		resourceType, err := resource.VisibleResourceTypeFromPrefix(userResourceType)
		if err != nil {
			return "", resource.ErrorInvalidType(userResourceType)
		}
		return resourceByNameAndTypeStr(resourceName, resourceType, resourcesRes, flagVerbose)
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
		return "no deployments found", nil
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
			{Title: "last updated"},
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
	case resource.PythonPackageType:
		return describePythonPackage(resourceName, resourcesRes)
	case resource.RawColumnType:
		return describeRawColumn(resourceName, resourcesRes)
	case resource.AggregateType:
		return describeAggregate(resourceName, resourcesRes)
	case resource.TransformedColumnType:
		return describeTransformedColumn(resourceName, resourcesRes)
	case resource.TrainingDatasetType:
		return describeTrainingDataset(resourceName, resourcesRes)
	case resource.ModelType:
		return describeModel(resourceName, resourcesRes)
	case resource.APIType:
		return describeAPI(resourceName, resourcesRes, flagVerbose)
	default:
		return "", resource.ErrorInvalidType(resourceType.String())
	}
}

func resourcesByTypeStr(resourceType resource.Type, resourcesRes *schema.GetResourcesResponse) (string, error) {
	switch resourceType {
	case resource.PythonPackageType:
		return pythonPackagesStr(resourcesRes.DataStatuses, resourcesRes.Context), nil
	case resource.RawColumnType:
		return rawColumnsStr(resourcesRes.DataStatuses, resourcesRes.Context), nil
	case resource.AggregateType:
		return aggregatesStr(resourcesRes.DataStatuses, resourcesRes.Context), nil
	case resource.TransformedColumnType:
		return transformedColumnsStr(resourcesRes.DataStatuses, resourcesRes.Context), nil
	case resource.TrainingDatasetType:
		return trainingDataStr(resourcesRes.DataStatuses, resourcesRes.Context), nil
	case resource.ModelType:
		return modelsStr(resourcesRes.DataStatuses, resourcesRes.Context), nil
	case resource.APIType:
		return apisStr(resourcesRes.APIGroupStatuses), nil
	default:
		return "", resource.ErrorInvalidType(resourceType.String())
	}
}

func resourceByNameAndTypeStr(resourceName string, resourceType resource.Type, resourcesRes *schema.GetResourcesResponse, flagVerbose bool) (string, error) {
	switch resourceType {
	case resource.PythonPackageType:
		return describePythonPackage(resourceName, resourcesRes)
	case resource.RawColumnType:
		return describeRawColumn(resourceName, resourcesRes)
	case resource.AggregateType:
		return describeAggregate(resourceName, resourcesRes)
	case resource.TransformedColumnType:
		return describeTransformedColumn(resourceName, resourcesRes)
	case resource.TrainingDatasetType:
		return describeTrainingDataset(resourceName, resourcesRes)
	case resource.ModelType:
		return describeModel(resourceName, resourcesRes)
	case resource.APIType:
		return describeAPI(resourceName, resourcesRes, flagVerbose)
	default:
		return "", resource.ErrorInvalidType(resourceType.String())
	}
}

func allResourcesStr(resourcesRes *schema.GetResourcesResponse) string {
	ctx := resourcesRes.Context

	out := ""
	out += pythonPackagesStr(resourcesRes.DataStatuses, ctx)
	out += rawColumnsStr(resourcesRes.DataStatuses, ctx)
	out += aggregatesStr(resourcesRes.DataStatuses, ctx)
	out += transformedColumnsStr(resourcesRes.DataStatuses, ctx)
	out += trainingDataStr(resourcesRes.DataStatuses, ctx)
	out += modelsStr(resourcesRes.DataStatuses, ctx)
	out += apisStr(resourcesRes.APIGroupStatuses)
	return out
}

func pythonPackagesStr(dataStatuses map[string]*resource.DataStatus, ctx *context.Context) string {
	if len(ctx.PythonPackages) == 0 {
		return ""
	}

	resources := make([]context.Resource, 0, len(ctx.PythonPackages))
	for _, pythonPackage := range ctx.PythonPackages {
		resources = append(resources, pythonPackage)
	}

	return "\n" + dataResourceTable(resources, dataStatuses, resource.PythonPackageType) + "\n"
}

func rawColumnsStr(dataStatuses map[string]*resource.DataStatus, ctx *context.Context) string {
	if len(ctx.RawColumns) == 0 {
		return ""
	}

	resources := make([]context.Resource, 0, len(ctx.RawColumns))
	for _, rawColumn := range ctx.RawColumns {
		resources = append(resources, rawColumn)
	}

	return "\n" + dataResourceTable(resources, dataStatuses, resource.RawColumnType) + "\n"
}

func aggregatesStr(dataStatuses map[string]*resource.DataStatus, ctx *context.Context) string {
	if len(ctx.Aggregates) == 0 {
		return ""
	}

	resources := make([]context.Resource, 0, len(ctx.Aggregates))
	for _, aggregate := range ctx.Aggregates {
		resources = append(resources, aggregate)
	}

	return "\n" + dataResourceTable(resources, dataStatuses, resource.AggregateType) + "\n"
}

func transformedColumnsStr(dataStatuses map[string]*resource.DataStatus, ctx *context.Context) string {
	if len(ctx.TransformedColumns) == 0 {
		return ""
	}

	resources := make([]context.Resource, 0, len(ctx.TransformedColumns))
	for _, transformedColumn := range ctx.TransformedColumns {
		resources = append(resources, transformedColumn)
	}

	return "\n" + dataResourceTable(resources, dataStatuses, resource.TransformedColumnType) + "\n"
}

func trainingDataStr(dataStatuses map[string]*resource.DataStatus, ctx *context.Context) string {
	if len(ctx.Models) == 0 {
		return ""
	}

	resources := make([]context.Resource, 0, len(ctx.Models))
	for _, model := range ctx.Models {
		resources = append(resources, model.Dataset)
	}

	return "\n" + dataResourceTable(resources, dataStatuses, resource.TrainingDatasetType) + "\n"
}

func modelsStr(dataStatuses map[string]*resource.DataStatus, ctx *context.Context) string {
	if len(ctx.Models) == 0 {
		return ""
	}

	resources := make([]context.Resource, 0, len(ctx.Models))
	for _, model := range ctx.Models {
		resources = append(resources, model)
	}

	return "\n" + dataResourceTable(resources, dataStatuses, resource.ModelType) + "\n"
}

func apisStr(apiGroupStatuses map[string]*resource.APIGroupStatus) string {
	if len(apiGroupStatuses) == 0 {
		return ""
	}

	return "\n" + apiResourceTable(apiGroupStatuses)
}

func describePythonPackage(name string, resourcesRes *schema.GetResourcesResponse) (string, error) {
	pythonPackage := resourcesRes.Context.PythonPackages[name]
	if pythonPackage == nil {
		return "", userconfig.ErrorUndefinedResource(name, resource.PythonPackageType)
	}
	dataStatus := resourcesRes.DataStatuses[pythonPackage.GetID()]
	return dataStatusSummary(dataStatus), nil
}

func describeRawColumn(name string, resourcesRes *schema.GetResourcesResponse) (string, error) {
	rawColumn := resourcesRes.Context.RawColumns[name]
	if rawColumn == nil {
		return "", userconfig.ErrorUndefinedResource(name, resource.RawColumnType)
	}
	dataStatus := resourcesRes.DataStatuses[rawColumn.GetID()]
	out := dataStatusSummary(dataStatus)
	out += "\n" + titleStr("configuration") + rawColumn.UserConfigStr()
	return out, nil
}

func describeAggregate(name string, resourcesRes *schema.GetResourcesResponse) (string, error) {
	aggregate := resourcesRes.Context.Aggregates[name]
	if aggregate == nil {
		return "", userconfig.ErrorUndefinedResource(name, resource.AggregateType)
	}
	dataStatus := resourcesRes.DataStatuses[aggregate.ID]
	out := dataStatusSummary(dataStatus)

	if dataStatus.ExitCode == resource.ExitCodeDataSucceeded {
		params := map[string]string{"appName": resourcesRes.Context.App.Name}
		httpResponse, err := HTTPGet("/aggregate/"+aggregate.ID, params)
		if err != nil {
			return "", err
		}

		var aggregateRes schema.GetAggregateResponse
		err = json.Unmarshal(httpResponse, &aggregateRes)
		if err != nil {
			return "", errors.Wrap(err, "/aggregate", "response", string(httpResponse))
		}

		obj, err := msgpack.UnmarshalToInterface(aggregateRes.Value)
		if err != nil {
			return "", errors.Wrap(err, "/aggregate", "response", msgpack.ErrorUnmarshalMsgpack().Error())
		}
		out += valueStr(obj)
	}

	out += "\n" + titleStr("configuration") + aggregate.UserConfigStr()
	return out, nil
}

func describeTransformedColumn(name string, resourcesRes *schema.GetResourcesResponse) (string, error) {
	transformedColumn := resourcesRes.Context.TransformedColumns[name]
	if transformedColumn == nil {
		return "", userconfig.ErrorUndefinedResource(name, resource.TransformedColumnType)
	}
	dataStatus := resourcesRes.DataStatuses[transformedColumn.ID]
	out := dataStatusSummary(dataStatus)
	out += "\n" + titleStr("configuration") + transformedColumn.UserConfigStr()
	return out, nil
}

func describeTrainingDataset(name string, resourcesRes *schema.GetResourcesResponse) (string, error) {
	trainingDataset := resourcesRes.Context.Models.GetTrainingDatasets()[name]
	if trainingDataset == nil {
		return "", userconfig.ErrorUndefinedResource(name, resource.TrainingDatasetType)
	}
	dataStatus := resourcesRes.DataStatuses[trainingDataset.ID]
	return dataStatusSummary(dataStatus), nil
}

func describeModel(name string, resourcesRes *schema.GetResourcesResponse) (string, error) {
	model := resourcesRes.Context.Models[name]
	if model == nil {
		return "", userconfig.ErrorUndefinedResource(name, resource.ModelType)
	}
	dataStatus := resourcesRes.DataStatuses[model.ID]
	out := dataStatusSummary(dataStatus)
	out += "\n" + titleStr("configuration") + model.UserConfigStr()
	return out, nil
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
		s.Int32(groupStatus.Requested),
		s.Int32(groupStatus.Available()),
		s.Int32(groupStatus.ReadyUpdated),
		s.Int32(groupStatus.ReadyStaleCompute),
		s.Int32(groupStatus.ReadyStaleModel),
		s.Int32(groupStatus.FailedUpdated),
	}

	headers := []table.Header{
		{Title: "status"},
		{Title: "requested"},
		{Title: "available"},
		{Title: "up-to-date"},
		{Title: "stale compute", Hidden: groupStatus.ReadyStaleCompute == 0},
		{Title: "stale model", Hidden: groupStatus.ReadyStaleModel == 0},
		{Title: "failed", Hidden: groupStatus.FailedUpdated == 0},
	}

	apiEndpoint := urls.Join(resourcesRes.APIsBaseURL, anyAPIStatus.Path)

	out := "\n" + console.Bold("url:  ") + apiEndpoint + "\n"
	out += fmt.Sprintf("%s curl -k -X POST -H \"Content-Type: application/json\" %s -d @samples.json\n\n", console.Bold("curl:"), apiEndpoint)
	out += fmt.Sprintf(console.Bold("updated at:")+" %s\n", libtime.LocalTimestamp(updatedAt))

	t := table.Table{
		Headers: headers,
		Rows:    [][]interface{}{row},
	}
	out += "\n" + table.MustFormat(t)

	out += "\n"
	out += "\n" + apiMetricsTable(ctx.App.Name, api)

	if !flagVerbose {
		return out, nil
	}

	out += "\n\n" + describeModelInput(groupStatus, apiEndpoint)

	if modelName, ok := yaml.ExtractAtSymbolText(api.Model); ok {
		model := ctx.Models[modelName]
		resIDs := strset.New()
		combinedInput := []interface{}{model.Input, model.TrainingInput}
		for _, res := range ctx.ExtractCortexResources(combinedInput, resource.ConstantType, resource.RawColumnType, resource.AggregateType, resource.TransformedColumnType) {
			resIDs.Add(res.GetID())
			resIDs.Merge(ctx.AllComputedResourceDependencies(res.GetID()))
		}
		var samplePlaceholderFields []string
		for rawColumnName, rawColumn := range ctx.RawColumns {
			if resIDs.Has(rawColumn.GetID()) {
				fieldStr := fmt.Sprintf("\"%s\": %s", rawColumnName, rawColumn.GetColumnType().JSONPlaceholder())
				samplePlaceholderFields = append(samplePlaceholderFields, fieldStr)
			}
		}
		sort.Strings(samplePlaceholderFields)
		samplesPlaceholderStr := `{ "samples": [ { ` + strings.Join(samplePlaceholderFields, ", ") + " } ] }"
		out += "\n\n" + console.Bold("payload:  ") + samplesPlaceholderStr
	}

	if api != nil {
		out += "\n" + titleStr("configuration") + api.UserConfigStr()
	}

	return out, nil
}

func apiMetricsTable(appName string, api *context.API) string {
	params := map[string]string{"appName": appName, "apiName": api.Name}
	httpResponse, err := HTTPGet("/metrics", params)
	if err != nil {
		errors.Exit(err)
	}

	var apiMetrics schema.APIMetrics
	err = json.Unmarshal(httpResponse, &apiMetrics)
	if err != nil {
		errors.Exit(err)
	}

	out := networkMetricsTable(&apiMetrics)
	out += "\n\n"
	if api.Tracker == nil {
		out += titleStr("prediction metrics")
		out += "tracker not configured to record predictions"
		return out
	}
	if api.Tracker.ModelType == userconfig.ClassificationModelType {
		out += classificationMetricsTable(&apiMetrics)
	} else {
		out += regressionMetricsTable(&apiMetrics)
	}
	return out
}

func networkMetricsTable(apiMetrics *schema.APIMetrics) string {
	latency := "-"
	if apiMetrics.NetworkStats.Latency != nil {
		latency = fmt.Sprintf("%.9g", *apiMetrics.NetworkStats.Latency)
	}

	t := table.Table{
		Headers: []table.Header{
			{Title: "2XX"},
			{Title: "4XX"},
			{Title: "5XX"},
			{Title: "total"},
			{Title: "avg latency"},
		},
		Rows: [][]interface{}{
			{
				apiMetrics.NetworkStats.Code2XX,
				apiMetrics.NetworkStats.Code4XX,
				apiMetrics.NetworkStats.Code5XX,
				apiMetrics.NetworkStats.Total,
				latency,
			},
		},
	}

	return table.MustFormat(t)
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
		Rows: [][]interface{}{
			{
				minStr, maxStr, avgStr,
			},
		},
	}

	return table.MustFormat(t)
}

func classificationMetricsTable(apiMetrics *schema.APIMetrics) string {
	var out string

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
			headers = append(headers, table.Header{Title: s.TruncateEllipses(className, 17), MaxWidth: 20})
			row = append(row, apiMetrics.ClassDistribution[className])
		}

		t := table.Table{
			Headers: headers,
			Rows:    [][]interface{}{row},
		}

		out += table.MustFormat(t)
	} else {
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
				{Title: "classes", MaxWidth: 40},
				{Title: "count", MaxWidth: 20},
			},
			Rows: rows,
		}

		out += table.MustFormat(t)

		if len(apiMetrics.ClassDistribution) > maxClassesToDisplay {
			out += fmt.Sprintf("\n\nlisting at most %d classes, the complete list can be found in your cloudwatch dashboard", maxClassesToDisplay)
		}
	}
	return out
}

func describeModelInput(groupStatus *resource.APIGroupStatus, apiEndpoint string) string {
	if groupStatus.Available() == 0 {
		return "waiting for api to be ready"
	}

	modelInput, err := getModelInput(urls.Join(apiEndpoint, "signature"))
	if err != nil {
		return "waiting for api to be ready"
	}

	rows := make([][]interface{}, len(modelInput.Signature))
	rowNum := 0
	for inputName, featureSignature := range modelInput.Signature {
		shapeStr := make([]string, len(featureSignature.Shape))
		for idx, dim := range featureSignature.Shape {
			if dim == 0 {
				shapeStr[idx] = "?"
			} else {
				shapeStr[idx] = s.Int(dim)
			}
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
	response, err := makeRequest(req)
	if err != nil {
		return nil, err
	}

	var modelInput schema.ModelInput
	err = json.Unmarshal(response, &modelInput)
	if err != nil {
		return nil, errors.Wrap(err, "unable to parse model input response")
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
	if resourceType == resource.PythonPackageType {
		title = resourceType.UserFacingPlural()
	}

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
		if groupStatus.Requested == 0 {
			continue
		}

		var updatedAt *time.Time
		if groupStatus.ActiveStatus != nil {
			updatedAt = groupStatus.ActiveStatus.Start
		}

		rows = append(rows, []interface{}{
			name,
			groupStatus.Available(),
			groupStatus.ReadyUpdated,
			groupStatus.Requested,
			groupStatus.FailedUpdated,
			libtime.Since(updatedAt),
		})

		totalFailed += int(groupStatus.FailedUpdated)
	}

	t := table.Table{
		Headers: []table.Header{
			{Title: resource.APIType.UserFacing()},
			{Title: "available"},
			{Title: "up-to-date"},
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
	ctx := resourcesRes.Context
	var titles, values []string

	if len(ctx.PythonPackages) != 0 {
		titles = append(titles, resource.PythonPackageType.UserFacingPlural())
		values = append(values, pythonPackageStatusesStr(resourcesRes.DataStatuses, resourcesRes.Context))
	}

	if len(ctx.RawColumns) != 0 {
		titles = append(titles, resource.RawColumnType.UserFacingPlural())
		values = append(values, rawColumnStatusesStr(resourcesRes.DataStatuses, resourcesRes.Context))
	}

	if len(ctx.Aggregates) != 0 {
		titles = append(titles, resource.AggregateType.UserFacingPlural())
		values = append(values, aggregateStatusesStr(resourcesRes.DataStatuses, resourcesRes.Context))
	}

	if len(ctx.TransformedColumns) != 0 {
		titles = append(titles, resource.TransformedColumnType.UserFacingPlural())
		values = append(values, transformedColumnStatusesStr(resourcesRes.DataStatuses, resourcesRes.Context))
	}

	if len(ctx.Models) != 0 {
		titles = append(titles, resource.TrainingDatasetType.UserFacingPlural())
		values = append(values, trainingDatasetStatusesStr(resourcesRes.DataStatuses, resourcesRes.Context))
	}

	if len(ctx.Models) != 0 {
		titles = append(titles, resource.ModelType.UserFacingPlural())
		values = append(values, modelStatusesStr(resourcesRes.DataStatuses, resourcesRes.Context))
	}

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

func pythonPackageStatusesStr(dataStatuses map[string]*resource.DataStatus, ctx *context.Context) string {
	var statuses = make([]resource.Status, len(ctx.PythonPackages))
	i := 0
	for _, pythonPackage := range ctx.PythonPackages {
		statuses[i] = dataStatuses[pythonPackage.GetID()]
		i++
	}
	return StatusStr(statuses)
}

func rawColumnStatusesStr(dataStatuses map[string]*resource.DataStatus, ctx *context.Context) string {
	var statuses = make([]resource.Status, len(ctx.RawColumns))
	i := 0
	for _, rawColumn := range ctx.RawColumns {
		statuses[i] = dataStatuses[rawColumn.GetID()]
		i++
	}
	return StatusStr(statuses)
}

func aggregateStatusesStr(dataStatuses map[string]*resource.DataStatus, ctx *context.Context) string {
	var statuses = make([]resource.Status, len(ctx.Aggregates))
	i := 0
	for _, aggregate := range ctx.Aggregates {
		statuses[i] = dataStatuses[aggregate.GetID()]
		i++
	}
	return StatusStr(statuses)
}

func transformedColumnStatusesStr(dataStatuses map[string]*resource.DataStatus, ctx *context.Context) string {
	var statuses = make([]resource.Status, len(ctx.TransformedColumns))
	i := 0
	for _, transformedColumn := range ctx.TransformedColumns {
		statuses[i] = dataStatuses[transformedColumn.GetID()]
		i++
	}
	return StatusStr(statuses)
}

func trainingDatasetStatusesStr(dataStatuses map[string]*resource.DataStatus, ctx *context.Context) string {
	var statuses = make([]resource.Status, len(ctx.Models))
	i := 0
	for _, model := range ctx.Models {
		statuses[i] = dataStatuses[model.Dataset.GetID()]
		i++
	}
	return StatusStr(statuses)
}

func modelStatusesStr(dataStatuses map[string]*resource.DataStatus, ctx *context.Context) string {
	var statuses = make([]resource.Status, len(ctx.Models))
	i := 0
	for _, model := range ctx.Models {
		statuses[i] = dataStatuses[model.GetID()]
		i++
	}
	return StatusStr(statuses)
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
