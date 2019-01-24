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
	"os"
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
	rootCmd.AddCommand(getCmd)
	addAppNameFlag(getCmd)
	addEnvFlag(getCmd)
	addResourceTypesToHelp(getCmd)
}

var getCmd = &cobra.Command{
	Use:   "get [RESOURCE_TYPE] [RESOURCE_NAME]",
	Short: "get information about resources",
	Long:  "Get information about resources.",
	Args:  cobra.RangeArgs(0, 2),
	Run: func(cmd *cobra.Command, args []string) {
		resourcesRes, err := getResourcesResponse()
		if err != nil {
			errors.Exit(err)
		}

		switch len(args) {
		case 0:
			printAllResources(resourcesRes)
			os.Exit(0)

		case 1:
			resourceNameOrType := args[0]

			resourceType, err := resource.VisibleResourceTypeFromPrefix(resourceNameOrType)
			if err != nil {
				if rerr, ok := err.(resource.ResourceError); ok && rerr.Kind != resource.ErrInvalidType {
					errors.Exit(err)
				}
			} else {
				printResourcesByType(resourceType, resourcesRes)
				os.Exit(0)
			}

			if _, err = resourcesRes.Context.VisibleResourceByName(resourceNameOrType); err != nil {
				if rerr, ok := err.(resource.ResourceError); ok && rerr.Kind == resource.ErrNotFound {
					errors.Exit(errors.New(s.ErrUndefinedNameOrType(resourceNameOrType)))
				}
				errors.Exit(err)
			}

			printResourceByName(resourceNameOrType, resourcesRes)
			os.Exit(0)

		case 2:
			userResourceType := args[0]
			resourceName := args[1]
			resourceType, err := resource.VisibleResourceTypeFromPrefix(userResourceType)
			if err != nil {
				errors.Exit(resource.ErrorInvalidType(userResourceType))
			}
			printResourceByNameAndType(resourceName, resourceType, resourcesRes)
		}
	},
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

func printResourceByName(resourceName string, resourcesRes *schema.GetResourcesResponse) {
	rs, err := resourcesRes.Context.VisibleResourceByName(resourceName)
	if err != nil {
		errors.Exit(err)
	}
	switch resourceType := rs.GetResourceType(); resourceType {
	case resource.RawFeatureType:
		describeRawFeature(resourceName, resourcesRes)
	case resource.AggregateType:
		describeAggregate(resourceName, resourcesRes)
	case resource.TransformedFeatureType:
		describeTransformedFeature(resourceName, resourcesRes)
	case resource.TrainingDatasetType:
		describeTrainingDataset(resourceName, resourcesRes)
	case resource.ModelType:
		describeModel(resourceName, resourcesRes)
	case resource.APIType:
		describeAPI(resourceName, resourcesRes)
	default:
		fmt.Println("Unknown resource type: ", resourceType)
	}
}

func printResourcesByType(resourceType resource.Type, resourcesRes *schema.GetResourcesResponse) {
	switch resourceType {
	case resource.RawFeatureType:
		printRawFeatures(resourcesRes.DataStatuses, resourcesRes.Context)
	case resource.AggregateType:
		printAggregates(resourcesRes.DataStatuses, resourcesRes.Context)
	case resource.TransformedFeatureType:
		printTransformedFeatures(resourcesRes.DataStatuses, resourcesRes.Context)
	case resource.TrainingDatasetType:
		printTrainingData(resourcesRes.DataStatuses, resourcesRes.Context)
	case resource.ModelType:
		printModels(resourcesRes.DataStatuses, resourcesRes.Context)
	case resource.APIType:
		printAPIs(resourcesRes.APIGroupStatuses)
	default:
		fmt.Println("Unknown resource type: ", resourceType)
	}
}

func printResourceByNameAndType(resourceName string, resourceType resource.Type, resourcesRes *schema.GetResourcesResponse) {
	switch resourceType {
	case resource.RawFeatureType:
		describeRawFeature(resourceName, resourcesRes)
	case resource.AggregateType:
		describeAggregate(resourceName, resourcesRes)
	case resource.TransformedFeatureType:
		describeTransformedFeature(resourceName, resourcesRes)
	case resource.TrainingDatasetType:
		describeTrainingDataset(resourceName, resourcesRes)
	case resource.ModelType:
		describeModel(resourceName, resourcesRes)
	case resource.APIType:
		describeAPI(resourceName, resourcesRes)
	default:
		fmt.Println("Unknown resource type: ", resourceType)
	}
}

func printAllResources(resourcesRes *schema.GetResourcesResponse) {
	printTitle("Raw Features")
	printRawFeatures(resourcesRes.DataStatuses, resourcesRes.Context)

	printTitle("Aggregates")
	printAggregates(resourcesRes.DataStatuses, resourcesRes.Context)

	printTitle("Transformed Features")
	printTransformedFeatures(resourcesRes.DataStatuses, resourcesRes.Context)

	printTitle("Training Datasets")
	printTrainingData(resourcesRes.DataStatuses, resourcesRes.Context)

	printTitle("Models")
	printModels(resourcesRes.DataStatuses, resourcesRes.Context)

	printTitle("APIs")
	printAPIs(resourcesRes.APIGroupStatuses)
}

func printRawFeatures(dataStatuses map[string]*resource.DataStatus, ctx *context.Context) {
	if len(ctx.RawFeatures) == 0 {
		fmt.Println("None")
		return
	}

	printDataResourcesHeader()
	strings := make(map[string]string)
	for name, rawFeature := range ctx.RawFeatures {
		strings[name] = dataResourceRow(name, rawFeature, dataStatuses)
	}
	printStrings(strings)
}

func printAggregates(dataStatuses map[string]*resource.DataStatus, ctx *context.Context) {
	if len(ctx.Aggregates) == 0 {
		fmt.Println("None")
		return
	}

	printDataResourcesHeader()
	strings := make(map[string]string)
	for name, aggregate := range ctx.Aggregates {
		strings[name] = dataResourceRow(name, aggregate, dataStatuses)
	}
	printStrings(strings)
}

func printTransformedFeatures(dataStatuses map[string]*resource.DataStatus, ctx *context.Context) {
	if len(ctx.TransformedFeatures) == 0 {
		fmt.Println("None")
		return
	}

	printDataResourcesHeader()
	strings := make(map[string]string)
	for name, transformedFeature := range ctx.TransformedFeatures {
		strings[name] = dataResourceRow(name, transformedFeature, dataStatuses)
	}
	printStrings(strings)
}

func printTrainingData(dataStatuses map[string]*resource.DataStatus, ctx *context.Context) {
	if len(ctx.Models) == 0 {
		fmt.Println("None")
		return
	}

	printDataResourcesHeader()
	strings := make(map[string]string)
	for _, model := range ctx.Models {
		name := model.Dataset.Name
		strings[name] = dataResourceRow(name, model.Dataset, dataStatuses)
	}
	printStrings(strings)
}

func printModels(dataStatuses map[string]*resource.DataStatus, ctx *context.Context) {
	if len(ctx.Models) == 0 {
		fmt.Println("None")
		return
	}

	printDataResourcesHeader()
	strings := make(map[string]string)
	for name, model := range ctx.Models {
		strings[name] = dataResourceRow(name, model, dataStatuses)
	}
	printStrings(strings)
}

func printAPIs(apiGroupStatuses map[string]*resource.APIGroupStatus) {
	if len(apiGroupStatuses) == 0 {
		fmt.Println("None")
		return
	}

	printAPIsHeader()
	strings := make(map[string]string)
	for name, apiGroupStatus := range apiGroupStatuses {
		strings[name] = apiResourceRow(apiGroupStatus)
	}
	printStrings(strings)
}

func describeRawFeature(name string, resourcesRes *schema.GetResourcesResponse) {
	rawFeature := resourcesRes.Context.RawFeatures[name]
	if rawFeature == nil {
		fmt.Println("Raw feature " + name + " does not exist")
		os.Exit(0)
	}
	dataStatus := resourcesRes.DataStatuses[rawFeature.GetID()]
	printDataStatusSummary(dataStatus)
	printResource(rawFeature.GetUserConfig())
}

func describeAggregate(name string, resourcesRes *schema.GetResourcesResponse) {
	aggregate := resourcesRes.Context.Aggregates[name]
	if aggregate == nil {
		fmt.Println("Aggregate " + name + " does not exist")
		os.Exit(0)
	}
	dataStatus := resourcesRes.DataStatuses[aggregate.ID]
	printDataStatusSummary(dataStatus)

	if dataStatus.ExitCode == resource.ExitCodeDataSucceeded {
		params := map[string]string{"appName": resourcesRes.Context.App.Name}
		httpResponse, err := HTTPGet("/aggregate/"+aggregate.ID, params)
		if err != nil {
			errors.Exit(err)
		}

		var aggregateRes schema.GetAggregateResponse
		err = json.Unmarshal(httpResponse, &aggregateRes)
		if err != nil {
			errors.Exit(err, "/aggregate", "response", s.ErrUnmarshalJson, string(httpResponse))
		}

		obj, err := util.UnmarshalMsgpackToInterface(aggregateRes.Value)
		if err != nil {
			errors.Exit(err, "/aggregate", "response", s.ErrUnmarshalMsgpack)
		}
		printValue(s.Obj(obj))
	}

	printResource(aggregate.Aggregate)
}

func describeTransformedFeature(name string, resourcesRes *schema.GetResourcesResponse) {
	transformedFeature := resourcesRes.Context.TransformedFeatures[name]
	if transformedFeature == nil {
		fmt.Println("Transformed feature " + name + " does not exist")
		os.Exit(0)
	}
	dataStatus := resourcesRes.DataStatuses[transformedFeature.ID]
	printDataStatusSummary(dataStatus)
	printResource(transformedFeature.TransformedFeature)
}

func describeTrainingDataset(name string, resourcesRes *schema.GetResourcesResponse) {
	trainingDataset := resourcesRes.Context.Models.GetTrainingDatasets()[name]

	if trainingDataset == nil {
		fmt.Println("Training dataset " + name + " does not exist")
		os.Exit(0)
	}
	dataStatus := resourcesRes.DataStatuses[trainingDataset.ID]
	printDataStatusSummary(dataStatus)
}

func describeModel(name string, resourcesRes *schema.GetResourcesResponse) {
	model := resourcesRes.Context.Models[name]
	if model == nil {
		fmt.Println("Model " + name + " does not exist")
		os.Exit(0)
	}
	dataStatus := resourcesRes.DataStatuses[model.ID]
	printDataStatusSummary(dataStatus)
	printResource(model.Model)
}

func describeAPI(name string, resourcesRes *schema.GetResourcesResponse) {
	groupStatus := resourcesRes.APIGroupStatuses[name]
	if groupStatus == nil {
		fmt.Println("API " + name + " does not exist")
		os.Exit(0)
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

	printTitle("Summary")

	if groupStatus.ActiveStatus != nil {
		fmt.Println("Endpoint:        ", util.URLJoin(resourcesRes.APIsBaseURL, groupStatus.ActiveStatus.Path))
	}

	fmt.Println("Status:          ", groupStatus.Message())
	fmt.Println("Created at:      ", util.LocalTimestamp(groupStatus.Start))

	if groupStatus.ActiveStatus != nil && groupStatus.ActiveStatus.Start != nil {
		fmt.Println("Refreshed at:    ", util.LocalTimestamp(groupStatus.ActiveStatus.Start))
	}

	if ctxAPIStatus != nil {
		fmt.Printf("Updated replicas: %d/%d ready\n", ctxAPIStatus.ReadyUpdated, ctxAPIStatus.RequestedReplicas)
	}

	if staleReplicas != 0 {
		fmt.Printf("Stale replicas:   %d ready\n", staleReplicas)
	}

	if api != nil {
		printResource(api.API)
	}
}

func printResource(resource userconfig.Resource) {
	printTitle("Configuration")

	fmt.Println(s.Obj(resource))
}

func printDataStatusSummary(dataStatus *resource.DataStatus) {
	printTitle("Summary")

	fmt.Println("Status:              ", dataStatus.Message())
	fmt.Println("Workload started at: ", util.LocalTimestamp(dataStatus.Start))
	fmt.Println("Workload ended at:   ", util.LocalTimestamp(dataStatus.End))
}

func printValue(value string) {
	printTitle("Value")

	fmt.Println(value)
}

func printStrings(strings map[string]string) {
	keys := make([]string, len(strings))
	for key := range strings {
		keys = append(keys, key)
	}

	sort.Strings(keys)

	for _, key := range keys {
		fmt.Printf(strings[key])
	}
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

func printDataResourcesHeader() {
	fmt.Printf(stringifyRow("NAME", "STATUS", "AGE"))
}

func printAPIsHeader() {
	fmt.Printf(stringifyRow("NAME", "STATUS", "LAST UPDATE"))
}

func stringifyRow(name string, status string, timeSince string) string {
	return fmt.Sprintf("%-35s%-24s%s\n", name, status, timeSince)
}

func printTitle(title string) {
	titleLength := len(title)
	top := strings.Repeat("-", titleLength)
	bottom := strings.Repeat("-", titleLength)

	fmt.Printf("\n")
	fmt.Println(top)
	fmt.Println(title)
	fmt.Println(bottom)
	fmt.Printf("\n")
}
