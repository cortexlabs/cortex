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

	"github.com/cortexlabs/cortex/pkg/lib/console"
	"github.com/cortexlabs/cortex/pkg/lib/exit"
	"github.com/cortexlabs/cortex/pkg/lib/json"
	"github.com/cortexlabs/cortex/pkg/lib/prompt"
	s "github.com/cortexlabs/cortex/pkg/lib/strings"
	"github.com/cortexlabs/cortex/pkg/lib/telemetry"
	"github.com/cortexlabs/cortex/pkg/operator/schema"
	"github.com/spf13/cobra"
)

var _flagKeepCache bool
var _flagDeleteForce bool

func init() {
	_deleteCmd.PersistentFlags().BoolVarP(&_flagKeepCache, "keep-cache", "c", false, "keep cached data for the deployment")
	_deleteCmd.PersistentFlags().BoolVarP(&_flagDeleteForce, "force", "f", false, "delete the api without confirmation")
	addEnvFlag(_deleteCmd)
}

var _deleteCmd = &cobra.Command{
	Use:   "delete API_NAME",
	Short: "delete an api",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		telemetry.Event("cli.delete")
		delete(args[0], _flagKeepCache)
	},
}

func delete(apiName string, keepCache bool) {
	if !_flagDeleteForce {
		readyReplicas := getReadyReplicasOrNil(apiName)
		if readyReplicas != nil && *readyReplicas > 2 {
			prompt.YesOrExit(fmt.Sprintf("are you sure you want to delete the %s api (which has %d running replicas)?", apiName, *readyReplicas), "")
		}
	}

	params := map[string]string{
		"apiName":   apiName,
		"keepCache": s.Bool(keepCache),
	}

	httpRes, err := HTTPPostJSONData("/delete", nil, params)
	if err != nil {
		exit.Error(err)
	}

	var deleteRes schema.DeleteResponse
	err = json.Unmarshal(httpRes, &deleteRes)
	if err != nil {
		exit.Error(err, "/delete", string(httpRes))
	}

	fmt.Println(console.Bold(deleteRes.Message))
}

func getReadyReplicasOrNil(apiName string) *int32 {
	httpRes, err := HTTPGet("/get/" + apiName)
	if err != nil {
		return nil
	}

	var apiRes schema.GetAPIResponse
	if err = json.Unmarshal(httpRes, &apiRes); err != nil {
		return nil
	}

	totalReady := apiRes.Status.Updated.Ready + apiRes.Status.Stale.Ready
	return &totalReady
}
