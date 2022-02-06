/*
Copyright 2022 Cortex Labs, Inc.

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

	"github.com/cortexlabs/cortex/cli/cluster"
	"github.com/cortexlabs/cortex/cli/types/flags"
	"github.com/cortexlabs/cortex/pkg/lib/exit"
	"github.com/cortexlabs/cortex/pkg/lib/files"
	libjson "github.com/cortexlabs/cortex/pkg/lib/json"
	"github.com/cortexlabs/cortex/pkg/lib/print"
	"github.com/cortexlabs/cortex/pkg/lib/telemetry"
	"github.com/cortexlabs/cortex/pkg/operator/schema"
	"github.com/spf13/cobra"
)

var (
	_warningFileBytes    = 1024 * 1024 * 10
	_warningProjectBytes = 1024 * 1024 * 10
	_warningFileCount    = 1000

	_maxFileSizeBytes    int64 = 1024 * 1024 * 32 // 32mb
	_maxProjectSizeBytes int64 = 1024 * 1024 * 32 // 32mb

	_flagDeployEnv            string
	_flagDeployForce          bool
	_flagDeployDisallowPrompt bool
)

func deployInit() {
	_deployCmd.Flags().SortFlags = false
	_deployCmd.Flags().StringVarP(&_flagDeployEnv, "env", "e", "", "environment to use")
	_deployCmd.Flags().BoolVarP(&_flagDeployForce, "force", "f", false, "override the in-progress api update")
	_deployCmd.Flags().BoolVarP(&_flagDeployDisallowPrompt, "yes", "y", false, "skip prompts")
	_deployCmd.Flags().VarP(&_flagOutput, "output", "o", fmt.Sprintf("output format: one of %s", strings.Join(flags.OutputTypeStringsExcluding(flags.YAMLOutputType), "|")))
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
		telemetry.Event("cli.deploy", map[string]interface{}{"env_name": env.Name})

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
			message := mergeResultMessages(deployResults)
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

func getDeploymentBytes(configPath string) (map[string][]byte, error) {
	configBytes, err := files.ReadFileBytes(configPath)
	if err != nil {
		return nil, err
	}

	uploadBytes := map[string][]byte{
		"config": configBytes,
	}

	return uploadBytes, nil
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
