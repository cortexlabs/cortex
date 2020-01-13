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

	// "github.com/cortexlabs/cortex/pkg/lib/console"
	// "github.com/cortexlabs/cortex/pkg/lib/exit"
	// "github.com/cortexlabs/cortex/pkg/lib/json"
	// "github.com/cortexlabs/cortex/pkg/lib/prompt"
	// s "github.com/cortexlabs/cortex/pkg/lib/strings"
	"github.com/cortexlabs/cortex/pkg/lib/telemetry"
	// "github.com/cortexlabs/cortex/pkg/operator/schema"
)

var _flagKeepCache bool

func init() {
	deleteCmd.PersistentFlags().BoolVarP(&_flagKeepCache, "keep-cache", "c", false, "keep cached data for the deployment")
	addEnvFlag(deleteCmd)
}

var deleteCmd = &cobra.Command{
	Use:   "delete API_NAME",
	Short: "delete an API",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		telemetry.Event("cli.delete")
		// delete(args[0], _flagKeepCache)
	},
}

// func delete(apiName string, keepCache bool) {
// 	apiName := args[0]

// 	apiInfo, err := getAPIInfo(apiName)
// 	if err != nil {
// 		exit.Error(err)
// 	}

// 	if apiInfo.APIGroupStatus.Requested > 2 {
// 		prompt.YesOrExit(fmt.Sprintf("are you sure you want to delete the %s API?", apiName), "")
// 	}

// 	params := map[string]string{
// 		"apiName":   apiName,
// 		"keepCache": s.Bool(keepCache),
// 	}
// 	httpResponse, err := HTTPPostJSONData("/delete", nil, params)
// 	if err != nil {
// 		exit.Error(err)
// 	}

// 	var deleteResponse schema.DeleteResponse
// 	err = json.Unmarshal(httpResponse, &deleteResponse)
// 	if err != nil {
// 		exit.Error(err, "/delete", string(httpResponse))
// 	}

// 	fmt.Println(console.Bold(deleteResponse.Message))
// }
