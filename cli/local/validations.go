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
	"net"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/cortexlabs/cortex/pkg/consts"
	"github.com/cortexlabs/cortex/pkg/lib/aws"
	"github.com/cortexlabs/cortex/pkg/lib/docker"
	"github.com/cortexlabs/cortex/pkg/lib/errors"
	"github.com/cortexlabs/cortex/pkg/lib/files"
	"github.com/cortexlabs/cortex/pkg/lib/k8s"
	"github.com/cortexlabs/cortex/pkg/lib/pointer"
	"github.com/cortexlabs/cortex/pkg/lib/regex"
	"github.com/cortexlabs/cortex/pkg/lib/sets/strset"
	s "github.com/cortexlabs/cortex/pkg/lib/strings"
	"github.com/cortexlabs/cortex/pkg/types"
	"github.com/cortexlabs/cortex/pkg/types/spec"
	"github.com/cortexlabs/cortex/pkg/types/userconfig"
)

var _startingPort = 8888

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

	dockerClient, err := docker.GetDockerClient()
	if err != nil {
		return err
	}

	apisRequiringGPU := strset.New()
	for i := range apis {
		api := &apis[i]

		if err := spec.ValidateAPI(api, projectFiles, types.LocalProviderType, awsClient); err != nil {
			return err
		}

		if api.Compute.CPU != nil && (api.Compute.CPU.MilliValue() > int64(dockerClient.Info.NCPU)*1000) {
			qty := k8s.NewQuantity(int64(dockerClient.Info.NCPU))
			api.Compute.CPU = &qty
		}

		if api.Compute.GPU > 0 {
			apisRequiringGPU.Add(api.Name)
		}
	}

	if len(apisRequiringGPU) > 0 {
		if _, ok := dockerClient.Info.Runtimes["nvidia"]; !ok {
			if !strings.HasPrefix(strings.ToLower(runtime.GOOS), "linux") {
				fmt.Printf("warning: %s will run without gpu access because the nvidia container runtime is not supported on your operating system; see https://cortex.dev/troubleshooting/nvidia-container-runtime-not-found for more information\n\n", s.StrsAnd(apisRequiringGPU.SliceSorted()))
			} else {
				fmt.Printf("warning: %s will run without gpu access because your local machine doesn't have a gpu or the nvidia container runtime is not configured properly; see https://cortex.dev/troubleshooting/nvidia-container-runtime-not-found for more information\n\n", s.StrsAnd(apisRequiringGPU.SliceSorted()))
			}

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

	portToRunningAPIsMap, err := getPortToRunningAPIsMap()
	if err != nil {
		return err
	}

	var usedPorts []int

	runningAPIsToPortMap := map[string]int{}
	for port, apiName := range portToRunningAPIsMap {
		runningAPIsToPortMap[apiName] = port
		usedPorts = append(usedPorts, port)
	}

	portToUpdatingAPIMap := map[int]string{}
	updatingAPIToPortMap := map[string]*int{}

	for i := range apis {
		api := &apis[i]

		updatingAPIToPortMap[api.Name] = api.LocalPort
		if api.LocalPort != nil {
			if collidingAPIName, ok := portToUpdatingAPIMap[*api.LocalPort]; ok {
				return errors.Wrap(ErrorDuplicateLocalPort(collidingAPIName), api.Identify(), userconfig.LocalPortKey, s.Int(*api.LocalPort))
			}
			usedPorts = append(usedPorts, *api.LocalPort)
			portToUpdatingAPIMap[*api.LocalPort] = api.Name
		}
	}

	for i := range apis {
		api := &apis[i]
		if api.LocalPort != nil {
			// same port as previous deployment of this API
			if *api.LocalPort == runningAPIsToPortMap[api.Name] {
				continue
			}

			// port is being used by another API
			if apiName, ok := portToRunningAPIsMap[*api.LocalPort]; ok {
				return errors.Wrap(ErrorDuplicateLocalPort(apiName), api.Identify(), userconfig.LocalPortKey, s.Int(*api.LocalPort))
			}
			isPortAvailable, err := checkPortAvailability(*api.LocalPort)
			if err != nil {
				return err
			}

			if !isPortAvailable {
				return errors.Wrap(ErrorPortAlreadyInUse(*api.LocalPort), api.Identify(), userconfig.LocalPortKey)
			}
		} else {
			// get previous api deployment port
			if port, ok := runningAPIsToPortMap[api.Name]; ok {

				// check that the previous api deployment port has not been claimed in new deployment
				if _, ok := portToUpdatingAPIMap[port]; !ok {
					api.LocalPort = pointer.Int(port)
				}
			}
		}
	}

	for i := range apis {
		api := &apis[i]
		if api.LocalPort == nil {
			availablePort, err := findTheNextAvailablePort(usedPorts)
			if err != nil {
				errors.Wrap(err, api.Identify())
			}

			api.LocalPort = pointer.Int(availablePort)
		}
	}

	return nil
}

func checkPortAvailability(port int) (bool, error) {
	ln, err := net.Listen("tcp", ":"+s.Int(port))
	if err != nil {
		return false, nil
	}
	err = ln.Close()
	if err != nil {
		return false, errors.WithStack(err)
	}

	return true, nil
}

func findTheNextAvailablePort(blackListedPorts []int) (int, error) {
	defer func() { _startingPort++ }()
	blackListedSet := map[int]struct{}{}
	for _, port := range blackListedPorts {
		blackListedSet[port] = struct{}{}
	}

	for _startingPort <= 65535 {
		if _, ok := blackListedSet[_startingPort]; ok {
			_startingPort++
			continue
		}

		isAvailable, err := checkPortAvailability(_startingPort)
		if err != nil {
			return 0, err
		}

		if isAvailable {
			return _startingPort, nil
		}

		_startingPort++
	}

	return 0, ErrorUnableToFindAvailablePorts()
}

func getPortToRunningAPIsMap() (map[int]string, error) {
	allContainers, err := GetAllRunningContainers()
	if err != nil {
		return nil, err
	}

	portMap := map[int]string{}

	for _, container := range allContainers {
		if container.Labels["type"] == _apiContainerName {
			for _, port := range container.Ports {
				if port.PrivatePort == 8888 {
					portMap[int(port.PublicPort)] = container.Labels["apiName"]
				}
			}
		}
	}

	return portMap, nil
}
