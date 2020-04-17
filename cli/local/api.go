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
	"os"
	"path/filepath"
	"strings"

	"github.com/cortexlabs/cortex/pkg/consts"
	"github.com/cortexlabs/cortex/pkg/lib/aws"
	"github.com/cortexlabs/cortex/pkg/lib/docker"
	"github.com/cortexlabs/cortex/pkg/lib/errors"
	"github.com/cortexlabs/cortex/pkg/lib/files"
	"github.com/cortexlabs/cortex/pkg/lib/msgpack"
	"github.com/cortexlabs/cortex/pkg/types/spec"
	"github.com/cortexlabs/cortex/pkg/types/userconfig"
)

var _deploymentID = "local"

func UpdateAPI(apiConfig *userconfig.API, projectID string, awsClient *aws.Client) (*spec.API, string, error) {
	err := docker.PullImage(apiConfig.Predictor.Image)
	if err != nil {
		return nil, "", err
	}

	if apiConfig.Predictor.Type == userconfig.TensorFlowPredictorType {
		err := docker.PullImage(apiConfig.Predictor.TFServeImage)
		if err != nil {
			return nil, "", err
		}
	}

	existingContainers, err := GetContainersByAPI(apiConfig.Name)
	if err != nil {
		return nil, "", err
	}

	apiSpec := spec.GetAPISpec(apiConfig, projectID, _deploymentID)
	if apiConfig.Predictor.Model != nil {
		modelMount, err := CacheModel(*apiConfig.Predictor.Model, awsClient)
		if err != nil {
			return nil, "", errors.Wrap(err, userconfig.ModelKey, userconfig.PredictorKey, apiConfig.Identify())
		}
		apiSpec.ModelMount = modelMount
	}

	if len(existingContainers) != 0 {
		prevContainerLabels := existingContainers[0].Labels
		newModelID := ""

		if apiSpec.ModelMount != nil {
			newModelID = apiSpec.ModelMount.ID
		}

		oldModelID := prevContainerLabels["modelID"]

		if prevContainerLabels["apiName"] == apiSpec.Name && prevContainerLabels["apiID"] == apiSpec.ID && newModelID == oldModelID {
			return apiSpec, fmt.Sprintf("%s is up to date", apiSpec.Name), nil
		}

		// keep the model cache if the model is being reused
		keepCache := apiSpec.ModelMount != nil && apiSpec.ModelMount.ID == oldModelID
		err = DeleteAPI(apiSpec.Name, keepCache)
		if err != nil {
			return nil, "", err
		}
	}

	apiBytes, err := msgpack.Marshal(apiSpec)
	if err != nil {
		return nil, "", err
	}

	err = os.MkdirAll(files.ParentDir(filepath.Join(_localWorkspaceDir, apiSpec.Key)), os.ModePerm)
	if err != nil {
		return nil, "", err
	}

	err = files.WriteFile(apiBytes, filepath.Join(_localWorkspaceDir, apiSpec.Key))
	if err != nil {
		return nil, "", err
	}

	if err := DeployContainers(apiSpec, awsClient); err != nil {
		return nil, "", err
	}

	if len(existingContainers) == 0 {
		return apiSpec, fmt.Sprintf("creating %s", apiSpec.Name), nil
	}

	return apiSpec, fmt.Sprintf("updating %s", apiSpec.Name), nil
}

func DeleteAPI(apiName string, keepCache bool) error {
	errList := []error{}
	var modelID string

	containers, err := GetContainersByAPI(apiName)
	errList = append(errList, err)
	if err == nil {
		for _, container := range containers {
			if _, ok := container.Labels["modelID"]; ok {
				modelID = container.Labels["modelID"]
			}
		}

		if len(containers) > 0 {
			errList = append(errList, DeleteContainers(apiName))
		}
	}

	apiSpec, err := FindAPISpec(apiName)
	if err == nil || errors.GetKind(err) == ErrCortexVersionMismatch {
		if apiSpec.ModelMount != nil {
			modelID = apiSpec.ModelMount.ID
		}
		_, err := files.DeleteDirIfPresent(filepath.Join(_localWorkspaceDir, "apis", apiName))
		if err != nil {
			errList = append(errList, errors.Wrap(ErrorFailedToDeleteAPISpec(filepath.Join(_localWorkspaceDir, "apis", apiName)), err.Error()))
		}

	} else {
		// only add error if it isn't ErrCortexVersionMismatch
		errList = append(errList, err)
	}

	if !keepCache && modelID != "" {
		apiSpecList, err := ListAllAPISpecs()
		errList = append(errList, err)

		foundAnotherUse := false
		for _, apiSpec := range apiSpecList {
			if apiSpec.ModelMount != nil && apiSpec.Name != apiName && apiSpec.ModelMount.ID == modelID {
				foundAnotherUse = true
			}
		}
		if !foundAnotherUse {
			err := files.DeleteDir(filepath.Join(_modelCacheDir, modelID))
			errList = append(errList, err)
		}
	}

	return errors.FirstError(errList...)
}

func FindAPISpec(apiName string) (spec.API, error) {
	apiWorkspace := filepath.Join(_localWorkspaceDir, "apis", apiName)
	if err := files.CheckDir(apiWorkspace); err != nil {
		return spec.API{}, ErrorAPINotDeployed(apiName)
	}

	filepaths, err := files.ListDirRecursive(apiWorkspace, false)
	if err != nil {
		return spec.API{}, errors.Wrap(err, "api", apiName)
	}

	var apiSpec spec.API
	for _, specPath := range filepaths {
		if strings.HasSuffix(specPath, "spec.msgpack") {
			apiSpecVersion := GetVersionFromAPISpecFilePath(specPath)
			if apiSpecVersion != consts.CortexVersion {
				return spec.API{}, ErrorCortexVersionMismatch(apiName, apiSpecVersion)
			}

			bytes, err := files.ReadFileBytes(specPath)
			if err != nil {
				return spec.API{}, errors.Wrap(err, "api", apiName)
			}
			err = msgpack.Unmarshal(bytes, &apiSpec)
			if err != nil {
				return spec.API{}, errors.Wrap(err, "api", apiName)
			}
			return apiSpec, nil
		}
	}
	return spec.API{}, ErrorAPINotDeployed(apiName)
}

func GetVersionFromAPISpecFilePath(path string) string {
	fileName := filepath.Base(path)
	return strings.Split(fileName, "-")[0]
}
