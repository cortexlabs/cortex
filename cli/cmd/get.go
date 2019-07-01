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
	"sort"
	"strings"
	"time"

	"github.com/cortexlabs/yaml"
	"github.com/spf13/cobra"

	"github.com/cortexlabs/cortex/pkg/lib/errors"
	"github.com/cortexlabs/cortex/pkg/lib/json"
	"github.com/cortexlabs/cortex/pkg/lib/msgpack"
	"github.com/cortexlabs/cortex/pkg/lib/sets/strset"
	"github.com/cortexlabs/cortex/pkg/lib/slices"
	s "github.com/cortexlabs/cortex/pkg/lib/strings"
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

		return resourceByNameStr(resourceNameOrType, resourcesRes)

	case 2:
		userResourceType := args[0]
		resourceName := args[1]
		resourceType, err := resource.VisibleResourceTypeFromPrefix(userResourceType)
		if err != nil {
			return "", resource.ErrorInvalidType(userResourceType)
		}
		return resourceByNameAndTypeStr(resourceName, resourceType, resourcesRes)
	}

	return "", errors.New("too many args") // unexpected
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

func resourceByNameStr(resourceName string, resourcesRes *schema.GetResourcesResponse) (string, error) {
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
		return describeAPI(resourceName, resourcesRes)
	default:
		return "", resource.ErrorInvalidType(resourceType.String())
	}
}

func resourcesByTypeStr(resourceType resource.Type, resourcesRes *schema.GetResourcesResponse) (string, error) {
	switch resourceType {
	case resource.PythonPackageType:
		return pythonPackagesStr(resourcesRes.DataStatuses, resourcesRes.Context, false), nil
	case resource.RawColumnType:
		return rawColumnsStr(resourcesRes.DataStatuses, resourcesRes.Context, false), nil
	case resource.AggregateType:
		return aggregatesStr(resourcesRes.DataStatuses, resourcesRes.Context, false), nil
	case resource.TransformedColumnType:
		return transformedColumnsStr(resourcesRes.DataStatuses, resourcesRes.Context, false), nil
	case resource.TrainingDatasetType:
		return trainingDataStr(resourcesRes.DataStatuses, resourcesRes.Context, false), nil
	case resource.ModelType:
		return modelsStr(resourcesRes.DataStatuses, resourcesRes.Context, false), nil
	case resource.APIType:
		return apisStr(resourcesRes.APIGroupStatuses, false), nil
	default:
		return "", resource.ErrorInvalidType(resourceType.String())
	}
}

func resourceByNameAndTypeStr(resourceName string, resourceType resource.Type, resourcesRes *schema.GetResourcesResponse) (string, error) {
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
		return describeAPI(resourceName, resourcesRes)
	default:
		return "", resource.ErrorInvalidType(resourceType.String())
	}
}

func allResourcesStr(resourcesRes *schema.GetResourcesResponse) string {
	ctx := resourcesRes.Context

	showTitle := false
	if slices.AreNGreaterThanZero(2,
		len(ctx.PythonPackages), len(ctx.RawColumns), len(ctx.Aggregates),
		len(ctx.TransformedColumns), len(ctx.Models), len(ctx.Models), // Models occurs twice because of training datasets
		len(resourcesRes.APIGroupStatuses),
	) {
		showTitle = true
	}

	out := ""
	out += pythonPackagesStr(resourcesRes.DataStatuses, ctx, showTitle)
	out += rawColumnsStr(resourcesRes.DataStatuses, ctx, showTitle)
	out += aggregatesStr(resourcesRes.DataStatuses, ctx, showTitle)
	out += transformedColumnsStr(resourcesRes.DataStatuses, ctx, showTitle)
	out += trainingDataStr(resourcesRes.DataStatuses, ctx, showTitle)
	out += modelsStr(resourcesRes.DataStatuses, ctx, showTitle)
	out += apisStr(resourcesRes.APIGroupStatuses, showTitle)
	return out
}

func pythonPackagesStr(dataStatuses map[string]*resource.DataStatus, ctx *context.Context, showTitle bool) string {
	if len(ctx.PythonPackages) == 0 {
		return ""
	}

	strings := make(map[string]string)
	for name, pythonPackage := range ctx.PythonPackages {
		strings[name] = dataResourceRow(name, pythonPackage, dataStatuses)
	}

	title := "\n"
	if showTitle {
		title = titleStr("Python Packages")
	}
	return title + dataResourcesHeader() + strMapToStr(strings)
}

func rawColumnsStr(dataStatuses map[string]*resource.DataStatus, ctx *context.Context, showTitle bool) string {
	if len(ctx.RawColumns) == 0 {
		return ""
	}

	strings := make(map[string]string)
	for name, rawColumn := range ctx.RawColumns {
		strings[name] = dataResourceRow(name, rawColumn, dataStatuses)
	}

	title := "\n"
	if showTitle {
		title = titleStr("Raw Columns")
	}
	return title + dataResourcesHeader() + strMapToStr(strings)
}

func aggregatesStr(dataStatuses map[string]*resource.DataStatus, ctx *context.Context, showTitle bool) string {
	if len(ctx.Aggregates) == 0 {
		return ""
	}

	strings := make(map[string]string)
	for name, aggregate := range ctx.Aggregates {
		strings[name] = dataResourceRow(name, aggregate, dataStatuses)
	}

	title := "\n"
	if showTitle {
		title = titleStr("Aggregates")
	}
	return title + dataResourcesHeader() + strMapToStr(strings)
}

func transformedColumnsStr(dataStatuses map[string]*resource.DataStatus, ctx *context.Context, showTitle bool) string {
	if len(ctx.TransformedColumns) == 0 {
		return ""
	}

	strings := make(map[string]string)
	for name, transformedColumn := range ctx.TransformedColumns {
		strings[name] = dataResourceRow(name, transformedColumn, dataStatuses)
	}

	title := "\n"
	if showTitle {
		title = titleStr("Transformed Columns")
	}
	return title + dataResourcesHeader() + strMapToStr(strings)
}

func trainingDataStr(dataStatuses map[string]*resource.DataStatus, ctx *context.Context, showTitle bool) string {
	if len(ctx.Models) == 0 {
		return ""
	}

	strings := make(map[string]string)
	for _, model := range ctx.Models {
		name := model.Dataset.Name
		strings[name] = dataResourceRow(name, model.Dataset, dataStatuses)
	}

	title := "\n"
	if showTitle {
		title = titleStr("Training Datasets")
	}
	return title + dataResourcesHeader() + strMapToStr(strings)
}

func modelsStr(dataStatuses map[string]*resource.DataStatus, ctx *context.Context, showTitle bool) string {
	if len(ctx.Models) == 0 {
		return ""
	}

	strings := make(map[string]string)
	for name, model := range ctx.Models {
		strings[name] = dataResourceRow(name, model, dataStatuses)
	}

	title := "\n"
	if showTitle {
		title = titleStr("Models")
	}
	return title + dataResourcesHeader() + strMapToStr(strings)
}

func apisStr(apiGroupStatuses map[string]*resource.APIGroupStatus, showTitle bool) string {
	if len(apiGroupStatuses) == 0 {
		return ""
	}

	strings := make(map[string]string)
	for name, apiGroupStatus := range apiGroupStatuses {
		strings[name] = apiResourceRow(apiGroupStatus)
	}

	title := "\n"
	if showTitle {
		title = titleStr("APIs")
	}
	return title + apisHeader() + strMapToStr(strings)
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
	out += titleStr("Configuration") + rawColumn.UserConfigStr()
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

	out += titleStr("Configuration") + aggregate.UserConfigStr()
	return out, nil
}

func describeTransformedColumn(name string, resourcesRes *schema.GetResourcesResponse) (string, error) {
	transformedColumn := resourcesRes.Context.TransformedColumns[name]
	if transformedColumn == nil {
		return "", userconfig.ErrorUndefinedResource(name, resource.TransformedColumnType)
	}
	dataStatus := resourcesRes.DataStatuses[transformedColumn.ID]
	out := dataStatusSummary(dataStatus)
	out += titleStr("Configuration") + transformedColumn.UserConfigStr()
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
	out += titleStr("Configuration") + model.UserConfigStr()
	return out, nil
}

func describeAPI(name string, resourcesRes *schema.GetResourcesResponse) (string, error) {
	groupStatus := resourcesRes.APIGroupStatuses[name]
	if groupStatus == nil {
		return "", userconfig.ErrorUndefinedResource(name, resource.APIType)
	}

	ctx := resourcesRes.Context
	api := ctx.APIs[name]

	var staleReplicas int32
	var ctxAPIStatus *resource.APIStatus
	var anyAPIStatus *resource.APIStatus
	for _, apiStatus := range resourcesRes.APIStatuses {
		if apiStatus.APIName != name {
			continue
		}
		anyAPIStatus = apiStatus
		if api != nil && apiStatus.ResourceID == api.ID {
			ctxAPIStatus = apiStatus
		}
		staleReplicas += apiStatus.TotalStaleReady()
	}

	out := titleStr("Summary")
	out += "Status:            " + groupStatus.Message() + "\n"
	if ctxAPIStatus != nil {
		out += fmt.Sprintf("Updated replicas:  %d ready\n", ctxAPIStatus.ReadyUpdated)
	}
	if staleReplicas != 0 {
		out += fmt.Sprintf("Stale replicas:    %d ready\n", staleReplicas)
	}
	out += "Created at:        " + libtime.LocalTimestamp(groupStatus.Start) + "\n"
	if groupStatus.ActiveStatus != nil && groupStatus.ActiveStatus.Start != nil {
		out += "Refreshed at:      " + libtime.LocalTimestamp(groupStatus.ActiveStatus.Start) + "\n"
	}

	out += titleStr("Endpoint")
	out += "URL:      " + urls.Join(resourcesRes.APIsBaseURL, anyAPIStatus.Path) + "\n"
	out += "Method:   POST\n"
	out += `Header:   "Content-Type: application/json"` + "\n"

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
		out += "Payload:  " + samplesPlaceholderStr + "\n"
	}
	if api != nil {
		out += titleStr("Configuration") + api.UserConfigStr()
	}

	return out, nil
}

func dataStatusSummary(dataStatus *resource.DataStatus) string {
	out := titleStr("Summary")
	out += "Status:               " + dataStatus.Message() + "\n"
	out += "Workload started at:  " + libtime.LocalTimestamp(dataStatus.Start) + "\n"
	out += "Workload ended at:    " + libtime.LocalTimestamp(dataStatus.End) + "\n"
	return out
}

func valueStr(value interface{}) string {
	return titleStr("Value") + s.Obj(value) + "\n"
}

func strMapToStr(strings map[string]string) string {
	var keys []string
	for key := range strings {
		keys = append(keys, key)
	}

	sort.Strings(keys)

	out := ""
	for _, key := range keys {
		out += strings[key] + "\n"
	}

	return out
}

func dataResourceRow(name string, resource context.Resource, dataStatuses map[string]*resource.DataStatus) string {
	dataStatus := dataStatuses[resource.GetID()]
	return resourceRow(name, dataStatus.Message(), dataStatus.End)
}

func apiResourceRow(groupStatus *resource.APIGroupStatus) string {
	var updatedAt *time.Time
	if groupStatus.ActiveStatus != nil {
		updatedAt = groupStatus.ActiveStatus.Start
	}
	return resourceRow(groupStatus.APIName, groupStatus.Message(), updatedAt)
}

func resourceRow(name string, status string, startTime *time.Time) string {
	if len(name) > 33 {
		name = name[0:30] + "..."
	}
	if len(status) > 23 {
		status = status[0:20] + "..."
	}
	timeSince := libtime.Since(startTime)
	return stringifyRow(name, status, timeSince)
}

func dataResourcesHeader() string {
	return stringifyRow("NAME", "STATUS", "AGE") + "\n"
}

func apisHeader() string {
	return stringifyRow("NAME", "STATUS", "LAST UPDATE") + "\n"
}

func stringifyRow(name string, status string, timeSince string) string {
	return fmt.Sprintf("%-35s%-24s%s", name, status, timeSince)
}

func titleStr(title string) string {
	titleLength := len(title)
	top := strings.Repeat("-", titleLength)
	bottom := strings.Repeat("-", titleLength)
	return "\n" + top + "\n" + title + "\n" + bottom + "\n\n"
}

func resourceStatusesStr(resourcesRes *schema.GetResourcesResponse) string {
	ctx := resourcesRes.Context
	var titles, values []string

	if len(ctx.PythonPackages) != 0 {
		titles = append(titles, "Python Packages")
		values = append(values, pythonPackageStatusesStr(resourcesRes.DataStatuses, resourcesRes.Context))
	}

	if len(ctx.RawColumns) != 0 {
		titles = append(titles, "Raw Columns")
		values = append(values, rawColumnStatusesStr(resourcesRes.DataStatuses, resourcesRes.Context))
	}

	if len(ctx.Aggregates) != 0 {
		titles = append(titles, "Aggregates")
		values = append(values, aggregateStatusesStr(resourcesRes.DataStatuses, resourcesRes.Context))
	}

	if len(ctx.TransformedColumns) != 0 {
		titles = append(titles, "Transformed Columns")
		values = append(values, transformedColumnStatusesStr(resourcesRes.DataStatuses, resourcesRes.Context))
	}

	if len(ctx.Models) != 0 {
		titles = append(titles, "Training Datasets")
		values = append(values, trainingDatasetStatusesStr(resourcesRes.DataStatuses, resourcesRes.Context))
	}

	if len(ctx.Models) != 0 {
		titles = append(titles, "Models")
		values = append(values, modelStatusesStr(resourcesRes.DataStatuses, resourcesRes.Context))
	}

	if len(resourcesRes.APIGroupStatuses) != 0 {
		titles = append(titles, "APIs")
		values = append(values, apiStatusesStr(resourcesRes.APIGroupStatuses))
	}

	maxTitleLen := s.MaxLen(titles...)

	out := "\n"
	for i, title := range titles {
		paddingWidth := maxTitleLen - len(title) + 3
		padding := strings.Repeat(" ", paddingWidth)
		out += title + ":" + padding + values[i] + "\n"
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
