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
)

var flagRefreshForce bool

func init() {
	refreshCmd.PersistentFlags().BoolVarP(&flagRefreshForce, "force", "f", false, "stop all running jobs")
	addEnvFlag(refreshCmd)
}

var refreshCmd = &cobra.Command{
	Use:   "refresh",
	Short: "deploy and override existing deployment",
	Long: `This command sends all project configuration and code to Cortex.
If validations pass, Cortex will attempt to create the desired state,
and override the existing deployment.`,
	Args: cobra.NoArgs,
	Run: func(cmd *cobra.Command, args []string) {
		deploy(flagRefreshForce, true)
	},
}
