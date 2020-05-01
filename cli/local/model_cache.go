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

	"github.com/cortexlabs/cortex/pkg/lib/aws"
	"github.com/cortexlabs/cortex/pkg/lib/cron"
	"github.com/cortexlabs/cortex/pkg/lib/errors"
	"github.com/cortexlabs/cortex/pkg/lib/files"
	"github.com/cortexlabs/cortex/pkg/lib/print"
	s "github.com/cortexlabs/cortex/pkg/lib/strings"
	"github.com/cortexlabs/cortex/pkg/lib/zip"
	"github.com/cortexlabs/cortex/pkg/types/spec"
)

func CacheModel(modelPath string, awsClient *aws.Client) (*spec.LocalModelCache, error) {
	localModelCache := spec.LocalModelCache{}

	if strings.HasPrefix(modelPath, "s3://") {
		bucket, prefix, err := aws.SplitS3Path(modelPath)
		if err != nil {
			return nil, err
		}
		hash, err := awsClient.HashS3Dir(bucket, prefix, nil)
		if err != nil {
			return nil, err
		}
		localModelCache.ID = hash
	} else {
		hash, err := localModelHash(modelPath)
		if err != nil {
			return nil, err
		}
		localModelCache.ID = hash
	}

	modelDir := filepath.Join(_modelCacheDir, localModelCache.ID)

	if files.IsFile(filepath.Join(modelDir, "_SUCCESS")) {
		localModelCache.HostPath = modelDir
		return &localModelCache, nil
	}

	err := ResetModelCacheDir(modelDir)
	if err != nil {
		return nil, err
	}

	if strings.HasPrefix(modelPath, "s3://") {
		err := downloadModel(modelPath, modelDir, awsClient)
		if err != nil {
			return nil, err
		}
	} else {
		if strings.HasSuffix(modelPath, ".zip") {
			err := unzipAndValidate(modelPath, modelPath, modelDir)
			if err != nil {
				return nil, err
			}
		} else {
			fmt.Println(fmt.Sprintf("caching model %s...", modelPath))
			err := files.CopyDirOverwrite(strings.TrimSuffix(modelPath, "/"), s.EnsureSuffix(modelDir, "/"))
			if err != nil {
				return nil, err
			}
		}
	}

	err = files.MakeEmptyFile(filepath.Join(modelDir, "_SUCCESS"))
	if err != nil {
		return nil, err
	}

	fmt.Println("") // Newline to group all of the model information

	localModelCache.HostPath = modelDir

	return &localModelCache, nil
}

func downloadModel(modelPath string, modelDir string, awsClient *aws.Client) error {
	fmt.Printf("downloading model %s ..", modelPath)
	defer fmt.Print("\n")
	dotCron := cron.Run(print.Dot, nil, 2*time.Second)
	defer dotCron.Cancel()

	bucket, prefix, err := aws.SplitS3Path(modelPath)
	if err != nil {
		return err
	}

	if strings.HasSuffix(modelPath, ".zip") || strings.HasSuffix(modelPath, ".onnx") {
		localPath := filepath.Join(modelDir, filepath.Base(modelPath))
		err := awsClient.DownloadFileFromS3(bucket, prefix, localPath)
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
		err := awsClient.DownloadDirFromS3(bucket, prefix, filepath.Join(modelDir, tfModelVersion), true, nil)
		if err != nil {
			return err
		}
	}

	return nil
}

func unzipAndValidate(originalModelPath string, zipFile string, destPath string) error {
	fmt.Println(fmt.Sprintf("unzipping model %s...", originalModelPath))
	tmpDir := filepath.Join(filepath.Dir(destPath), filepath.Base(destPath)+"-tmp")
	err := files.CreateDir(tmpDir)
	if err != nil {
		return err
	}

	_, err = zip.UnzipFileToDir(zipFile, tmpDir)
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
