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
	"github.com/spf13/cobra"

	"github.com/cortexlabs/cortex/pkg/lib/errors"
	"github.com/cortexlabs/cortex/pkg/operator/api/resource"
)

func init() {
	addAppNameFlag(logsCmd)
	addVerboseFlag(logsCmd)
	// addResourceTypesToHelp(logsCmd)
}

var logsCmd = &cobra.Command{
	Use:   "logs [RESOURCE_TYPE] RESOURCE_NAME",
	Short: "get logs for a resource",
	Long:  "Get logs for a resource.",
	Args:  cobra.RangeArgs(1, 2),
	Run: func(cmd *cobra.Command, args []string) {
		resourceName, resourceTypeStr := "", ""
		switch len(args) {
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

		err = StreamLogs(appName, resourceName, resourceTypeStr, flagVerbose)
		if err != nil {
			errors.Exit(err)
		}
	},
}
