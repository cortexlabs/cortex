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

	"github.com/spf13/cobra"

	"github.com/cortexlabs/cortex/pkg/consts"
	"github.com/cortexlabs/cortex/pkg/lib/errors"
	"github.com/cortexlabs/cortex/pkg/lib/json"
	"github.com/cortexlabs/cortex/pkg/operator/api/schema"
)

func init() {
	addEnvFlag(versionCmd)
}

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "print the version of the Cortex CLI and cluster",
	Long:  `This command prints the version of the Cortex CLI and cluster`,
	Args:  cobra.NoArgs,
	Run: func(cmd *cobra.Command, args []string) {
		httpResponse, err := HTTPGet("/info")
		if err != nil {
			errors.Exit(err)
		}
		var infoResponse schema.InfoResponse
		err = json.Unmarshal(httpResponse, &infoResponse)
		if err != nil {
			errors.Exit(err, "/info", string(httpResponse))
		}

		fmt.Println("CLI version:     " + consts.CortexVersion)
		fmt.Println("Cluster version: " + infoResponse.ClusterConfig.APIVersion)
	},
}
