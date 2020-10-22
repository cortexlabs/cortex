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
	"time"

	"github.com/cortexlabs/cortex/cli/types/flags"
	"github.com/cortexlabs/cortex/pkg/lib/archive"
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

func CacheModels(apiSpec *spec.API, awsClient *aws.Client) ([]*spec.LocalModelCache, error) {
	modelPaths := make([]string, len(apiSpec.Predictor.Models))
	for i, modelResource := range apiSpec.Predictor.Models {
		modelPaths[i] = modelResource.ModelPath
	}

	uncachedModelCount := 0

	localModelCaches := make([]*spec.LocalModelCache, len(modelPaths))
	for i, modelPath := range modelPaths {
		var err error
		modelCacheID, err := modelCacheID(modelPath, awsClient)
		if err != nil {
			if apiSpec.Predictor.ModelPath != nil {
				return nil, errors.Wrap(err, apiSpec.Identify(), userconfig.PredictorKey, userconfig.ModelPathKey)
			}
			return nil, errors.Wrap(err, apiSpec.Identify(), userconfig.PredictorKey, userconfig.ModelsKey, apiSpec.Predictor.Models[i].Name, userconfig.ModelPathKey)
		}

		localModelCache := spec.LocalModelCache{
			ID:         modelCacheID,
			HostPath:   filepath.Join(_modelCacheDir, modelCacheID),
			TargetPath: apiSpec.Predictor.Models[i].Name,
		}

		if !files.IsFile(filepath.Join(localModelCache.HostPath, "_SUCCESS")) {
			err = cacheModel(modelPath, localModelCache, awsClient)
			if err != nil {
				if apiSpec.Predictor.ModelPath != nil {
					return nil, errors.Wrap(err, apiSpec.Identify(), userconfig.PredictorKey, userconfig.ModelPathKey)
				}
				return nil, errors.Wrap(err, apiSpec.Identify(), userconfig.PredictorKey, userconfig.ModelsKey, apiSpec.Predictor.Models[i].Name, userconfig.ModelPathKey)
			}
			uncachedModelCount++
		}

		localModelCaches[i] = &localModelCache
	}

	if uncachedModelCount > 0 {
		localPrintln("") // Newline to group all of the model information
	}

	return localModelCaches, nil
}

func modelCacheID(modelPath string, awsClient *aws.Client) (string, error) {
	if strings.HasPrefix(modelPath, "s3://") {
		awsClientForBucket, err := aws.NewFromClientS3Path(modelPath, awsClient)
		if err != nil {
			return "", err
		}
		bucket, prefix, err := aws.SplitS3Path(modelPath)
		if err != nil {
			return "", err
		}
		hash, err := awsClientForBucket.HashS3Dir(bucket, prefix, nil)
		if err != nil {
			return "", err
		}
		return hash, nil
	}

	hash, err := localModelHash(modelPath)
	if err != nil {
		return "", err
	}
	return hash, nil
}

func cacheModel(modelPath string, localModelCache spec.LocalModelCache, awsClient *aws.Client) error {
	modelDir := localModelCache.HostPath

	if files.IsFile(filepath.Join(modelDir, "_SUCCESS")) {
		return nil
	}

	err := ResetModelCacheDir(modelDir)
	if err != nil {
		return err
	}

	if strings.HasPrefix(modelPath, "s3://") {
		awsClientForBucket, err := aws.NewFromClientS3Path(modelPath, awsClient)
		if err != nil {
			return err
		}

		err = downloadModel(modelPath, modelDir, awsClientForBucket)
		if err != nil {
			return err
		}
	} else {
		if strings.HasSuffix(modelPath, ".zip") {
			err := unzipAndValidate(modelPath, modelPath, modelDir)
			if err != nil {
				return err
			}
		} else if strings.HasSuffix(modelPath, ".onnx") {
			localPrintln(fmt.Sprintf("￮ caching model %s ...", modelPath))
			err := files.CopyFileOverwrite(modelPath, filepath.Join(modelDir, filepath.Base(modelPath)))
			if err != nil {
				return err
			}
		} else {
			localPrintln(fmt.Sprintf("￮ caching model %s ...", modelPath))
			tfModelVersion := filepath.Base(modelPath)
			err := files.CopyDirOverwrite(strings.TrimSuffix(modelPath, "/"), s.EnsureSuffix(filepath.Join(modelDir, tfModelVersion), "/"))
			if err != nil {
				return err
			}
		}
	}

	err = files.MakeEmptyFile(filepath.Join(modelDir, "_SUCCESS"))
	if err != nil {
		return err
	}

	return nil
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

func downloadModel(modelPath string, modelDir string, awsClientForBucket *aws.Client) error {
	localPrintf("￮ downloading model %s ", modelPath)
	defer localPrint(" ✓\n")
	if OutputType != flags.JSONOutputType {
		dotCron := cron.Run(print.Dot, nil, 2*time.Second)
		defer dotCron.Cancel()
	}

	bucket, prefix, err := aws.SplitS3Path(modelPath)
	if err != nil {
		return err
	}

	if strings.HasSuffix(modelPath, ".zip") || strings.HasSuffix(modelPath, ".onnx") {
		localPath := filepath.Join(modelDir, filepath.Base(modelPath))
		err := awsClientForBucket.DownloadFileFromS3(bucket, prefix, localPath)
		if err != nil {
			return err
		}
		if strings.HasSuffix(modelPath, ".zip") {
			err := unzipAndValidate(modelPath, localPath, modelDir)
			if err != nil {
				return err
			}
			err = os.Remove(localPath)
			if err != nil {
				return err
			}
		}
	} else {
		tfModelVersion := filepath.Base(prefix)
		err := awsClientForBucket.DownloadDirFromS3(bucket, prefix, filepath.Join(modelDir, tfModelVersion), true, nil)
		if err != nil {
			return err
		}
	}

	return nil
}

func unzipAndValidate(originalModelPath string, zipFile string, destPath string) error {
	localPrintln(fmt.Sprintf("￮ unzipping model %s ...", originalModelPath))
	tmpDir := filepath.Join(filepath.Dir(destPath), filepath.Base(destPath)+"-tmp")
	err := files.CreateDir(tmpDir)
	if err != nil {
		return err
	}

	_, err = archive.UnzipFileToDir(zipFile, tmpDir)
	if err != nil {
		return err
	}

	// returns a tensorflow directory with the version as the suffix of the path
	tensorflowDir, err := spec.GetTFServingExportFromLocalPath(tmpDir)
	if err != nil {
		return err
	}

	isValid, err := spec.IsValidTensorFlowLocalDirectory(tensorflowDir)
	if err != nil {
		return err
	} else if !isValid {
		return ErrorInvalidTensorFlowZip()
	}

	destPathWithVersion := filepath.Join(destPath, filepath.Base(tensorflowDir))
	err = os.Rename(strings.TrimSuffix(tensorflowDir, "/"), strings.TrimSuffix(destPathWithVersion, "/"))
	if err != nil {
		return errors.WithStack(err)
	}

	err = files.DeleteDir(tmpDir)
	if err != nil {
		return err
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
