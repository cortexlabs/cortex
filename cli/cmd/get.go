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
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"github.com/cortexlabs/cortex/pkg/api/context"
	"github.com/cortexlabs/cortex/pkg/api/resource"
	"github.com/cortexlabs/cortex/pkg/api/schema"
	s "github.com/cortexlabs/cortex/pkg/api/strings"
	"github.com/cortexlabs/cortex/pkg/api/userconfig"
	"github.com/cortexlabs/cortex/pkg/utils/errors"
	"github.com/cortexlabs/cortex/pkg/utils/util"
)

func init() {
	addAppNameFlag(getCmd)
	addEnvFlag(getCmd)
	addWatchFlag(getCmd)
	addResourceTypesToHelp(getCmd)
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
		return allResourcesStr(resourcesRes), nil

	case 1:
		resourceNameOrType := args[0]

		resourceType, err := resource.VisibleResourceTypeFromPrefix(resourceNameOrType)
		if err != nil {
			if rerr, ok := err.(resource.ResourceError); ok && rerr.Kind != resource.ErrInvalidType {
				return "", err
			}
		} else {
			return resourcesByTypeStr(resourceType, resourcesRes)
		}

		if _, err = resourcesRes.Context.VisibleResourceByName(resourceNameOrType); err != nil {
			if rerr, ok := err.(resource.ResourceError); ok && rerr.Kind == resource.ErrNameNotFound {
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
		return "\n" + pythonPackagesStr(resourcesRes.DataStatuses, resourcesRes.Context), nil
	case resource.RawColumnType:
		return "\n" + rawColumnsStr(resourcesRes.DataStatuses, resourcesRes.Context), nil
	case resource.AggregateType:
		return "\n" + aggregatesStr(resourcesRes.DataStatuses, resourcesRes.Context), nil
	case resource.TransformedColumnType:
		return "\n" + transformedColumnsStr(resourcesRes.DataStatuses, resourcesRes.Context), nil
	case resource.TrainingDatasetType:
		return "\n" + trainingDataStr(resourcesRes.DataStatuses, resourcesRes.Context), nil
	case resource.ModelType:
		return "\n" + modelsStr(resourcesRes.DataStatuses, resourcesRes.Context), nil
	case resource.APIType:
		return "\n" + apisStr(resourcesRes.APIGroupStatuses), nil
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
	out := ""
	out += titleStr("Python Packages")
	out += pythonPackagesStr(resourcesRes.DataStatuses, resourcesRes.Context)
	out += titleStr("Raw Columns")
	out += rawColumnsStr(resourcesRes.DataStatuses, resourcesRes.Context)
	out += titleStr("Aggregates")
	out += aggregatesStr(resourcesRes.DataStatuses, resourcesRes.Context)
	out += titleStr("Transformed Columns")
	out += transformedColumnsStr(resourcesRes.DataStatuses, resourcesRes.Context)
	out += titleStr("Training Datasets")
	out += trainingDataStr(resourcesRes.DataStatuses, resourcesRes.Context)
	out += titleStr("Models")
	out += modelsStr(resourcesRes.DataStatuses, resourcesRes.Context)
	out += titleStr("APIs")
	out += apisStr(resourcesRes.APIGroupStatuses)
	return out
}

func pythonPackagesStr(dataStatuses map[string]*resource.DataStatus, ctx *context.Context) string {
	if len(ctx.PythonPackages) == 0 {
		return "None\n"
	}

	strings := make(map[string]string)
	for name, pythonPackage := range ctx.PythonPackages {
		strings[name] = dataResourceRow(name, pythonPackage, dataStatuses)
	}
	return dataResourcesHeader() + strMapToStr(strings)
}

func rawColumnsStr(dataStatuses map[string]*resource.DataStatus, ctx *context.Context) string {
	if len(ctx.RawColumns) == 0 {
		return "None\n"
	}

	strings := make(map[string]string)
	for name, rawColumn := range ctx.RawColumns {
		strings[name] = dataResourceRow(name, rawColumn, dataStatuses)
	}
	return dataResourcesHeader() + strMapToStr(strings)
}

func aggregatesStr(dataStatuses map[string]*resource.DataStatus, ctx *context.Context) string {
	if len(ctx.Aggregates) == 0 {
		return "None\n"
	}

	strings := make(map[string]string)
	for name, aggregate := range ctx.Aggregates {
		strings[name] = dataResourceRow(name, aggregate, dataStatuses)
	}
	return dataResourcesHeader() + strMapToStr(strings)
}

func transformedColumnsStr(dataStatuses map[string]*resource.DataStatus, ctx *context.Context) string {
	if len(ctx.TransformedColumns) == 0 {
		return "None\n"
	}

	strings := make(map[string]string)
	for name, transformedColumn := range ctx.TransformedColumns {
		strings[name] = dataResourceRow(name, transformedColumn, dataStatuses)
	}
	return dataResourcesHeader() + strMapToStr(strings)
}

func trainingDataStr(dataStatuses map[string]*resource.DataStatus, ctx *context.Context) string {
	if len(ctx.Models) == 0 {
		return "None\n"
	}

	strings := make(map[string]string)
	for _, model := range ctx.Models {
		name := model.Dataset.Name
		strings[name] = dataResourceRow(name, model.Dataset, dataStatuses)
	}
	return dataResourcesHeader() + strMapToStr(strings)
}

func modelsStr(dataStatuses map[string]*resource.DataStatus, ctx *context.Context) string {
	if len(ctx.Models) == 0 {
		return "None\n"
	}

	strings := make(map[string]string)
	for name, model := range ctx.Models {
		strings[name] = dataResourceRow(name, model, dataStatuses)
	}
	return dataResourcesHeader() + strMapToStr(strings)
}

func apisStr(apiGroupStatuses map[string]*resource.APIGroupStatus) string {
	if len(apiGroupStatuses) == 0 {
		return "None\n"
	}

	strings := make(map[string]string)
	for name, apiGroupStatus := range apiGroupStatuses {
		strings[name] = apiResourceRow(apiGroupStatus)
	}
	return apisHeader() + strMapToStr(strings)
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
	out += resourceStr(rawColumn.GetUserConfig())
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
			return "", errors.Wrap(err, "/aggregate", "response", s.ErrUnmarshalJson, string(httpResponse))
		}

		obj, err := util.UnmarshalMsgpackToInterface(aggregateRes.Value)
		if err != nil {
			return "", errors.Wrap(err, "/aggregate", "response", s.ErrUnmarshalMsgpack)
		}
		out += valueStr(obj)
	}

	out += resourceStr(aggregate.Aggregate)
	return out, nil
}

func describeTransformedColumn(name string, resourcesRes *schema.GetResourcesResponse) (string, error) {
	transformedColumn := resourcesRes.Context.TransformedColumns[name]
	if transformedColumn == nil {
		return "", userconfig.ErrorUndefinedResource(name, resource.TransformedColumnType)
	}
	dataStatus := resourcesRes.DataStatuses[transformedColumn.ID]
	out := dataStatusSummary(dataStatus)
	out += resourceStr(transformedColumn.TransformedColumn)
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
	out += resourceStr(model.Model)
	return out, nil
}

func describeAPI(name string, resourcesRes *schema.GetResourcesResponse) (string, error) {
	groupStatus := resourcesRes.APIGroupStatuses[name]
	if groupStatus == nil {
		return "", userconfig.ErrorUndefinedResource(name, resource.APIType)
	}

	api := resourcesRes.Context.APIs[name]

	var staleReplicas int32 = 0
	var ctxAPIStatus *resource.APIStatus = nil
	for _, apiStatus := range resourcesRes.APIStatuses {
		if apiStatus.APIName != name {
			continue
		}
		if api != nil && apiStatus.ResourceID == api.ID {
			ctxAPIStatus = apiStatus
		}
		staleReplicas += apiStatus.TotalStaleReady()
	}

	out := titleStr("Summary")

	if groupStatus.ActiveStatus != nil {
		out += "Endpoint:          " + util.URLJoin(resourcesRes.APIsBaseURL, groupStatus.ActiveStatus.Path) + "\n"
	}

	out += "Status:            " + groupStatus.Message() + "\n"
	out += "Created at:        " + util.LocalTimestamp(groupStatus.Start) + "\n"

	if groupStatus.ActiveStatus != nil && groupStatus.ActiveStatus.Start != nil {
		out += "Refreshed at:      " + util.LocalTimestamp(groupStatus.ActiveStatus.Start) + "\n"
	}

	if ctxAPIStatus != nil {
		out += fmt.Sprintf("Updated replicas:  %d/%d ready\n", ctxAPIStatus.ReadyUpdated, ctxAPIStatus.RequestedReplicas)
	}

	if staleReplicas != 0 {
		out += fmt.Sprintf("Stale replicas:    %d ready\n", staleReplicas)
	}

	if api != nil {
		out += resourceStr(api.API)
	}

	return out, nil
}

func dataStatusSummary(dataStatus *resource.DataStatus) string {
	out := titleStr("Summary")
	out += "Status:               " + dataStatus.Message() + "\n"
	out += "Workload started at:  " + util.LocalTimestamp(dataStatus.Start) + "\n"
	out += "Workload ended at:    " + util.LocalTimestamp(dataStatus.End) + "\n"
	return out
}

func resourceStr(resource userconfig.Resource) string {
	return titleStr("Configuration") + s.Obj(resource) + "\n"
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
	var updatedAt *time.Time = nil
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
	timeSince := util.TimeSince(startTime)
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
