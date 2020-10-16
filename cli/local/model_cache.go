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
	"github.com/cortexlabs/cortex/pkg/lib/errors"
	"github.com/cortexlabs/cortex/pkg/lib/files"
	"github.com/cortexlabs/cortex/pkg/lib/sets/strset"
	s "github.com/cortexlabs/cortex/pkg/lib/strings"
	"github.com/cortexlabs/cortex/pkg/types/spec"
	"github.com/cortexlabs/cortex/pkg/types/userconfig"
)

func CacheLocalModels(apiSpec *spec.API, onlyLocalModels bool) error {
	var err error
	var wasAlreadyCached bool
	var localModelCache *spec.LocalModelCache
	localModelCaches := make([]*spec.LocalModelCache, 0)

	modelsThatWereCachedAlready := 0
	for i, model := range apiSpec.CuratedModelResources {
		if onlyLocalModels && model.S3Path {
			continue
		}

		localModelCache, wasAlreadyCached, err = cacheLocalModel(model)
		if err != nil {
			if apiSpec.Predictor.ModelPath != nil {
				return errors.Wrap(err, apiSpec.Identify(), userconfig.PredictorKey, userconfig.ModelPathKey)
			} else if apiSpec.Predictor.Models != nil && apiSpec.Predictor.Models.Dir != nil {
				return errors.Wrap(err, apiSpec.Identify(), userconfig.PredictorKey, userconfig.ModelsKey, userconfig.ModelsDirKey, apiSpec.CuratedModelResources[i].Name, *apiSpec.Predictor.Models.Dir)
			}
			return errors.Wrap(err, apiSpec.Identify(), userconfig.PredictorKey, userconfig.ModelsKey, userconfig.ModelsPathsKey, apiSpec.CuratedModelResources[i].Name, userconfig.ModelPathKey)
		}
		if wasAlreadyCached {
			modelsThatWereCachedAlready++
		}
		if len(model.Versions) == 0 {
			localModelCache.TargetPath = filepath.Join(apiSpec.CuratedModelResources[i].Name, "1")
		} else {
			localModelCache.TargetPath = apiSpec.CuratedModelResources[i].Name
		}

		localModelCaches = append(localModelCaches, localModelCache)
	}
	apiSpec.LocalModelCaches = localModelCaches

	if len(localModelCaches) > modelsThatWereCachedAlready && len(localModelCaches) > 0 {
		fmt.Println("") // Newline to group all of the model information
	}

	return nil
}

func cacheLocalModel(model spec.CuratedModelResource) (*spec.LocalModelCache, bool, error) {
	localModelCache := spec.LocalModelCache{}
	var err error

	if model.S3Path {
		return nil, false, nil
	}

	hash, err := localModelHash(model.ModelPath)
	if err != nil {
		return nil, false, err
	}
	localModelCache.ID = hash

	destModelDir := filepath.Join(_modelCacheDir, localModelCache.ID)

	if files.IsDir(destModelDir) {
		localModelCache.HostPath = destModelDir
		return &localModelCache, true, nil
	}

	err = resetModelCacheDir(destModelDir)
	if err != nil {
		return nil, false, err
	}
	if len(model.Versions) == 0 {
		if _, err := files.CreateDirIfMissing(filepath.Join(destModelDir, "1")); err != nil {
			return nil, false, err
		}
	}

	if model.Name == consts.SingleModelName {
		switch numVersions := len(model.Versions); numVersions {
		case 0:
			fmt.Println("￮ caching model ...")
		case 1:
			fmt.Println(fmt.Sprintf("￮ caching model (version %d) ...", model.Versions[0]))
		default:
			fmt.Println(fmt.Sprintf("￮ caching model (versions %s) ...", s.UserStrsAnd(model.Versions)))
		}

	} else {
		switch numVersions := len(model.Versions); numVersions {
		case 0:
			fmt.Println(fmt.Sprintf("￮ caching model %s ...", model.Name))
		case 1:
			fmt.Println(fmt.Sprintf("￮ caching model %s (version %d) ...", model.Name, model.Versions[0]))
		default:
			fmt.Println(fmt.Sprintf("￮ caching model %s (versions %s) ...", model.Name, s.UserStrsAnd(model.Versions)))
		}
	}

	if len(model.Versions) == 0 {
		destModelDir = filepath.Join(destModelDir, "1")
	}
	if err := files.CopyDirOverwrite(strings.TrimSuffix(model.ModelPath, "/"), s.EnsureSuffix(destModelDir, "/")); err != nil {
		return nil, false, err
	}

	localModelCache.HostPath = destModelDir
	return &localModelCache, false, nil
}

func deleteCachedModels(apiName string, modelsToDelete []string) error {
	var errList []error
	modelsInUse := strset.New()
	apiSpecList, err := ListAPISpecs()
	errList = append(errList, err)

	if err == nil {
		for _, apiSpec := range apiSpecList {
			if len(apiSpec.LocalModelCaches) > 0 && apiSpec.Name != apiName {
				for _, modelCache := range apiSpec.LocalModelCaches {
					modelsInUse.Add(modelCache.ID)
				}
			}
		}
	}

	toDeleteModels := strset.Difference(
		strset.FromSlice(modelsToDelete),
		modelsInUse,
	)
	err = deleteCachedModelsByID(toDeleteModels.Slice())

	errList = append(errList, err)
	return errors.FirstError(errList...)
}

func deleteCachedModelsByID(modelIDs []string) error {
	errList := []error{}
	for _, modelID := range modelIDs {
		err := files.DeleteDir(filepath.Join(_modelCacheDir, modelID))
		if err != nil {
			errList = append(errList, err)
		}
	}

	return errors.FirstError(errList...)
}

func localModelHash(modelPath string) (string, error) {
	var err error
	modelHash := ""
	if files.IsDir(modelPath) {
		modelHash, err = files.HashDirectory(modelPath, files.IgnoreHiddenFiles, files.IgnoreHiddenFolders)
		if err != nil {
			return "", err
		}
	} else {
		if err := files.CheckFile(modelPath); err != nil {
			return "", err
		}
		modelHash, err = files.HashFile(modelPath)
		if err != nil {
			return "", err
		}
	}

	return modelHash, nil
}

func resetModelCacheDir(modelDir string) error {
	_, err := files.DeleteDirIfPresent(modelDir)
	if err != nil {
		return err
	}

	_, err = files.CreateDirIfMissing(modelDir)
	if err != nil {
		return err
	}

	return nil
}
