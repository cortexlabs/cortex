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
	"os"
	"sort"
	"strings"

	"github.com/spf13/cobra"

	"github.com/cortexlabs/cortex/pkg/api/context"
	"github.com/cortexlabs/cortex/pkg/api/resource"
	"github.com/cortexlabs/cortex/pkg/api/schema"
	"github.com/cortexlabs/cortex/pkg/utils/errors"
)

func init() {
	addAppNameFlag(statusCmd)
	addEnvFlag(statusCmd)
	addResourceTypesToHelp(statusCmd)
}

var statusCmd = &cobra.Command{
	Use:   "status [RESOURCE_TYPE] [RESOURCE_NAME]",
	Short: "get resource statuses",
	Long:  "Get resource statuses.",
	Args:  cobra.RangeArgs(0, 2),
	Run: func(cmd *cobra.Command, args []string) {
		resourceName, resourceTypeStr := "", ""
		switch len(args) {
		case 0:
			resourcesRes, err := getResourcesResponse()
			if err != nil {
				errors.Exit(err)
			}
			printAllResourceStatuses(resourcesRes)
			os.Exit(0)
		case 1:
			resourceName = args[0]
		case 2:
			userResourceType := args[0]
			resourceName = args[1]

			if userResourceType != "" {
				resourceType, err := resource.VisibleResourceTypeFromPrefix(userResourceType)
				if err != nil {
					errors.Exit(err)
				}

				resourceTypeStr = resourceType.String()
			}
		}

		appName, err := AppNameFromFlagOrConfig()
		if err != nil {
			errors.Exit(err)
		}
		StreamLogs(appName, resourceName, resourceTypeStr, false)
	},
}

func printAllResourceStatuses(resourcesRes *schema.GetResourcesResponse) {
	fmt.Println("")
	printPythonPackageStatuses(resourcesRes.DataStatuses, resourcesRes.Context)
	printRawFeatureStatuses(resourcesRes.DataStatuses, resourcesRes.Context)
	printAggregateStatuses(resourcesRes.DataStatuses, resourcesRes.Context)
	printTransformedFeatureStatuses(resourcesRes.DataStatuses, resourcesRes.Context)
	printTrainingDatasetStatuses(resourcesRes.DataStatuses, resourcesRes.Context)
	printModelStatuses(resourcesRes.DataStatuses, resourcesRes.Context)
	printAPIStatuses(resourcesRes.APIGroupStatuses)
}

func printPythonPackageStatuses(dataStatuses map[string]*resource.DataStatus, ctx *context.Context) {
	var statuses = make([]resource.Status, len(ctx.PythonPackages))
	i := 0
	for _, pythonPackage := range ctx.PythonPackages {
		statuses[i] = dataStatuses[pythonPackage.GetID()]
		i++
	}
	fmt.Println("Python Packages:        " + StatusStr(statuses))
}

func printRawFeatureStatuses(dataStatuses map[string]*resource.DataStatus, ctx *context.Context) {
	var statuses = make([]resource.Status, len(ctx.RawFeatures))
	i := 0
	for _, rawFeature := range ctx.RawFeatures {
		statuses[i] = dataStatuses[rawFeature.GetID()]
		i++
	}
	fmt.Println("Raw Features:           " + StatusStr(statuses))
}

func printAggregateStatuses(dataStatuses map[string]*resource.DataStatus, ctx *context.Context) {
	var statuses = make([]resource.Status, len(ctx.Aggregates))
	i := 0
	for _, aggregate := range ctx.Aggregates {
		statuses[i] = dataStatuses[aggregate.GetID()]
		i++
	}
	fmt.Println("Aggregates:             " + StatusStr(statuses))
}

func printTransformedFeatureStatuses(dataStatuses map[string]*resource.DataStatus, ctx *context.Context) {
	var statuses = make([]resource.Status, len(ctx.TransformedFeatures))
	i := 0
	for _, transformedFeature := range ctx.TransformedFeatures {
		statuses[i] = dataStatuses[transformedFeature.GetID()]
		i++
	}
	fmt.Println("Transformed Features:   " + StatusStr(statuses))
}

func printTrainingDatasetStatuses(dataStatuses map[string]*resource.DataStatus, ctx *context.Context) {
	var statuses = make([]resource.Status, len(ctx.Models))
	i := 0
	for _, model := range ctx.Models {
		statuses[i] = dataStatuses[model.Dataset.GetID()]
		i++
	}
	fmt.Println("Training Datasets:      " + StatusStr(statuses))
}

func printModelStatuses(dataStatuses map[string]*resource.DataStatus, ctx *context.Context) {
	var statuses = make([]resource.Status, len(ctx.Models))
	i := 0
	for _, model := range ctx.Models {
		statuses[i] = dataStatuses[model.GetID()]
		i++
	}
	fmt.Println("Models:                 " + StatusStr(statuses))
}

func printAPIStatuses(apiGroupStatuses map[string]*resource.APIGroupStatus) {
	var statuses = make([]resource.Status, len(apiGroupStatuses))
	i := 0
	for _, apiGroupStatus := range apiGroupStatuses {
		statuses[i] = apiGroupStatus
		i++
	}
	fmt.Println("APIs:                   " + StatusStr(statuses))
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
