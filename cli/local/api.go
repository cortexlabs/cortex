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
	"encoding/json"
	"fmt"
	"path/filepath"
	"strings"

	"github.com/cortexlabs/cortex/pkg/consts"
	"github.com/cortexlabs/cortex/pkg/lib/aws"
	"github.com/cortexlabs/cortex/pkg/lib/errors"
	"github.com/cortexlabs/cortex/pkg/lib/files"
	"github.com/cortexlabs/cortex/pkg/lib/prompt"
	"github.com/cortexlabs/cortex/pkg/lib/sets/strset"
	"github.com/cortexlabs/cortex/pkg/lib/telemetry"
	"github.com/cortexlabs/cortex/pkg/operator/schema"
	"github.com/cortexlabs/cortex/pkg/types"
	"github.com/cortexlabs/cortex/pkg/types/spec"
	"github.com/cortexlabs/cortex/pkg/types/userconfig"
)

var _deploymentID = "local"

func UpdateAPI(apiConfig *userconfig.API, models []spec.CuratedModelResource, projectRoot string, projectID string, deployDisallowPrompt bool, awsClient *aws.Client) (*schema.APIResponse, string, error) {
	telemetry.Event("operator.deploy", apiConfig.TelemetryEvent(types.LocalProviderType))

	var incompatibleVersion string
	encounteredVersionMismatch := false
	prevAPISpec, err := FindAPISpec(apiConfig.Name)
	if err != nil {
		if errors.GetKind(err) == ErrCortexVersionMismatch {
			encounteredVersionMismatch = true
			if incompatibleVersion, err = GetVersionFromAPISpec(apiConfig.Name); err != nil {
				return nil, "", err
			}

			incompatibleMinorVersion := strings.Join(strings.Split(incompatibleVersion, ".")[:2], ".")
			if consts.CortexVersionMinor != incompatibleMinorVersion && !deployDisallowPrompt {
				prompt.YesOrExit(
					fmt.Sprintf(
						"api %s was deployed using CLI version %s but the current CLI version is %s; "+
							"re-deploying %s with current CLI version %s might yield an unexpected outcome; any cached models won't be deleted\n\n"+
							"it is recommended to install version %s of the CLI (pip install cortex==%s), delete the API using version %s of the CLI, and then re-deploy the API using the latest version of the CLI\n\n"+
							"do you still want to re-deploy?",
						apiConfig.Name, incompatibleMinorVersion, consts.CortexVersionMinor, apiConfig.Name, consts.CortexVersionMinor, incompatibleMinorVersion, incompatibleVersion, incompatibleMinorVersion),
					"", "",
				)
			}
			if err := DeleteAPI(apiConfig.Name); err != nil {
				return nil, "", err
			}
		} else if errors.GetKind(err) != ErrAPINotDeployed {
			return nil, "", err
		}
	}

	prevAPIContainers, err := GetContainersByAPI(apiConfig.Name)
	if err != nil {
		return nil, "", err
	}

	newAPISpec := spec.GetAPISpec(apiConfig, projectID, _deploymentID, "")

	if newAPISpec != nil && TotalLocalModelVersions(models) > 0 {
		if err := CacheLocalModels(newAPISpec, models); err != nil {
			return nil, "", err
		}
	}

	newAPISpec.LocalProjectDir = projectRoot

	if areAPIsEqual(newAPISpec, prevAPISpec) {
		return toAPIResponse(newAPISpec), fmt.Sprintf("%s is up to date", newAPISpec.Resource.UserString()), nil
	}

	if prevAPISpec != nil || len(prevAPIContainers) != 0 {
		err = errors.FirstError(
			DeleteAPI(newAPISpec.Name),
			deleteCachedModels(newAPISpec.Name, prevAPISpec.SubtractModelIDs(newAPISpec)),
		)
		if err != nil {
			return nil, "", err
		}
	}

	err = writeAPISpec(newAPISpec)
	if err != nil {
		DeleteAPI(newAPISpec.Name)
		deleteCachedModels(newAPISpec.Name, newAPISpec.ModelIDs())
		return nil, "", err
	}

	if err := DeployContainers(newAPISpec, awsClient); err != nil {
		DeleteAPI(newAPISpec.Name)
		deleteCachedModels(newAPISpec.Name, newAPISpec.ModelIDs())
		return nil, "", err
	}

	if prevAPISpec == nil && len(prevAPIContainers) == 0 {
		if encounteredVersionMismatch {
			return toAPIResponse(newAPISpec), fmt.Sprintf(
				"creating api %s with current CLI version %s",
				newAPISpec.Name,
				consts.CortexVersion,
			), nil
		}

		return toAPIResponse(newAPISpec), fmt.Sprintf("creating %s", newAPISpec.Resource.UserString()), nil
	}

	return toAPIResponse(newAPISpec), fmt.Sprintf("updating %s", newAPISpec.Resource.UserString()), nil
}

func toAPIResponse(api *spec.API) *schema.APIResponse {
	return &schema.APIResponse{
		Spec:     *api,
		Endpoint: fmt.Sprintf("http://localhost:%d", *api.Networking.LocalPort),
	}
}

func writeAPISpec(apiSpec *spec.API) error {
	apiBytes, err := json.Marshal(apiSpec)
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
	if a1.SpecID != a2.SpecID {
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

	if ContainersHaveAPINameVolume(containers) {
		err = DeleteVolume(apiName)
		if err != nil {
			errList = append(errList, err)
		}
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
		}
		if strings.HasSuffix(filepath.Base(specPath), "-spec.json") {
			apiSpecVersion := GetVersionFromAPISpecFilePath(specPath)
			if apiSpecVersion != consts.CortexVersion {
				return nil, ErrorCortexVersionMismatch(apiName, apiSpecVersion)
			}

			bytes, err := files.ReadFileBytes(specPath)
			if err != nil {
				return nil, errors.Wrap(err, "api", apiName)
			}
			err = json.Unmarshal(bytes, &apiSpec)
			if err != nil {
				return nil, errors.Wrap(err, "api", apiName)
			}
			return &apiSpec, nil
		}
	}
	return nil, ErrorAPINotDeployed(apiName)
}

func GetVersionFromAPISpec(apiName string) (string, error) {
	apiWorkspace := filepath.Join(_localWorkspaceDir, "apis", apiName)
	if !files.IsDir(apiWorkspace) {
		return "", ErrorAPINotDeployed(apiName)
	}

	filepaths, err := files.ListDirRecursive(apiWorkspace, false)
	if err != nil {
		return "", errors.Wrap(err, "api", apiName)
	}

	for _, specPath := range filepaths {
		if strings.HasSuffix(filepath.Base(specPath), "-spec.json") || strings.HasSuffix(filepath.Base(specPath), "-spec.msgpack") {
			return GetVersionFromAPISpecFilePath(specPath), nil
		}
	}
	return "", ErrorAPINotDeployed(apiName)
}

func GetVersionFromAPISpecFilePath(path string) string {
	fileName := filepath.Base(path)
	return strings.Split(fileName, "-")[0]
}

func TotalLocalModelVersions(models []spec.CuratedModelResource) int {
	totalLocalModelVersions := 0
	for _, model := range models {
		if !model.LocalPath {
			continue
		}
		if len(model.Versions) > 0 {
			totalLocalModelVersions += len(model.Versions)
		} else {
			totalLocalModelVersions++
		}
	}
	return totalLocalModelVersions
}
