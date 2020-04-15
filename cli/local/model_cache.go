package local

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/cortexlabs/cortex/pkg/lib/aws"
	"github.com/cortexlabs/cortex/pkg/lib/errors"
	"github.com/cortexlabs/cortex/pkg/lib/files"
	s "github.com/cortexlabs/cortex/pkg/lib/strings"
	"github.com/cortexlabs/cortex/pkg/lib/zip"
	"github.com/cortexlabs/cortex/pkg/types/spec"
	"github.com/cortexlabs/cortex/pkg/types/userconfig"
)

func CacheModel(modelPath string, awsClient *aws.Client) (*spec.ModelMount, error) {
	modelMount := spec.ModelMount{}

	if strings.HasPrefix(modelPath, "s3://") {
		bucket, prefix, err := aws.SplitS3Path(modelPath)
		if err != nil {
			return nil, err
		}
		hash, err := awsClient.HashS3Dir(bucket, prefix, nil)
		if err != nil {
			return nil, err
		}
		modelMount.ID = hash
	} else {
		hash, err := getLocalModelHash(modelPath)
		if err != nil {
			return nil, err
		}
		modelMount.ID = hash
	}

	modelDir := filepath.Join(ModelCacheDir, modelMount.ID)

	if files.IsFile(filepath.Join(modelDir, "_SUCCESS")) {
		return &modelMount, nil
	}

	err := ResetCachedModel(modelDir)
	if err != nil {
		return nil, err
	}

	if strings.HasPrefix(modelPath, "s3://") {
		fmt.Println(fmt.Sprintf("downloading model %s...", modelPath))
		bucket, prefix, err := aws.SplitS3Path(modelPath)
		if err != nil {
			return nil, err
		}
		if strings.HasSuffix(modelPath, ".zip") || strings.HasSuffix(modelPath, ".onnx") {
			err := awsClient.DownloadFileFromS3(bucket, prefix, filepath.Join(modelDir, filepath.Base(modelPath)))
			if err != nil {
				return nil, err
			}
			if strings.HasSuffix(modelPath, ".zip") {
				err := unzipAndValidate(modelPath, filepath.Join(modelDir, filepath.Base(modelPath)), modelDir)
				if err != nil {
					return nil, err
				}
			}
		} else {
			version := filepath.Base(prefix)
			err := awsClient.DownloadDirFromS3(bucket, prefix, filepath.Join(modelDir, version), true, nil)
			if err != nil {
				return nil, err
			}
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

	modelMount.HostPath = modelDir

	return &modelMount, nil
}

func unzipAndValidate(originalModelPath string, zipFile string, destPath string) error {
	fmt.Println(fmt.Sprintf("unzipping model %s...", originalModelPath))
	_, err := zip.UnzipFileToDir(zipFile, destPath)
	if err != nil {
		return err
	}

	isValid, err := spec.IsValidTensorFlowLocalDirectory(destPath)
	if err != nil {
		return errors.Wrap(err, userconfig.ModelKey)
	} else if !isValid {
		return errors.Wrap(ErrorInvalidTensorFlowZip(originalModelPath))
	}
	err = os.Remove(zipFile)
	if err != nil {
		return err
	}
	return nil
}

func getLocalModelHash(modelPath string) (string, error) {
	var err error
	modelHash := ""
	if files.IsDir(modelPath) {
		modelHash, err = files.HashDirectory(modelPath, files.IgnoreHiddenFiles, files.IgnoreHiddenFolders)
		if err != nil {
			return "", err
		}
	} else {
		modelHash, err = files.HashFile(modelPath)
		if err != nil {
			return "", err
		}
	}

	return modelHash, nil
}

func ResetCachedModel(modelDir string) error {
	err := files.DeleteDir(modelDir)
	if err != nil {
		return err
	}

	_, err = files.CreateDirIfMissing(modelDir)
	if err != nil {
		return err
	}

	return nil
}
