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
		if errors.GetKind(err) != ErrAPINotDeployed {
			return nil, "", err
		}
	}

	prevAPIContainers, err := GetContainersByAPI(apiConfig.Name)
	if err != nil {
		return nil, "", err
	}

	apiSpec := spec.GetAPISpec(apiConfig, projectID, _deploymentID)

	// apiConfig.Predictor.Model was already added to apiConfig.Predictor.Models for ease of use
	if len(apiConfig.Predictor.Models) > 0 {
		localModelCaches, err := CacheModels(apiSpec, awsClient)
		if err != nil {
			return nil, "", err
		}
		apiSpec.LocalModelCaches = localModelCaches
	}

	apiSpec.LocalProjectDir = filepath.Dir(cortexYAMLPath)

	if areAPIsEqual(apiSpec, prevAPISpec) {
		return apiSpec, fmt.Sprintf("%s is up to date", apiSpec.Name), nil
	}

	if prevAPISpec != nil || len(prevAPIContainers) != 0 {
		err = DeleteAPI(apiSpec.Name, prevAPISpec, apiSpec)
		if err != nil {
			return nil, "", err
		}
	}

	err = writeAPISpec(apiSpec)
	if err != nil {
		DeleteAPI(apiSpec.Name, apiSpec, nil)
		return nil, "", err
	}

	if err := DeployContainers(apiSpec, awsClient); err != nil {
		DeleteAPI(apiSpec.Name, apiSpec, nil)
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

func areAPIsEqual(a1, a2 *spec.API) bool {
	if a1 != nil && a2 != nil {
		return strset.FromSlice(a1.ModelIDs()).IsEqual(strset.FromSlice(a2.ModelIDs())) && a1.ID == a2.ID && a1.Compute.Equals(a2.Compute)
	} else if a1 == nil && a2 == nil {
		return true
	} else {
		return false
	}
}

// DeleteAPI deletes a locally-deployed API by removing its containers, its workspace, and its models.
// apiName is the name of the API within Cortex, prevAPISpec is the current existing API and newAPISpec
// is the new API that'll be deployed soon after removing the existing API.
// If prevAPISpec is nil & newAPISpec is nil, no models are removed.
// If prevAPISpec is !nil & newAPISpec is nil, all models belonging to prevAPISpec are removed.
// If prevAPISPec is !nil & newAPISpec is !nil, all models belonging to prevAPISpec that are not found in newAPISpec are removed.
func DeleteAPI(apiName string, prevAPISpec *spec.API, newAPISpec *spec.API) error {
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

	if prevAPISpec != nil {
		prevModelIDs := strset.FromSlice(prevAPISpec.ModelIDs())
		newModelIDs := strset.FromSlice(newAPISpec.ModelIDs())

		if !prevModelIDs.IsEqual(newModelIDs) {
			toDeleteModels := strset.Difference(prevModelIDs, newModelIDs)

			for modelID := range toDeleteModels {
				err := files.DeleteDir(filepath.Join(_modelCacheDir, modelID))
				if err != nil {
					errList = append(errList, err)
				}
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
