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

	"github.com/spf13/cobra"

	"github.com/cortexlabs/cortex/pkg/api/context"
	"github.com/cortexlabs/cortex/pkg/api/resource"
	"github.com/cortexlabs/cortex/pkg/api/schema"
)

func init() {
	addAppNameFlag(statusCmd)
	addEnvFlag(statusCmd)
	addWatchFlag(statusCmd)
	addResourceTypesToHelp(statusCmd)
}

var statusCmd = &cobra.Command{
	Use:   "status [RESOURCE_TYPE] [RESOURCE_NAME]",
	Short: "get resource statuses",
	Long:  "Get resource statuses.",
	Args:  cobra.RangeArgs(0, 2),
	Run: func(cmd *cobra.Command, args []string) {
		rerun(func() (string, error) {
			return runStatus(cmd, args)
		})
	},
}

func runStatus(cmd *cobra.Command, args []string) (string, error) {
	resourceName, resourceTypeStr := "", ""
	switch len(args) {
	case 0:
		resourcesRes, err := getResourcesResponse()
		if err != nil {
			return "", err
		}
		return resourceStatusesStr(resourcesRes), nil
	case 1:
		resourceName = args[0]
	case 2:
		userResourceType := args[0]
		resourceName = args[1]

		if userResourceType != "" {
			resourceType, err := resource.VisibleResourceTypeFromPrefix(userResourceType)
			if err != nil {
				return "", err
			}

			resourceTypeStr = resourceType.String()
		}
	}

	appName, err := AppNameFromFlagOrConfig()
	if err != nil {
		return "", err
	}

	err = StreamLogs(appName, resourceName, resourceTypeStr, false)
	if err != nil {
		return "", err
	}

	return "", nil
}

func resourceStatusesStr(resourcesRes *schema.GetResourcesResponse) string {
	out := "\n"
	out += rawColumnStatusesStr(resourcesRes.DataStatuses, resourcesRes.Context) + "\n"
	out += aggregateStatusesStr(resourcesRes.DataStatuses, resourcesRes.Context) + "\n"
	out += transformedColumnStatusesStr(resourcesRes.DataStatuses, resourcesRes.Context) + "\n"
	out += trainingDatasetStatusesStr(resourcesRes.DataStatuses, resourcesRes.Context) + "\n"
	out += modelStatusesStr(resourcesRes.DataStatuses, resourcesRes.Context) + "\n"
	out += apiStatusesStr(resourcesRes.APIGroupStatuses)
	return out
}

func rawColumnStatusesStr(dataStatuses map[string]*resource.DataStatus, ctx *context.Context) string {
	var statuses = make([]resource.Status, len(ctx.RawColumns))
	i := 0
	for _, rawColumn := range ctx.RawColumns {
		statuses[i] = dataStatuses[rawColumn.GetID()]
		i++
	}
	return "Raw Columns:           " + StatusStr(statuses)
}

func aggregateStatusesStr(dataStatuses map[string]*resource.DataStatus, ctx *context.Context) string {
	var statuses = make([]resource.Status, len(ctx.Aggregates))
	i := 0
	for _, aggregate := range ctx.Aggregates {
		statuses[i] = dataStatuses[aggregate.GetID()]
		i++
	}
	return "Aggregates:            " + StatusStr(statuses)
}

func transformedColumnStatusesStr(dataStatuses map[string]*resource.DataStatus, ctx *context.Context) string {
	var statuses = make([]resource.Status, len(ctx.TransformedColumns))
	i := 0
	for _, transformedColumn := range ctx.TransformedColumns {
		statuses[i] = dataStatuses[transformedColumn.GetID()]
		i++
	}
	return "Transformed Columns:   " + StatusStr(statuses)
}

func trainingDatasetStatusesStr(dataStatuses map[string]*resource.DataStatus, ctx *context.Context) string {
	var statuses = make([]resource.Status, len(ctx.Models))
	i := 0
	for _, model := range ctx.Models {
		statuses[i] = dataStatuses[model.Dataset.GetID()]
		i++
	}
	return "Training Datasets:     " + StatusStr(statuses)
}

func modelStatusesStr(dataStatuses map[string]*resource.DataStatus, ctx *context.Context) string {
	var statuses = make([]resource.Status, len(ctx.Models))
	i := 0
	for _, model := range ctx.Models {
		statuses[i] = dataStatuses[model.GetID()]
		i++
	}
	return "Models:                " + StatusStr(statuses)
}

func apiStatusesStr(apiGroupStatuses map[string]*resource.APIGroupStatus) string {
	var statuses = make([]resource.Status, len(apiGroupStatuses))
	i := 0
	for _, apiGroupStatus := range apiGroupStatuses {
		statuses[i] = apiGroupStatus
		i++
	}
	return "APIs:                  " + StatusStr(statuses)
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
