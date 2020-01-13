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

	"github.com/cortexlabs/cortex/pkg/lib/telemetry"
)

func init() {
	addAppNameFlag(logsCmd)
	addEnvFlag(logsCmd)
}

var logsCmd = &cobra.Command{
	Use:   "logs API_NAME",
	Short: "stream logs from an api",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		telemetry.Event("cli.logs")

		// resourceName := args[0]

		// appName, err := AppNameFromFlagOrConfig()
		// if err != nil {
		// 	exit.Error(err)
		// }

		// err = StreamLogs(appName, resourceName, resource.APIType.String())
		// if err != nil {
		// 	// note: if modifying this string, search the codebase for it and change all occurrences
		// 	// TODO
		// 	if strings.HasSuffix(err.Error(), "is not deployed") {
		// 		fmt.Println(console.Bold(err.Error()))
		// 		return
		// 	}
		// 	exit.Error(err)
		// }
	},
}
