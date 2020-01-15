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
	"path/filepath"
	"strings"

	"github.com/cortexlabs/cortex/pkg/lib/console"
	"github.com/cortexlabs/cortex/pkg/lib/errors"
	"github.com/cortexlabs/cortex/pkg/lib/exit"
	"github.com/cortexlabs/cortex/pkg/lib/files"
	"github.com/cortexlabs/cortex/pkg/lib/json"
	s "github.com/cortexlabs/cortex/pkg/lib/strings"
	"github.com/cortexlabs/cortex/pkg/lib/telemetry"
	"github.com/cortexlabs/cortex/pkg/lib/zip"
	"github.com/cortexlabs/cortex/pkg/operator/schema"
	"github.com/spf13/cobra"
)

var _maxProjectSize = 1024 * 1024 * 50
var _flagDeployForce bool
var _flagRefresh bool

func init() {
	_deployCmd.PersistentFlags().BoolVarP(&_flagDeployForce, "force", "f", false, "override the in-progress deployment update")
	_deployCmd.PersistentFlags().BoolVarP(&_flagRefresh, "refresh", "r", false, "re-deploy all apis with cleared cache and rolling updates")
	addEnvFlag(_deployCmd)
}

var _deployCmd = &cobra.Command{
	Use:   "deploy [CONFIG_FILE]",
	Short: "create or update a deployment",
	Args:  cobra.RangeArgs(0, 1),
	Run: func(cmd *cobra.Command, args []string) {
		telemetry.Event("cli.deploy")

		configPath := getConfigPath(args)
		deploy(configPath, _flagDeployForce, _flagRefresh)
	},
}

func getConfigPath(args []string) string {
	var configPath string

	if len(args) == 0 {
		configPath = "cortex.yaml"
		if !files.IsFile(configPath) {
			exit.Error("no api config file was specified, and ./cortex.yaml does not exits; create cortex.yaml, or reference an existing config file by running `cortex deploy <config_file_path>`")
		}
	} else {
		configPath = args[0]
		if !files.IsFile(configPath) {
			exit.Error("no such file", configPath)
		}
	}

	return configPath
}

func deploy(configPath string, force bool, refresh bool) {
	params := map[string]string{
		"force":      s.Bool(force),
		"refresh":    s.Bool(refresh),
		"configPath": configPath,
	}

	configBytes, err := ioutil.ReadFile(configPath)
	if err != nil {
		exit.Error(errors.Wrap(err, configPath))
	}

	uploadBytes := map[string][]byte{
		"config": configBytes,
	}

	projectRoot := filepath.Dir(files.UserRelToAbsPath(configPath))

	projectPaths, err := files.ListDirRecursive(projectRoot, false,
		files.IgnoreSpecificFiles(files.UserRelToAbsPath(configPath)),
		files.IgnoreCortexDebug,
		files.IgnoreHiddenFiles,
		files.IgnoreHiddenFolders,
		files.IgnorePythonGeneratedFiles,
	)
	if err != nil {
		exit.Error(err)
	}

	projectZipBytes, err := zip.ToMem(&zip.Input{
		FileLists: []zip.FileListInput{
			{
				Sources:      projectPaths,
				RemovePrefix: projectRoot,
			},
		},
	})

	if err != nil {
		exit.Error(errors.Wrap(err, "failed to zip project folder"))
	}

	if len(projectZipBytes) > _maxProjectSize {
		exit.Error(errors.New("zipped project folder exceeds " + s.Int(_maxProjectSize) + " bytes"))
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
