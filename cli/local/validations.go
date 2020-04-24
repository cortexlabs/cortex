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
	"context"
	"fmt"
	"path/filepath"
	"strings"

	"github.com/cortexlabs/cortex/pkg/lib/aws"
	"github.com/cortexlabs/cortex/pkg/lib/docker"
	"github.com/cortexlabs/cortex/pkg/lib/errors"
	"github.com/cortexlabs/cortex/pkg/lib/files"
	"github.com/cortexlabs/cortex/pkg/lib/sets/strset"
	s "github.com/cortexlabs/cortex/pkg/lib/strings"
	"github.com/cortexlabs/cortex/pkg/types"
	"github.com/cortexlabs/cortex/pkg/types/spec"
	"github.com/cortexlabs/cortex/pkg/types/userconfig"
)

type ProjectFiles struct {
	projectFileList []string // make sure it is absolute paths
	configFilePath  string
}

func NewProjectFiles(projectFileList []string, absoluteConfigFilePath string) (ProjectFiles, error) {
	if !filepath.IsAbs(absoluteConfigFilePath) {
		return ProjectFiles{}, ErrorNotAbsolutePath(absoluteConfigFilePath) // unexpected
	}

	for _, projectFile := range projectFileList {
		if !filepath.IsAbs(projectFile) {
			return ProjectFiles{}, ErrorNotAbsolutePath(absoluteConfigFilePath) // unexpected
		}
	}

	return ProjectFiles{
		projectFileList: projectFileList,
		configFilePath:  absoluteConfigFilePath,
	}, nil
}

func (projectFiles ProjectFiles) GetAllPaths() []string {
	return projectFiles.projectFileList
}

func (projectFiles ProjectFiles) GetFile(fileName string) ([]byte, error) {
	baseDir := filepath.Dir(projectFiles.configFilePath)

	absPath := files.RelToAbsPath(fileName, baseDir)
	for _, path := range projectFiles.projectFileList {
		if path == absPath {
			bytes, err := files.ReadFileBytes(absPath)
			if err != nil {
				return nil, err
			}
			return bytes, nil
		}
	}

	return nil, files.ErrorFileDoesNotExist(fileName)
}

func (projectFiles ProjectFiles) GetConfigFilePath() string {
	return projectFiles.configFilePath
}

func ValidateLocalAPIs(apis []userconfig.API, projectFiles ProjectFiles, awsClient *aws.Client) error {
	if len(apis) == 0 {
		return spec.ErrorNoAPIs()
	}

	didPrintWarning := false
	apiPortMap := map[int]string{}
	apisRequiringGPU := strset.New()
	for i := range apis {
		api := &apis[i]
		if err := spec.ValidateAPI(api, projectFiles, types.LocalProviderType, awsClient); err != nil {
			return err
		}

		if api.LocalPort != nil {
			if collidingAPIName, ok := apiPortMap[*api.LocalPort]; ok {
				return errors.Wrap(ErrorDuplicateLocalPort(collidingAPIName), api.Identify(), userconfig.LocalPortKey, s.Int(*apis[i].LocalPort))
			}
			apiPortMap[*api.LocalPort] = api.Name
		}

		if api.Endpoint != nil || api.Autoscaling != nil || api.Tracker != nil || api.UpdateStrategy != nil {
			if !didPrintWarning {
				fmt.Println(fmt.Sprintf("note: %s, %s, %s, and %s keys will be ignored because they are not supported in local environment\n", userconfig.EndpointKey, userconfig.AutoscalingKey, userconfig.TrackerKey, userconfig.UpdateStrategyKey))
			}
			didPrintWarning = true
		}

		if api.Compute.GPU > 0 {
			apisRequiringGPU.Add(api.Name)
		}
	}

	if len(apisRequiringGPU) > 0 {
		dockerClient, err := docker.GetDockerClient()
		if err != nil {
			return err
		}

		infoResponse, err := dockerClient.Info(context.Background())
		if err != nil {
			return err
		}

		if _, ok := infoResponse.Runtimes["nvidia"]; !ok {
			fmt.Println(fmt.Sprintf("warning: unable to find nvidia runtime on your docker (confirm with `docker info | grep -i runtime`); see https://github.com/NVIDIA/nvidia-container-runtime#installation to register nvidia runtime on your docker; in the meantime, the following api(s) will be run without GPU access: %s\n", strings.Join(apisRequiringGPU.Slice(), ", ")))
		}

		for i := range apis {
			api := &apis[i]
			if apisRequiringGPU.Has(api.Name) {
				api.Compute.GPU = 0
			}
		}
	}

	dups := spec.FindDuplicateNames(apis)
	if len(dups) > 0 {
		return spec.ErrorDuplicateName(dups)
	}

	return nil
}
