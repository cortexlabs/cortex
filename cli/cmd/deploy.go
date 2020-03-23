/*
Copyright 2020 Cortex Labs, Inc.

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
	"path"
	"path/filepath"

	"github.com/cortexlabs/cortex/pkg/lib/errors"
	"github.com/cortexlabs/cortex/pkg/lib/exit"
	"github.com/cortexlabs/cortex/pkg/lib/files"
	"github.com/cortexlabs/cortex/pkg/lib/json"
	"github.com/cortexlabs/cortex/pkg/lib/print"
	"github.com/cortexlabs/cortex/pkg/lib/prompt"
	s "github.com/cortexlabs/cortex/pkg/lib/strings"
	"github.com/cortexlabs/cortex/pkg/lib/telemetry"
	"github.com/cortexlabs/cortex/pkg/lib/zip"
	"github.com/cortexlabs/cortex/pkg/operator/schema"
	"github.com/spf13/cobra"
)

var _warningFileBytes = 1024 * 1024 * 10
var _warningProjectBytes = 1024 * 1024 * 10
var _warningFileCount = 100

var _flagDeployForce bool
var _flagDeployYes bool

func init() {
	_deployCmd.PersistentFlags().BoolVarP(&_flagDeployForce, "force", "f", false, "override the in-progress api update")
	_deployCmd.PersistentFlags().BoolVarP(&_flagDeployYes, "yes", "y", false, "skip prompts")
	addEnvFlag(_deployCmd)
}

var _deployCmd = &cobra.Command{
	Use:   "deploy [CONFIG_FILE]",
	Short: "create or update apis",
	Args:  cobra.RangeArgs(0, 1),
	Run: func(cmd *cobra.Command, args []string) {
		telemetry.Event("cli.deploy")

		configPath := getConfigPath(args)
		deploy(configPath, _flagDeployForce)
	},
}

func getConfigPath(args []string) string {
	var configPath string

	if len(args) == 0 {
		configPath = "cortex.yaml"
		if !files.IsFile(configPath) {
			exit.Error(ErrorCortexYAMLNotFound())
		}
	} else {
		configPath = args[0]
		if err := files.CheckFile(configPath); err != nil {
			exit.Error(err)
		}
	}

	return configPath
}

func deploy(configPath string, force bool) {
	params := map[string]string{
		"force":      s.Bool(force),
		"configPath": configPath,
	}

	configBytes, err := files.ReadFileBytes(configPath)
	if err != nil {
		exit.Error(err)
	}

	uploadBytes := map[string][]byte{
		"config": configBytes,
	}

	projectRoot := filepath.Dir(files.UserRelToAbsPath(configPath))

	ignoreFns := []files.IgnoreFn{
		files.IgnoreSpecificFiles(files.UserRelToAbsPath(configPath)),
		files.IgnoreCortexDebug,
		files.IgnoreHiddenFiles,
		files.IgnoreHiddenFolders,
		files.IgnorePythonGeneratedFiles,
	}

	cortexIgnorePath := path.Join(projectRoot, ".cortexignore")
	if files.IsFile(cortexIgnorePath) {
		cortexIgnore, err := files.GitIgnoreFn(cortexIgnorePath)
		if err != nil {
			exit.Error(err)
		}
		ignoreFns = append(ignoreFns, cortexIgnore)
	}

	if !_flagDeployYes {
		ignoreFns = append(ignoreFns, files.PromptForFilesAboveSize(_warningFileBytes, "do you want to upload %s (%s)?"))
	}

	projectPaths, err := files.ListDirRecursive(projectRoot, false, ignoreFns...)
	if err != nil {
		exit.Error(err)
	}

	canSkipPromptMsg := "you can skip this prompt next time with `cortex deploy --yes`\n"
	rootDirMsg := "this directory"
	if s.EnsureSuffix(projectRoot, "/") != _cwd {
		rootDirMsg = fmt.Sprintf("./%s", files.DirPathRelativeToCWD(projectRoot))
	}

	didPromptFileCount := false
	if !_flagDeployYes && len(projectPaths) >= _warningFileCount {
		msg := fmt.Sprintf("cortex will zip %d files in %s and upload them to the cluster, though we recommend you upload large files (e.g. models) to s3 and download them in your api's __init__ function. Would you like to continue?", len(projectPaths), rootDirMsg)
		prompt.YesOrExit(msg, canSkipPromptMsg, "")
		didPromptFileCount = true
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

	if !_flagDeployYes && !didPromptFileCount && len(projectZipBytes) >= _warningProjectBytes {
		msg := fmt.Sprintf("cortex will zip %d files in %s (%s) and upload them to the cluster, though we recommend you upload large files (e.g. models) to s3 and download them in your api's __init__ function. Would you like to continue?", len(projectPaths), rootDirMsg, s.IntToBase2Byte(len(projectZipBytes)))
		prompt.YesOrExit(msg, canSkipPromptMsg, "")
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

	print.ForUserSplitDoubleNewLine(deployResponse.Message)
}
