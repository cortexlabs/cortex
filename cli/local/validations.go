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

	"github.com/cortexlabs/cortex/pkg/consts"
	"github.com/cortexlabs/cortex/pkg/lib/aws"
	"github.com/cortexlabs/cortex/pkg/lib/docker"
	"github.com/cortexlabs/cortex/pkg/lib/errors"
	"github.com/cortexlabs/cortex/pkg/lib/files"
	"github.com/cortexlabs/cortex/pkg/lib/regex"
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

	apiPortMap := map[int]string{}
	apisRequiringGPU := strset.New()
	nonLocalConfigs := strset.New()
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

		if api.Endpoint != nil {
			nonLocalConfigs.Add(userconfig.EndpointKey)
		}
		if api.Autoscaling != nil {
			nonLocalConfigs.Add(userconfig.AutoscalingKey)
		}
		if api.Tracker != nil {
			nonLocalConfigs.Add(userconfig.TrackerKey)
		}
		if api.UpdateStrategy != nil {
			nonLocalConfigs.Add(userconfig.UpdateStrategyKey)
		}

		if api.Compute.GPU > 0 {
			apisRequiringGPU.Add(api.Name)
		}
	}

	if len(nonLocalConfigs) > 0 {
		configurationStr := "configuration"
		if len(nonLocalConfigs) > 1 {
			configurationStr = "configurations"
		}
		fmt.Println(fmt.Sprintf("note: you're deploying locally, so the %s %s won't apply\n", s.StrsAnd(nonLocalConfigs.SliceSorted()), configurationStr))
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
			fmt.Println(fmt.Sprintf("warning: %s will run without GPU access because your machine doesn't have GPUs or the nvidia runtime is not properly configured (use `docker info | grep -i runtime` to list docker runtimes, see https://github.com/NVIDIA/nvidia-container-runtime#installation for instructions)\n", s.StrsAnd(apisRequiringGPU.Slice())))
			for i := range apis {
				api := &apis[i]
				if apisRequiringGPU.Has(api.Name) {
					api.Compute.GPU = 0
				}
				switch api.Predictor.Image {
				case consts.DefaultImageONNXPredictorGPU:
					api.Predictor.Image = consts.DefaultImageONNXPredictorCPU
				case consts.DefaultImagePythonPredictorGPU:
					api.Predictor.Image = consts.DefaultImagePythonPredictorCPU
				}

				if api.Predictor.Type == userconfig.TensorFlowPredictorType && api.Predictor.TensorFlowServingImage == consts.DefaultImageTensorFlowServingGPU {
					api.Predictor.TensorFlowServingImage = consts.DefaultImageTensorFlowServingCPU
				}
			}
		}
	}

	dups := spec.FindDuplicateNames(apis)
	if len(dups) > 0 {
		return spec.ErrorDuplicateName(dups)
	}

	imageSet := strset.New()
	for _, api := range apis {
		imageSet.Add(api.Predictor.Image)
		if api.Predictor.Type == userconfig.TensorFlowPredictorType {
			imageSet.Add(api.Predictor.TensorFlowServingImage)
		}
	}

	pulledImage := false
	for image := range imageSet {
		var err error
		dockerAuth := docker.NoAuth
		if regex.IsValidECRURL(image) && !awsClient.IsAnonymous {
			dockerAuth, err = docker.AWSAuthConfig(awsClient)
			if err != nil {
				return err
			}
		}

		pulledThisImage, err := docker.PullImage(image, dockerAuth, docker.PrintDots)
		if err != nil {
			return errors.Wrap(err, "failed to pull image", image)
		}

		if pulledThisImage {
			pulledImage = true
		}
	}

	if pulledImage {
		fmt.Println()
	}

	return nil
}
