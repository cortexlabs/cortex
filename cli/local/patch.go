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

package local

import (
	"path"
	"path/filepath"

	"github.com/cortexlabs/cortex/cli/types/cliconfig"
	"github.com/cortexlabs/cortex/pkg/lib/files"
	"github.com/cortexlabs/cortex/pkg/operator/schema"
	"github.com/cortexlabs/cortex/pkg/types"
	"github.com/cortexlabs/cortex/pkg/types/spec"
	"github.com/cortexlabs/cortex/pkg/types/userconfig"
)

func Patch(env cliconfig.Environment, configPath string) ([]schema.DeployResult, error) {
	configFileName := filepath.Base(configPath)

	configBytes, err := files.ReadFileBytes(configPath)
	if err != nil {
		return nil, err
	}

	apiConfigs, err := spec.ExtractAPIConfigs(configBytes, types.LocalProviderType, configFileName, nil, nil)
	if err != nil {
		return nil, err
	}

	deployResults := []schema.DeployResult{}
	for i := range apiConfigs {
		apiConfig := &apiConfigs[i]
		apiResponse, err := GetAPI(apiConfig.Name)
		if err != nil {
			return nil, err
		}

		localProjectDir := apiResponse[0].Spec.LocalProjectDir

		projectFileList, err := findProjectFiles(localProjectDir)
		if err != nil {
			return nil, err
		}

		projectFiles, err := newProjectFiles(projectFileList, localProjectDir)
		if err != nil {
			return nil, err
		}

		deployResult, err := deploy(env, []userconfig.API{*apiConfig}, projectFiles, true)
		if err != nil {
			return nil, err
		}

		deployResults = append(deployResults, deployResult...)
	}

	return deployResults, nil
}

func findProjectFiles(projectRoot string) ([]string, error) {
	ignoreFns := []files.IgnoreFn{
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
