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
	"math"
	"net"
	"path"
	"path/filepath"
	"strings"

	"github.com/cortexlabs/cortex/cli/types/flags"
	"github.com/cortexlabs/cortex/pkg/lib/aws"
	"github.com/cortexlabs/cortex/pkg/lib/docker"
	"github.com/cortexlabs/cortex/pkg/lib/errors"
	"github.com/cortexlabs/cortex/pkg/lib/files"
	"github.com/cortexlabs/cortex/pkg/lib/gcp"
	"github.com/cortexlabs/cortex/pkg/lib/k8s"
	"github.com/cortexlabs/cortex/pkg/lib/pointer"
	"github.com/cortexlabs/cortex/pkg/lib/regex"
	"github.com/cortexlabs/cortex/pkg/lib/sets/strset"
	"github.com/cortexlabs/cortex/pkg/lib/slices"
	s "github.com/cortexlabs/cortex/pkg/lib/strings"
	"github.com/cortexlabs/cortex/pkg/types"
	"github.com/cortexlabs/cortex/pkg/types/spec"
	"github.com/cortexlabs/cortex/pkg/types/userconfig"
)

var _startingPort = 8889

type ProjectFiles struct {
	relFilePaths []string
	projectRoot  string
}

func newProjectFiles(projectFileList []string, projectRoot string) (ProjectFiles, error) {
	relFilePaths := make([]string, len(projectFileList))
	for i, projectFilePath := range projectFileList {
		if !files.IsAbsOrTildePrefixed(projectFilePath) {
			return ProjectFiles{}, errors.ErrorUnexpected(fmt.Sprintf("%s is not an absolute path", projectFilePath))
		}
		if !strings.HasPrefix(projectFilePath, projectRoot) {
			return ProjectFiles{}, errors.ErrorUnexpected(fmt.Sprintf("%s is not located within in the project", projectFilePath))
		}
		relFilePaths[i] = strings.TrimPrefix(projectFilePath, projectRoot)
	}

	return ProjectFiles{
		relFilePaths: relFilePaths,
		projectRoot:  projectRoot,
	}, nil
}

func (projectFiles ProjectFiles) AllPaths() []string {
	return projectFiles.relFilePaths
}

func (projectFiles ProjectFiles) AllAbsPaths() []string {
	absPaths := make([]string, 0, len(projectFiles.relFilePaths))
	for _, relPath := range projectFiles.relFilePaths {
		absPaths = append(absPaths, path.Join(projectFiles.projectRoot, relPath))
	}

	return absPaths
}

func (projectFiles ProjectFiles) GetFile(path string) ([]byte, error) {
	for _, projectFilePath := range projectFiles.relFilePaths {
		if path == projectFilePath {
			bytes, err := files.ReadFileBytes(filepath.Join(projectFiles.projectRoot, path))
			if err != nil {
				return nil, err
			}
			return bytes, nil
		}
	}

	return nil, files.ErrorFileDoesNotExist(path)
}

func (projectFiles ProjectFiles) HasFile(path string) bool {
	return slices.HasString(projectFiles.relFilePaths, path)
}

func (projectFiles ProjectFiles) HasDir(path string) bool {
	path = s.EnsureSuffix(path, "/")
	for _, projectFilePath := range projectFiles.relFilePaths {
		if strings.HasPrefix(projectFilePath, path) {
			return true
		}
	}
	return false
}

// Get the absolute path to the project directory
func (projectFiles ProjectFiles) ProjectDir() string {
	return projectFiles.projectRoot
}

func ValidateLocalAPIs(apis []userconfig.API, models *[]spec.CuratedModelResource, projectFiles ProjectFiles, awsClient *aws.Client, gcpClient *gcp.Client) error {
	if len(apis) == 0 {
		return spec.ErrorNoAPIs()
	}

	dockerClient, err := docker.GetDockerClient()
	if err != nil {
		return err
	}

	for i := range apis {
		api := &apis[i]

		if err := spec.ValidateAPI(api, models, projectFiles, types.LocalProviderType, awsClient, gcpClient, nil); err != nil {
			return errors.Wrap(err, api.Identify())
		}

		if api.Compute.CPU != nil && (api.Compute.CPU.MilliValue() > int64(dockerClient.Info.NCPU)*1000) {
			api.Compute.CPU = k8s.NewQuantity(int64(dockerClient.Info.NCPU))
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

		pullVerbosity := docker.PrintDots
		if OutputType == flags.JSONOutputType {
			pullVerbosity = docker.NoPrint
		}

		pulledThisImage, err := docker.PullImage(image, dockerAuth, pullVerbosity)
		if err != nil {
			return errors.Wrap(err, "failed to pull image", image)
		}

		if pulledThisImage {
			pulledImage = true
		}
	}

	if pulledImage {
		localPrintln()
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

		updatingAPIToPortMap[api.Name] = api.Networking.LocalPort
		if api.Networking.LocalPort != nil {
			if collidingAPIName, ok := portToUpdatingAPIMap[*api.Networking.LocalPort]; ok {
				return errors.Wrap(ErrorDuplicateLocalPort(collidingAPIName), api.Identify(), userconfig.LocalPortKey, s.Int(*api.Networking.LocalPort))
			}
			usedPorts = append(usedPorts, *api.Networking.LocalPort)
			portToUpdatingAPIMap[*api.Networking.LocalPort] = api.Name
		}
	}

	for i := range apis {
		api := &apis[i]
		if api.Networking.LocalPort != nil {
			// same port as previous deployment of this API
			if *api.Networking.LocalPort == runningAPIsToPortMap[api.Name] {
				continue
			}

			// port is being used by another API
			if apiName, ok := portToRunningAPIsMap[*api.Networking.LocalPort]; ok {
				return errors.Wrap(ErrorDuplicateLocalPort(apiName), api.Identify(), userconfig.LocalPortKey, s.Int(*api.Networking.LocalPort))
			}
			isPortAvailable, err := checkPortAvailability(*api.Networking.LocalPort)
			if err != nil {
				return err
			}

			if !isPortAvailable {
				return errors.Wrap(ErrorPortAlreadyInUse(*api.Networking.LocalPort), api.Identify(), userconfig.LocalPortKey)
			}
		} else {
			// get previous api deployment port
			if port, ok := runningAPIsToPortMap[api.Name]; ok {

				// check that the previous api deployment port has not been claimed in new deployment
				if _, ok := portToUpdatingAPIMap[port]; !ok {
					api.Networking.LocalPort = pointer.Int(port)
				}
			}
		}
	}

	for i := range apis {
		api := &apis[i]
		if api.Networking.LocalPort == nil {
			availablePort, err := findTheNextAvailablePort(usedPorts)
			if err != nil {
				errors.Wrap(err, api.Identify())
			}

			api.Networking.LocalPort = pointer.Int(availablePort)
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

	for _startingPort <= math.MaxUint16 {
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
