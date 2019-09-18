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
	"io/ioutil"

	"github.com/spf13/cobra"

	"github.com/cortexlabs/cortex/pkg/lib/console"
	"github.com/cortexlabs/cortex/pkg/lib/errors"
	"github.com/cortexlabs/cortex/pkg/lib/files"
	"github.com/cortexlabs/cortex/pkg/lib/json"
	s "github.com/cortexlabs/cortex/pkg/lib/strings"
	"github.com/cortexlabs/cortex/pkg/lib/zip"
	"github.com/cortexlabs/cortex/pkg/operator/api/schema"
	"github.com/cortexlabs/cortex/pkg/operator/api/userconfig"
)

var MaxProjectSize = 1024 * 1024 * 50
var flagDeployForce bool

func init() {
	deployCmd.PersistentFlags().BoolVarP(&flagDeployForce, "force", "f", false, "stop all running jobs")
	addEnvFlag(deployCmd)
}

var deployCmd = &cobra.Command{
	Use:   "deploy",
	Short: "create or update a deployment",
	Long:  "This command sends all deployment configuration and code to Cortex. If validations pass, Cortex will attempt to create the desired state on the cluster.",
	Args:  cobra.NoArgs,
	Run: func(cmd *cobra.Command, args []string) {
		deploy(flagDeployForce, false)
	},
}

func deploy(force bool, ignoreCache bool) {
	root := mustAppRoot()
	config, err := readConfig() // Check proper cortex.yaml
	if err != nil {
		errors.Exit(err)
	}

	params := map[string]string{
		"force":       s.Bool(force),
		"ignoreCache": s.Bool(ignoreCache),
	}

	configBytes, err := ioutil.ReadFile("cortex.yaml")
	if err != nil {
		errors.Exit(errors.Wrap(err, "cortex.yaml", userconfig.ErrorReadConfig().Error()))
	}

	uploadBytes := map[string][]byte{
		"cortex.yaml": configBytes,
	}

	if config.AreProjectFilesRequired() {
		projectPaths, err := files.ListDirRecursive(root, false,
			files.IgnoreCortexYAML,
			files.IgnoreHiddenFiles,
			files.IgnoreHiddenFolders,
			files.IgnorePythonGeneratedFiles,
		)
		if err != nil {
			errors.Exit(err)
		}

		projectZipBytes, err := zip.ToMem(&zip.Input{
			FileLists: []zip.FileListInput{
				{
					Sources:      projectPaths,
					RemovePrefix: root,
				},
			},
		})

		if err != nil {
			errors.Exit(errors.Wrap(err, "failed to zip project folder"))
		}

		if len(projectZipBytes) > MaxProjectSize {
			errors.Exit(errors.New("zipped project folder exceeds " + s.Int(MaxProjectSize) + " bytes"))
		}

		uploadBytes["project.zip"] = projectZipBytes
	}

	uploadInput := &HTTPUploadInput{
		Bytes: uploadBytes,
	}

	response, err := HTTPUpload("/deploy", uploadInput, params)
	if err != nil {
		errors.Exit(err)
	}

	var deployResponse schema.DeployResponse
	if err := json.Unmarshal(response, &deployResponse); err != nil {
		errors.Exit(err, "/deploy", "response", string(response))
	}

	fmt.Println("\n" + console.Bold(deployResponse.Message))
}
