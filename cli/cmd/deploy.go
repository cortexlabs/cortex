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
	"github.com/cortexlabs/cortex/pkg/lib/zip"
)

var flagDeployForce bool

func init() {
	deployCmd.PersistentFlags().BoolVarP(&flagDeployForce, "force", "f", false, "stop all running jobs")
	addEnvFlag(deployCmd)
}

var deployCmd = &cobra.Command{
	Use:   "deploy",
	Short: "deploy an application",
	Long:  "Deploy an application.",
	Args:  cobra.NoArgs,
	Run: func(cmd *cobra.Command, args []string) {
		deploy(flagDeployForce, false)
	},
}

func deploy(force bool, ignoreCache bool) {
	root := mustAppRoot()
	_, err := appNameFromConfig() // Check proper app.yaml
	if err != nil {
		errors.Exit(err)
	}

	zipInput := &zip.Input{
		FileLists: []zip.FileListInput{
			{
				Sources:      allConfigPaths(root),
				RemovePrefix: root,
			},
		},
	}

	params := map[string]string{
		"environment": flagEnv,
		"force":       s.Bool(force),
		"ignoreCache": s.Bool(ignoreCache),
	}

	response, err := HTTPUploadZip("/deploy", zipInput, "config.zip", params)
	if err != nil {
		errors.Exit(err)
	}

	var deployResponse schema.DeployResponse
	if err := json.Unmarshal(response, &deployResponse); err != nil {
		errors.Exit(err, "/deploy", "response", s.ErrUnmarshalJSON, string(response))
	}

	fmt.Println(deployResponse.Message)
}
