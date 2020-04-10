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
	"strings"

	"github.com/cortexlabs/cortex/cli/cluster"
	"github.com/cortexlabs/cortex/cli/local"
	"github.com/cortexlabs/cortex/pkg/lib/errors"
	"github.com/cortexlabs/cortex/pkg/lib/exit"
	"github.com/cortexlabs/cortex/pkg/lib/files"
	"github.com/cortexlabs/cortex/pkg/lib/pointer"
	"github.com/cortexlabs/cortex/pkg/lib/print"
	"github.com/cortexlabs/cortex/pkg/lib/prompt"
	s "github.com/cortexlabs/cortex/pkg/lib/strings"
	"github.com/cortexlabs/cortex/pkg/lib/table"
	"github.com/cortexlabs/cortex/pkg/lib/telemetry"
	"github.com/cortexlabs/cortex/pkg/lib/zip"
	"github.com/cortexlabs/cortex/pkg/operator/schema"
	"github.com/cortexlabs/cortex/pkg/types"
	"github.com/spf13/cobra"
)

var _warningFileBytes = 1024 * 1024 * 10
var _warningProjectBytes = 1024 * 1024 * 10
var _warningFileCount = 1000

var _flagDeployForce bool
var _flagDeployYes bool

func init() {
	_deployCmd.PersistentFlags().BoolVarP(&_flagDeployForce, "force", "f", false, "override the in-progress api update")
	_deployCmd.PersistentFlags().BoolVarP(&_flagDeployYes, "yes", "y", false, "skip prompts")
	addEnvFlag(_deployCmd, types.LocalProviderType.String())
}

var _deployCmd = &cobra.Command{
	Use:   "deploy [CONFIG_FILE]",
	Short: "create or update apis",
	Args:  cobra.RangeArgs(0, 1),
	Run: func(cmd *cobra.Command, args []string) {
		telemetry.Event("cli.deploy")

		env := MustReadOrConfigureEnv(_flagEnv)
		configPath := getConfigPath(args)
		deploymentBytes := getDeploymentBytes(configPath)
		var deployResponse schema.DeployResponse
		var err error
		if env.Provider == types.AWSProviderType {
			params := map[string]string{
				"force":      s.Bool(_flagDeployForce),
				"configPath": configPath,
			}
			deployResponse, err = cluster.Deploy(MustGetOperatorConfig(env.Name), deploymentBytes, params)
			if err != nil {
				// TODO
			}
		} else {
			deployResponse, err = local.Deploy(configPath, deploymentBytes)
			if err != nil {
				// TODO
			}
		}
		message := deployMessage(deployResponse.Results)
		print.BoldFirstBlock(message)
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

func getDeploymentBytes(configPath string) map[string][]byte {
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
		msg := fmt.Sprintf("cortex will zip %d files in %s and upload them to the cluster; we recommend that you upload large files/directories (e.g. models) to s3 and download them in your api's __init__ function, and avoid sending unnecessary files by removing them from this directory or referencing them in a .cortexignore file. Would you like to continue?", len(projectPaths), rootDirMsg)
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
	return uploadBytes
}

func deployMessage(results []schema.DeployResult) string {
	statusMessage := mergeResultMessages(results)

	if didAllResultsError(results) {
		return statusMessage
	}

	apiCommandsMessage := getAPICommandsMessage(results)

	return statusMessage + "\n\n" + apiCommandsMessage
}

func mergeResultMessages(results []schema.DeployResult) string {
	var okMessages []string
	var errMessages []string

	for _, result := range results {
		if result.Error != "" {
			errMessages = append(errMessages, result.Error)
		} else {
			okMessages = append(okMessages, result.Message)
		}
	}

	messages := append(okMessages, errMessages...)

	return strings.Join(messages, "\n")
}

func didAllResultsError(results []schema.DeployResult) bool {
	for _, result := range results {
		if result.Error == "" {
			return false
		}
	}
	return true
}

func getAPICommandsMessage(results []schema.DeployResult) string {
	apiName := "<api_name>"
	if len(results) == 1 {
		apiName = results[0].API.Name
	}

	var items table.KeyValuePairs
	items.Add("cortex get", "(show api statuses)")
	items.Add(fmt.Sprintf("cortex get %s", apiName), "(show api info)")
	items.Add(fmt.Sprintf("cortex logs %s", apiName), "(stream api logs)")

	return strings.TrimSpace(items.String(&table.KeyValuePairOpts{
		Delimiter: pointer.String(""),
		NumSpaces: pointer.Int(2),
	}))
}
