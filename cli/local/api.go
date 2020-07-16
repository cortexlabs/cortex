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
	"strings"

	"github.com/cortexlabs/cortex/pkg/consts"
	"github.com/cortexlabs/cortex/pkg/lib/aws"
	"github.com/cortexlabs/cortex/pkg/lib/errors"
	"github.com/cortexlabs/cortex/pkg/lib/files"
	"github.com/cortexlabs/cortex/pkg/lib/msgpack"
	"github.com/cortexlabs/cortex/pkg/lib/pointer"
	"github.com/cortexlabs/cortex/pkg/lib/sets/strset"
	"github.com/cortexlabs/cortex/pkg/types/spec"
	"github.com/cortexlabs/cortex/pkg/types/userconfig"
)

var _deploymentID = "local"

func UpdateAPI(apiConfig *userconfig.API, models []spec.CuratedModelResource, configPath string, projectID string, awsClient *aws.Client) (*spec.API, string, error) {
	prevAPISpec, err := FindAPISpec(apiConfig.Name)
	if err != nil {
		if errors.GetKind(err) != ErrAPINotDeployed {
			return nil, "", err
		}
	}

	prevAPIContainers, err := GetContainersByAPI(apiConfig.Name)
	if err != nil {
		return nil, "", err
	}

	newAPISpec := spec.GetAPISpec(apiConfig, models, projectID, _deploymentID)

	// apiConfig.Predictor.ModelPath was already added to apiConfig.Predictor.Models for ease of use
	if len(apiConfig.Predictor.Models.Paths) > 0 {
		localModelCaches, err := CacheModels(newAPISpec, awsClient)
		if err != nil {
			return nil, "", err
		}
		newAPISpec.LocalModelCaches = localModelCaches
	}

	newAPISpec.LocalProjectDir = files.Dir(configPath)

	if areAPIsEqual(newAPISpec, prevAPISpec) {
		return newAPISpec, fmt.Sprintf("%s is up to date", newAPISpec.Name), nil
	}

	if prevAPISpec != nil || len(prevAPIContainers) != 0 {
		err = errors.FirstError(
			DeleteAPI(newAPISpec.Name),
			DeleteCachedModels(newAPISpec.Name, prevAPISpec.SubtractModelIDs(newAPISpec)),
		)
		if err != nil {
			return nil, "", err
		}
	}

	err = writeAPISpec(newAPISpec)
	if err != nil {
		DeleteAPI(newAPISpec.Name)
		DeleteCachedModels(newAPISpec.Name, newAPISpec.ModelIDs())
		return nil, "", err
	}

	if err := DeployContainers(newAPISpec, awsClient); err != nil {
		DeleteAPI(newAPISpec.Name)
		DeleteCachedModels(newAPISpec.Name, newAPISpec.ModelIDs())
		return nil, "", err
	}

	if prevAPISpec == nil && len(prevAPIContainers) == 0 {
		return newAPISpec, fmt.Sprintf("creating %s", newAPISpec.Name), nil
	}

	return newAPISpec, fmt.Sprintf("updating %s", newAPISpec.Name), nil
}

func writeAPISpec(apiSpec *spec.API) error {
	apiBytes, err := msgpack.Marshal(apiSpec)
	if err != nil {
		return err
	}

	err = files.CreateDir(files.ParentDir(filepath.Join(_localWorkspaceDir, apiSpec.Key)))
	if err != nil {
		return err
	}

	err = files.WriteFile(apiBytes, filepath.Join(_localWorkspaceDir, apiSpec.Key))
	if err != nil {
		return err
	}

	return nil
}

func areAPIsEqual(a1, a2 *spec.API) bool {
	if a1 == nil && a2 == nil {
		return true
	}
	if a1 == nil || a2 == nil {
		return false
	}
	if a1.ID != a2.ID {
		return false
	}
	if !pointer.AreIntsEqual(a1.Networking.LocalPort, a2.Networking.LocalPort) {
		return false
	}
	if !a1.Compute.Equals(a2.Compute) {
		return false
	}
	if !strset.FromSlice(a1.ModelIDs()).IsEqual(strset.FromSlice(a2.ModelIDs())) {
		return false
	}
	return true
}

func DeleteAPI(apiName string) error {
	errList := []error{}

	containers, err := GetContainersByAPI(apiName)
	if err == nil {
		if len(containers) > 0 {
			err = DeleteContainers(apiName)
			if err != nil {
				errList = append(errList, err)
			}
		}
	} else {
		errList = append(errList, err)
	}

	_, err = FindAPISpec(apiName)
	if err == nil {
		_, err := files.DeleteDirIfPresent(filepath.Join(_localWorkspaceDir, "apis", apiName))
		if err != nil {
			errList = append(errList, ErrorFailedToDeleteAPISpec(filepath.Join(_localWorkspaceDir, "apis", apiName), err))
		}
	} else if errors.GetKind(err) == ErrCortexVersionMismatch {
		_, err := files.DeleteDirIfPresent(filepath.Join(_localWorkspaceDir, "apis", apiName))
		if err != nil {
			errList = append(errList, ErrorFailedToDeleteAPISpec(filepath.Join(_localWorkspaceDir, "apis", apiName), err))
		}
	} else {
		// only add error if it isn't ErrCortexVersionMismatch
		errList = append(errList, err)
	}

	return errors.FirstError(errList...)
}

func FindAPISpec(apiName string) (*spec.API, error) {
	apiWorkspace := filepath.Join(_localWorkspaceDir, "apis", apiName)
	if !files.IsDir(apiWorkspace) {
		return nil, ErrorAPINotDeployed(apiName)
	}

	filepaths, err := files.ListDirRecursive(apiWorkspace, false)
	if err != nil {
		return nil, errors.Wrap(err, "api", apiName)
	}

	var apiSpec spec.API
	for _, specPath := range filepaths {
		if strings.HasSuffix(filepath.Base(specPath), "-spec.msgpack") {
			apiSpecVersion := GetVersionFromAPISpecFilePath(specPath)
			if apiSpecVersion != consts.CortexVersion {
				return nil, ErrorCortexVersionMismatch(apiName, apiSpecVersion)
			}

			bytes, err := files.ReadFileBytes(specPath)
			if err != nil {
				return nil, errors.Wrap(err, "api", apiName)
			}
			err = msgpack.Unmarshal(bytes, &apiSpec)
			if err != nil {
				return nil, errors.Wrap(err, "api", apiName)
			}
			return &apiSpec, nil
		}
	}
	return nil, ErrorAPINotDeployed(apiName)
}

func GetVersionFromAPISpecFilePath(path string) string {
	fileName := filepath.Base(path)
	return strings.Split(fileName, "-")[0]
}
