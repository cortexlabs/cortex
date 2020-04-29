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
	"github.com/cortexlabs/cortex/pkg/lib/sets/strset"
	"github.com/cortexlabs/cortex/pkg/types/spec"
	"github.com/cortexlabs/cortex/pkg/types/userconfig"
)

var _deploymentID = "local"

func UpdateAPI(apiConfig *userconfig.API, cortexYAMLPath string, projectID string, awsClient *aws.Client) (*spec.API, string, error) {
	prevAPISpec, err := FindAPISpec(apiConfig.Name)
	if err != nil {
		if errors.GetKind(err) == ErrCortexVersionMismatch {
			DeleteAPI(apiConfig.Name, false)
		}

		if errors.GetKind(err) != ErrAPINotDeployed {
			return nil, "", err
		}
	}

	prevAPIContainers, err := GetContainersByAPI(apiConfig.Name)
	if err != nil {
		return nil, "", err
	}

	apiSpec := spec.GetAPISpec(apiConfig, projectID, _deploymentID)
	if apiConfig.Predictor.Model != nil {
		localModelCache, err := CacheModel(*apiConfig.Predictor.Model, awsClient)
		if err != nil {
			return nil, "", errors.Wrap(err, userconfig.ModelKey, userconfig.PredictorKey, apiConfig.Identify())
		}
		apiSpec.LocalModelCache = localModelCache
	}

	apiSpec.LocalProjectDir = filepath.Dir(cortexYAMLPath)

	keepCache := false
	if prevAPISpec != nil {
		prevModelID := ""
		if prevAPISpec.LocalModelCache != nil {
			prevModelID = prevAPISpec.LocalModelCache.ID
		}

		newModelID := ""
		if apiSpec.LocalModelCache != nil {
			newModelID = apiSpec.LocalModelCache.ID
		}

		if prevAPISpec.ID == apiSpec.ID && newModelID == prevModelID && prevAPISpec.Compute.Equals(apiSpec.Compute) {
			return apiSpec, fmt.Sprintf("%s is up to date", apiSpec.Name), nil
		}
		keepCache = newModelID == prevModelID
	}

	if prevAPISpec != nil || len(prevAPIContainers) != 0 {
		err = DeleteAPI(apiSpec.Name, keepCache)
		if err != nil {
			return nil, "", err
		}
	}

	err = writeAPISpec(apiSpec)
	if err != nil {
		DeleteAPI(apiSpec.Name, false)
		return nil, "", err
	}

	if err := DeployContainers(apiSpec, awsClient); err != nil {
		DeleteAPI(apiSpec.Name, false)
		return nil, "", err
	}

	if prevAPISpec == nil && len(prevAPIContainers) == 0 {
		return apiSpec, fmt.Sprintf("creating %s", apiSpec.Name), nil
	}

	return apiSpec, fmt.Sprintf("updating %s", apiSpec.Name), nil
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

func DeleteAPI(apiName string, keepCache bool) error {
	errList := []error{}
	modelsToDelete := strset.New()

	containers, err := GetContainersByAPI(apiName)
	if err == nil {
		for _, container := range containers {
			if _, ok := container.Labels["modelID"]; ok {
				modelsToDelete.Add(container.Labels["modelID"])
			}
		}

		if len(containers) > 0 {
			err = DeleteContainers(apiName)
			if err != nil {
				errList = append(errList, err)
			}
		}
	} else {
		errList = append(errList, err)
	}

	apiSpec, err := FindAPISpec(apiName)

	if err == nil {
		if apiSpec.LocalModelCache != nil {
			modelsToDelete.Add(apiSpec.LocalModelCache.ID)
		}
		_, err := files.DeleteDirIfPresent(filepath.Join(_localWorkspaceDir, "apis", apiName))
		if err != nil {
			errList = append(errList, errors.Wrap(ErrorFailedToDeleteAPISpec(filepath.Join(_localWorkspaceDir, "apis", apiName)), err.Error()))
		}
	} else if errors.GetKind(err) == ErrCortexVersionMismatch {
		_, err := files.DeleteDirIfPresent(filepath.Join(_localWorkspaceDir, "apis", apiName))
		if err != nil {
			errList = append(errList, errors.Wrap(ErrorFailedToDeleteAPISpec(filepath.Join(_localWorkspaceDir, "apis", apiName)), err.Error()))
		}
	} else {
		// only add error if it isn't ErrCortexVersionMismatch
		errList = append(errList, err)
	}

	if !keepCache && len(modelsToDelete) > 0 {
		modelsInUse := strset.New()
		apiSpecList, err := ListAPISpecs()
		errList = append(errList, err)

		for _, apiSpec := range apiSpecList {
			if apiSpec.LocalModelCache != nil && apiSpec.Name != apiName {
				modelsInUse.Add(apiSpec.LocalModelCache.ID)
			}
		}

		modelsToDelete.Subtract(modelsInUse)
		for modelID := range modelsToDelete {
			err := files.DeleteDir(filepath.Join(_modelCacheDir, modelID))
			if err != nil {
				errList = append(errList, err)
			}
		}
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
