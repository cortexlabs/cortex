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
	"strings"

	"github.com/cortexlabs/cortex/pkg/lib/console"
	"github.com/cortexlabs/cortex/pkg/lib/errors"
	"github.com/cortexlabs/cortex/pkg/lib/exit"
	"github.com/cortexlabs/cortex/pkg/lib/json"
	s "github.com/cortexlabs/cortex/pkg/lib/strings"
	"github.com/cortexlabs/cortex/pkg/lib/telemetry"
	"github.com/cortexlabs/cortex/pkg/operator/schema"
	"github.com/spf13/cobra"
)

var _flagRefreshForce bool

func init() {
	_refreshCmd.PersistentFlags().BoolVarP(&_flagRefreshForce, "force", "f", false, "override the in-progress deployment update")
	addEnvFlag(_refreshCmd)
}

var _refreshCmd = &cobra.Command{
	Use:   "refresh API_NAME",
	Short: "restart all replicas for an api (witout downtime)",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		telemetry.Event("cli.refresh")
		refresh(args[0], _flagRefreshForce)
	},
}

func refresh(apiName string, force bool) {
	params := map[string]string{
		"force": s.Bool(force),
	}

	httpRes, err := HTTPPostNoBody("/refresh/"+apiName, params)
	if err != nil {
		// note: if modifying this string, search the codebase for it and change all occurrences
		if strings.HasSuffix(errors.Message(err), "is not deployed") || strings.Contains(errors.Message(err), "--force") {
			fmt.Println(console.Bold(errors.Message(err)))
			exit.ErrorNoPrintNoTelemetry()
		}
		exit.Error(err)
	}

	var refreshRes schema.RefreshResponse
	err = json.Unmarshal(httpRes, &refreshRes)
	if err != nil {
		exit.Error(err, "/refresh", string(httpRes))
	}

	fmt.Println(console.Bold(refreshRes.Message))
}
