/*
Copyright 2021 Cortex Labs, Inc.

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
	"strings"

	"github.com/cortexlabs/cortex/cli/cluster"
	"github.com/cortexlabs/cortex/cli/types/flags"
	"github.com/cortexlabs/cortex/pkg/lib/archive"
	"github.com/cortexlabs/cortex/pkg/lib/errors"
	"github.com/cortexlabs/cortex/pkg/lib/exit"
	"github.com/cortexlabs/cortex/pkg/lib/files"
	libjson "github.com/cortexlabs/cortex/pkg/lib/json"
	"github.com/cortexlabs/cortex/pkg/lib/pointer"
	"github.com/cortexlabs/cortex/pkg/lib/print"
	"github.com/cortexlabs/cortex/pkg/lib/prompt"
	s "github.com/cortexlabs/cortex/pkg/lib/strings"
	"github.com/cortexlabs/cortex/pkg/lib/table"
	"github.com/cortexlabs/cortex/pkg/lib/telemetry"
	"github.com/cortexlabs/cortex/pkg/operator/schema"
	"github.com/cortexlabs/cortex/pkg/types/userconfig"
	"github.com/spf13/cobra"
)

var (
	_warningFileBytes    = 1024 * 1024 * 10
	_warningProjectBytes = 1024 * 1024 * 10
	_warningFileCount    = 1000

	_maxFileSizeBytes    int64 = 1024 * 1024 * 512
	_maxProjectSizeBytes int64 = 1024 * 1024 * 512

	_flagDeployEnv            string
	_flagDeployForce          bool
	_flagDeployDisallowPrompt bool
)

func deployInit() {
	_deployCmd.Flags().SortFlags = false
	_deployCmd.Flags().StringVarP(&_flagDeployEnv, "env", "e", "", "environment to use")
	_deployCmd.Flags().BoolVarP(&_flagDeployForce, "force", "f", false, "override the in-progress api update")
	_deployCmd.Flags().BoolVarP(&_flagDeployDisallowPrompt, "yes", "y", false, "skip prompts")
	_deployCmd.Flags().VarP(&_flagOutput, "output", "o", fmt.Sprintf("output format: one of %s", strings.Join(flags.UserOutputTypeStrings(), "|")))
}

var _deployCmd = &cobra.Command{
	Use:   "deploy [CONFIG_FILE]",
	Short: "create or update apis",
	Args:  cobra.RangeArgs(0, 1),
	Run: func(cmd *cobra.Command, args []string) {
		envName, err := getEnvFromFlag(_flagDeployEnv)
		if err != nil {
			telemetry.Event("cli.deploy")
			exit.Error(err)
		}

		env, err := ReadOrConfigureEnv(envName)
		if err != nil {
			telemetry.Event("cli.deploy")
			exit.Error(err)
		}
		telemetry.Event("cli.deploy", map[string]interface{}{"provider": env.Provider.String(), "env_name": env.Name})

		err = printEnvIfNotSpecified(env.Name, cmd)
		if err != nil {
			exit.Error(err)
		}

		configPath := getConfigPath(args)

		projectRoot := files.Dir(configPath)
		if projectRoot == _homeDir {
			exit.Error(ErrorDeployFromTopLevelDir("home"))
		}
		if projectRoot == "/" {
			exit.Error(ErrorDeployFromTopLevelDir("root"))
		}

		deploymentBytes, err := getDeploymentBytes(configPath)
		if err != nil {
			exit.Error(err)
		}

		deployResults, err := cluster.Deploy(MustGetOperatorConfig(env.Name), configPath, deploymentBytes, _flagDeployForce)
		if err != nil {
			exit.Error(err)
		}

		switch _flagOutput {
		case flags.JSONOutputType:
			bytes, err := libjson.Marshal(deployResults)
			if err != nil {
				exit.Error(err)
			}
			fmt.Print(string(bytes))
		case flags.PrettyOutputType:
			message, err := deployMessage(deployResults, env.Name)
			if err != nil {
				exit.Error(err)
			}
			if didAnyResultsError(deployResults) {
				print.StderrBoldFirstBlock(message)
			} else {
				print.BoldFirstBlock(message)
			}
		}

		if didAnyResultsError(deployResults) {
			exit.Error(nil)
		}
	},
}

// Returns absolute path
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

	return files.RelToAbsPath(configPath, _cwd)
}

func findProjectFiles(configPath string) ([]string, error) {
	projectRoot := files.Dir(configPath)

	ignoreFns := []files.IgnoreFn{
		files.IgnoreSpecificFiles(configPath),
		files.IgnoreCortexDebug,
		files.IgnoreHiddenFiles,
		files.IgnoreHiddenFolders,
		files.IgnorePythonGeneratedFiles,
	}

	cortexIgnorePath := path.Join(projectRoot, ".cortexignore")
	if files.IsFile(cortexIgnorePath) {
		cortexIgnore, err := files.GitIgnoreFn(cortexIgnorePath)
		if err != nil {
			return nil, err
		}
		ignoreFns = append(ignoreFns, cortexIgnore)
	}

	if !_flagDeployDisallowPrompt {
		ignoreFns = append(ignoreFns, files.PromptForFilesAboveSize(_warningFileBytes, "do you want to upload %s (%s)?"))
	}
	ignoreFns = append(ignoreFns,
		files.ErrorOnBigFilesFn(_maxFileSizeBytes),
		// must be the last appended IgnoreFn
		files.ErrorOnProjectSizeLimit(_maxProjectSizeBytes),
	)

	projectPaths, err := files.ListDirRecursive(projectRoot, false, ignoreFns...)
	if err != nil {
		return nil, err
	}

	// Include .env file containing environment variables
	dotEnvPath := path.Join(projectRoot, ".env")
	if files.IsFile(dotEnvPath) {
		projectPaths = append(projectPaths, dotEnvPath)
	}

	return projectPaths, nil
}

func getDeploymentBytes(configPath string) (map[string][]byte, error) {
	configBytes, err := files.ReadFileBytes(configPath)
	if err != nil {
		return nil, err
	}

	uploadBytes := map[string][]byte{
		"config": configBytes,
	}

	projectRoot := files.Dir(configPath)

	projectPaths, err := findProjectFiles(configPath)
	if err != nil {
		return nil, err
	}

	canSkipPromptMsg := "you can skip this prompt next time with `cortex deploy --yes`\n"
	rootDirMsg := "this directory"
	if projectRoot != _cwd {
		rootDirMsg = fmt.Sprintf("./%s", files.DirPathRelativeToCWD(projectRoot))
	}

	didPromptFileCount := false
	if !_flagDeployDisallowPrompt && len(projectPaths) >= _warningFileCount {
		msg := fmt.Sprintf("cortex will zip %d files in %s and upload them to the cluster; we recommend that you upload large files/directories (e.g. models) to s3 and download them in your api's __init__ function, and avoid sending unnecessary files by removing them from this directory or referencing them in a .cortexignore file. Would you like to continue?", len(projectPaths), rootDirMsg)
		prompt.YesOrExit(msg, canSkipPromptMsg, "")
		didPromptFileCount = true
	}

	projectZipBytes, _, err := archive.ZipToMem(&archive.Input{
		FileLists: []archive.FileListInput{
			{
				Sources:      projectPaths,
				RemovePrefix: projectRoot,
			},
		},
	})
	if err != nil {
		return nil, errors.Wrap(err, "failed to zip project folder")
	}

	if !_flagDeployDisallowPrompt && !didPromptFileCount && len(projectZipBytes) >= _warningProjectBytes {
		msg := fmt.Sprintf("cortex will zip %d files in %s (%s) and upload them to the cluster, though we recommend you upload large files (e.g. models) to s3 and download them in your api's __init__ function. Would you like to continue?", len(projectPaths), rootDirMsg, s.IntToBase2Byte(len(projectZipBytes)))
		prompt.YesOrExit(msg, canSkipPromptMsg, "")
	}

	uploadBytes["project.zip"] = projectZipBytes
	return uploadBytes, nil
}

func deployMessage(results []schema.DeployResult, envName string) (string, error) {
	statusMessage := mergeResultMessages(results)

	if didAllResultsError(results) {
		return statusMessage, nil
	}

	apiCommandsMessage, err := getAPICommandsMessage(results, envName)
	if err != nil {
		return "", err
	}

	return statusMessage + "\n\n" + apiCommandsMessage, nil
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

	output := ""

	if len(okMessages) > 0 {
		output += strings.Join(okMessages, "\n")
		if len(errMessages) > 0 {
			output += "\n\n"
		}
	}

	if len(errMessages) > 0 {
		output += strings.Join(errMessages, "\n")
	}

	return output
}

func didAllResultsError(results []schema.DeployResult) bool {
	for _, result := range results {
		if result.Error == "" {
			return false
		}
	}
	return true
}

func didAnyResultsError(results []schema.DeployResult) bool {
	for _, result := range results {
		if result.Error != "" {
			return true
		}
	}
	return false
}

func getAPICommandsMessage(results []schema.DeployResult, envName string) (string, error) {
	apiName := "<api_name>"
	if len(results) == 1 {
		apiName = results[0].API.Spec.Name
	}

	defaultEnv, err := getDefaultEnv()
	if err != nil {
		return "", err
	}
	var envArg string
	if defaultEnv == nil || envName != *defaultEnv {
		envArg = " --env " + envName
	}

	var items table.KeyValuePairs
	items.Add("cortex get"+envArg, "(show api statuses)")
	items.Add(fmt.Sprintf("cortex get %s%s", apiName, envArg), "(show api info)")

	for _, result := range results {
		if len(result.Error) > 0 {
			continue
		}
		if result.API != nil && result.API.Spec.Kind == userconfig.RealtimeAPIKind {
			items.Add(fmt.Sprintf("cortex logs %s%s", apiName, envArg), "(stream api logs)")
			break
		}
	}

	return strings.TrimSpace(items.String(&table.KeyValuePairOpts{
		Delimiter: pointer.String(""),
		NumSpaces: pointer.Int(2),
	})), nil
}
