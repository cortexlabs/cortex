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
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"

	cr "github.com/cortexlabs/cortex/pkg/lib/configreader"
	"github.com/cortexlabs/cortex/pkg/lib/console"
	"github.com/cortexlabs/cortex/pkg/lib/errors"
	"github.com/cortexlabs/cortex/pkg/lib/exit"
	"github.com/cortexlabs/cortex/pkg/lib/files"
	"github.com/cortexlabs/cortex/pkg/lib/json"
	"github.com/cortexlabs/cortex/pkg/lib/prompt"
	s "github.com/cortexlabs/cortex/pkg/lib/strings"
	"github.com/cortexlabs/cortex/pkg/lib/telemetry"
	"github.com/cortexlabs/cortex/pkg/lib/zip"
	"github.com/cortexlabs/cortex/pkg/operator/api/schema"
)

var _warningFileSize = 1024 * 1024 * 10
var flagDeployForce bool
var flagDeployRefresh bool

func init() {
	deployCmd.PersistentFlags().BoolVarP(&flagDeployForce, "force", "f", false, "override the in-progress deployment update")
	deployCmd.PersistentFlags().BoolVarP(&flagDeployRefresh, "refresh", "r", false, "re-deploy all apis with cleared cache and rolling updates")
	addEnvFlag(deployCmd)
}

var deployCmd = &cobra.Command{
	Use:   "deploy",
	Short: "create or update a deployment",
	Args:  cobra.NoArgs,
	Run: func(cmd *cobra.Command, args []string) {
		telemetry.EventNotify("cli.deploy")
		deploy(flagDeployForce, flagDeployRefresh)
	},
}

func PromptForFilesAboveSize(size int) files.IgnoreFn {
	return func(path string, fi os.FileInfo) (bool, error) {
		if !fi.IsDir() && fi.Size() > int64(size) {
			prompt.YesOrExit(fmt.Sprintf("are you sure you want to zip %s (%s)?", path, s.IntToBase2Byte(int(fi.Size()))), "error: cancelled deployment")
		}
		return false, nil
	}
}

func deploy(force bool, ignoreCache bool) {
	root := mustAppRoot()
	_, err := readConfig() // Check proper cortex.yaml
	if err != nil {
		exit.Error(err)
	}

	params := map[string]string{
		"force":       s.Bool(force),
		"ignoreCache": s.Bool(ignoreCache),
	}

	configBytes, err := ioutil.ReadFile(filepath.Join(root, "cortex.yaml"))
	if err != nil {
		exit.Error(errors.Wrap(err, "cortex.yaml", cr.ErrorReadConfig().Error()))
	}

	uploadBytes := map[string][]byte{
		"cortex.yaml": configBytes,
	}
	fmt.Println("zipping files in current working directory")
	projectPaths, err := files.ListDirRecursive(root, false,
		files.IgnoreCortexYAML,
		files.IgnoreCortexDebug,
		files.IgnoreHiddenFiles,
		files.IgnoreHiddenFolders,
		files.IgnorePythonGeneratedFiles,
		PromptForFilesAboveSize(_warningFileSize),
	)
	if err != nil {
		exit.Error(err)
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
		exit.Error(errors.Wrap(err, "failed to zip project folder"))
	}

	uploadBytes["project.zip"] = projectZipBytes

	uploadInput := &HTTPUploadInput{
		Bytes: uploadBytes,
	}

	response, err := HTTPUpload("/deploy", uploadInput, params)
	if err != nil {
		exit.Error(err)
	}

	var deployResponse schema.DeployResponse
	if err := json.Unmarshal(response, &deployResponse); err != nil {
		exit.Error(err, "/deploy", string(response))
	}

	msgParts := strings.Split(deployResponse.Message, "\n\n")
	fmt.Println(console.Bold(msgParts[0]))
	if len(msgParts) > 1 {
		fmt.Println("\n" + strings.Join(msgParts[1:], "\n\n"))
	}
}
