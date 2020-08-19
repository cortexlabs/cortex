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
	"fmt"
	"path/filepath"

	"github.com/cortexlabs/cortex/cli/types/cliconfig"
	"github.com/cortexlabs/cortex/pkg/consts"
	"github.com/cortexlabs/cortex/pkg/lib/aws"
	"github.com/cortexlabs/cortex/pkg/lib/docker"
	"github.com/cortexlabs/cortex/pkg/lib/errors"
	"github.com/cortexlabs/cortex/pkg/lib/files"
	"github.com/cortexlabs/cortex/pkg/operator/schema"
	"github.com/cortexlabs/cortex/pkg/types"
	"github.com/cortexlabs/cortex/pkg/types/spec"
)

func Deploy(env cliconfig.Environment, configPath string, projectFileList []string, deployDisallowPrompt bool) (schema.DeployResponse, error) {
	configFileName := filepath.Base(configPath)

	_, err := docker.GetDockerClient()
	if err != nil {
		return schema.DeployResponse{}, err
	}

	configBytes, err := files.ReadFileBytes(configPath)
	if err != nil {
		return schema.DeployResponse{}, err
	}

	projectFiles, err := newProjectFiles(projectFileList, configPath)
	if err != nil {
		return schema.DeployResponse{}, err
	}

	var awsClient *aws.Client
	if env.AWSAccessKeyID != nil {
		awsClient, err = aws.NewFromCreds(*env.AWSRegion, *env.AWSAccessKeyID, *env.AWSSecretAccessKey)
		if err != nil {
			return schema.DeployResponse{}, err
		}
	} else {
		awsClient, err = aws.NewAnonymousClient()
		if err != nil {
			return schema.DeployResponse{}, err
		}
	}

	apiConfigs, err := spec.ExtractAPIConfigs(configBytes, types.LocalProviderType, configFileName)
	if err != nil {
		return schema.DeployResponse{}, err
	}

	err = ValidateLocalAPIs(apiConfigs, projectFiles, awsClient)
	if err != nil {
		err = errors.Append(err, fmt.Sprintf("\n\napi configuration schema for Realtime API can be found at https://docs.cortex.dev/v/%s/deployments/realtime-api/api-configuration", consts.CortexVersionMinor))
		return schema.DeployResponse{}, err
	}

	projectID, err := files.HashFile(projectFileList[0], projectFileList[1:]...)
	if err != nil {
		return schema.DeployResponse{}, errors.Wrap(err, "failed to hash directory", filepath.Dir(configPath))
	}

	results := make([]schema.DeployResult, len(apiConfigs))
	for i, apiConfig := range apiConfigs {
		api, msg, err := UpdateAPI(&apiConfig, configPath, projectID, deployDisallowPrompt, awsClient)
		results[i].Message = msg
		if err != nil {
			results[i].Error = errors.Message(err)
		} else {
			results[i].API = *api
		}
	}

	return schema.DeployResponse{
		Results: results,
	}, nil
}
