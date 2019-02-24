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

	"github.com/spf13/cobra"

	"github.com/cortexlabs/cortex/pkg/api/schema"
	s "github.com/cortexlabs/cortex/pkg/api/strings"
	"github.com/cortexlabs/cortex/pkg/lib/errors"
)

var flagKeepCache bool

func init() {
	deleteCmd.PersistentFlags().BoolVarP(&flagKeepCache, "keep-cache", "c", false, "keep cached data for the app")
	addEnvFlag(deleteCmd)
}

var deleteCmd = &cobra.Command{
	Use:   "delete [APP_NAME]",
	Short: "delete a deployment",
	Long:  "Delete a deployment.",
	Args:  cobra.MaximumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		var appName string
		var err error
		if len(args) == 1 {
			appName = args[0]
		} else {
			appName, err = appNameFromConfig()
			if err != nil {
				errors.Exit(err)
			}
		}

		params := map[string]string{
			"appName":   appName,
			"keepCache": s.Bool(flagKeepCache),
		}
		httpResponse, err := HTTPPostJSONData("/delete", nil, params)
		if err != nil {
			errors.Exit(err)
		}

		var deleteResponse schema.DeleteResponse
		err = json.Unmarshal(httpResponse, &deleteResponse)
		if err != nil {
			errors.Exit(err, "/delete", "response", s.ErrUnmarshalJSON, string(httpResponse))
		}
		fmt.Println(deleteResponse.Message)
	},
}
