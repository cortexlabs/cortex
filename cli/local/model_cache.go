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
	"time"

	"github.com/cortexlabs/cortex/pkg/consts"
	"github.com/cortexlabs/cortex/pkg/lib/aws"
	"github.com/cortexlabs/cortex/pkg/lib/cron"
	"github.com/cortexlabs/cortex/pkg/lib/errors"
	"github.com/cortexlabs/cortex/pkg/lib/files"
	"github.com/cortexlabs/cortex/pkg/lib/print"
	"github.com/cortexlabs/cortex/pkg/lib/sets/strset"
	s "github.com/cortexlabs/cortex/pkg/lib/strings"
	"github.com/cortexlabs/cortex/pkg/types/spec"
	"github.com/cortexlabs/cortex/pkg/types/userconfig"
)

func CacheModels(apiSpec *spec.API, awsClient *aws.Client) error {
	var err error
	var wasAlreadyCached bool
	localModelCaches := make([]*spec.LocalModelCache, len(apiSpec.CuratedModelResources))

	modelsThatWereCachedAlready := 0
	for i, modelPath := range apiSpec.CuratedModelResources {
		localModelCaches[i], wasAlreadyCached, err = CacheModel(modelPath, awsClient)
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
		localModelCaches[i].TargetPath = apiSpec.CuratedModelResources[i].Name
	}
	apiSpec.LocalModelCaches = localModelCaches

	if len(localModelCaches) > modelsThatWereCachedAlready && len(localModelCaches) > 0 {
		fmt.Println("") // Newline to group all of the model information
	}

	return nil
}

func CacheModel(model spec.CuratedModelResource, awsClient *aws.Client) (*spec.LocalModelCache, bool, error) {
	localModelCache := spec.LocalModelCache{}
	var awsClientForBucket *aws.Client
	var err error

	if model.S3Path {
		awsClientForBucket, err = aws.NewFromClientS3Path(model.ModelPath, awsClient)
		if err != nil {
			return nil, false, err
		}
		bucket, prefix, err := aws.SplitS3Path(model.ModelPath)
		if err != nil {
			return nil, false, err
		}
		hash, err := awsClientForBucket.HashS3Dir(bucket, prefix, nil)
		if err != nil {
			return nil, false, err
		}
		localModelCache.ID = hash
	} else {
		hash, err := localModelHash(model.ModelPath)
		if err != nil {
			return nil, false, err
		}
		localModelCache.ID = hash
	}

	destModelDir := filepath.Join(_modelCacheDir, localModelCache.ID)

	if files.IsDir(destModelDir) {
		localModelCache.HostPath = destModelDir
		return &localModelCache, true, nil
	}

	err = ResetModelCacheDir(destModelDir)
	if err != nil {
		return nil, false, err
	}

	if model.S3Path {
		err := downloadModel(model, destModelDir, awsClientForBucket)
		if err != nil {
			return nil, false, err
		}
	} else {
		if len(model.Versions) == 1 {
			fmt.Println(fmt.Sprintf("￮ caching model %s (version %d) ...", model.Name, model.Versions[0]))
		} else if len(model.Versions) > 0 {
			fmt.Println(fmt.Sprintf("￮ caching model %s (versions %s) ...", model.Name, s.UserStrsAnd(model.Versions)))
		} else {
			if model.Name == consts.SingleModelName {
				fmt.Println("￮ caching model ...")
			} else {
				fmt.Println(fmt.Sprintf("￮ caching model %s ...", model.Name))
			}
		}
		if strings.HasSuffix(model.ModelPath, ".onnx") {
			if _, err := files.CreateDirIfMissing(filepath.Join(destModelDir, "1")); err != nil {
				return nil, false, err
			}
			err := files.CopyFileOverwrite(model.ModelPath, filepath.Join(destModelDir, "1", "default.onnx"))
			if err != nil {
				return nil, false, err
			}
		} else {
			err := files.CopyDirOverwrite(strings.TrimSuffix(model.ModelPath, "/"), s.EnsureSuffix(destModelDir, "/"))
			if err != nil {
				return nil, false, err
			}
		}
	}

	localModelCache.HostPath = destModelDir
	return &localModelCache, false, nil
}

func DeleteCachedModels(apiName string, modelsToDelete []string) error {
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
	err = DeleteCachedModelsByID(toDeleteModels.Slice())

	errList = append(errList, err)
	return errors.FirstError(errList...)
}

func DeleteCachedModelsByID(modelIDs []string) error {
	errList := []error{}
	for _, modelID := range modelIDs {
		err := files.DeleteDir(filepath.Join(_modelCacheDir, modelID))
		if err != nil {
			errList = append(errList, err)
		}
	}

	return errors.FirstError(errList...)
}

func downloadModel(model spec.CuratedModelResource, destModelDir string, awsClientForBucket *aws.Client) error {
	if len(model.Versions) == 1 {
		fmt.Println(fmt.Sprintf("￮ downloading model %s (version %d) ...", model.Name, model.Versions[0]))
	} else if len(model.Versions) > 0 {
		fmt.Println(fmt.Sprintf("￮ downloading model %s (version %v) ...", model.Name, model.Versions))
	} else {
		if model.Name == consts.SingleModelName {
			fmt.Println("￮ downloading model ...")
		} else {
			fmt.Println(fmt.Sprintf("￮ downloading model %s ...", model.Name))
		}
	}
	defer fmt.Print(" ✓\n")
	dotCron := cron.Run(print.Dot, nil, 2*time.Second)
	defer dotCron.Cancel()

	bucket, prefix, err := aws.SplitS3Path(model.ModelPath)
	if err != nil {
		return err
	}

	if strings.HasSuffix(model.ModelPath, ".onnx") {
		err := awsClientForBucket.DownloadFileFromS3(bucket, prefix, filepath.Join(destModelDir, "1", "default.onnx"))
		if err != nil {
			return err
		}
	} else {
		err := awsClientForBucket.DownloadDirFromS3(bucket, prefix, destModelDir, true, nil)
		if err != nil {
			return err
		}
	}

	return nil
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

func ResetModelCacheDir(modelDir string) error {
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
