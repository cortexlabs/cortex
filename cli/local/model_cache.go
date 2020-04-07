package local

import (
	"crypto/md5"
	"encoding/hex"
	"fmt"
	"io"
	"path/filepath"
	"strings"

	"github.com/cortexlabs/cortex/pkg/lib/aws"
	"github.com/cortexlabs/cortex/pkg/lib/files"
	"github.com/cortexlabs/cortex/pkg/lib/pointer"
	s "github.com/cortexlabs/cortex/pkg/lib/strings"
	"github.com/cortexlabs/cortex/pkg/types/userconfig"
)

func CacheModel(api *userconfig.API) (string, error) {
	if strings.HasPrefix(*api.Predictor.Model, "s3://") {
		modelDir, err := getS3ModelCachePath(api)
		if err != nil {
			return "", err
		}

		if files.IsFile(filepath.Join(modelDir, "_SUCCESS")) {
			fmt.Println("already cached")
			if strings.HasSuffix(*api.Predictor.Model, ".zip") || strings.HasSuffix(*api.Predictor.Model, ".onnx") {
				_, key, err := aws.SplitS3Path(*api.Predictor.Model)
				if err != nil {
					return "", err
				}
				return filepath.Join(modelDir, filepath.Base(key)), nil
			}
			return modelDir, nil
		}

		err = ResetCachedModel(modelDir)
		if err != nil {
			return "", err
		}

		if strings.HasSuffix(*api.Predictor.Model, ".zip") || strings.HasSuffix(*api.Predictor.Model, ".onnx") {
			return S3DownloadObj(api, modelDir)
		}

		return S3DownloadDir(api, modelDir)
	}

	modelDir, err := getLocalModelCachePath(api)
	if err != nil {
		return "", err
	}

	if files.IsFile(filepath.Join(modelDir, "_SUCCESS")) {
		if files.IsFile(*api.Predictor.Model) {
			return filepath.Join(modelDir, filepath.Base(*api.Predictor.Model)), nil
		}
		return modelDir, nil
	}

	err = ResetCachedModel(modelDir)
	if err != nil {
		return "", err
	}

	if files.IsDir(*api.Predictor.Model) {
		err := files.Copy(strings.TrimSuffix(*api.Predictor.Model, "/"), s.EnsureSuffix(modelDir, "/"))
		if err != nil {
			return "", err
		}
		err = files.MakeEmptyFile(filepath.Join(modelDir, "_SUCCESS"))
		if err != nil {
			return "", err
		}

		return modelDir, nil
	}

	modelFilePath := filepath.Join(modelDir, filepath.Base(*api.Predictor.Model))
	err = files.Copy(*api.Predictor.Model, modelFilePath)
	if err != nil {
		return "", err
	}

	err = files.MakeEmptyFile(filepath.Join(modelDir, "_SUCCESS"))
	if err != nil {
		return "", err
	}

	return modelFilePath, nil
}

func getLocalModelCachePath(api *userconfig.API) (string, error) {
	var err error
	modelHash := ""
	if files.IsDir(*api.Predictor.Model) {
		modelHash, err = files.HashDirectory(*api.Predictor.Model, files.IgnoreHiddenFiles, files.IgnoreHiddenFolders)
		if err != nil {
			return "", err
		}
	} else {
		modelHash, err = files.HashFile(*api.Predictor.Model)
		if err != nil {
			return "", err
		}
	}
	modelDir := filepath.Join(ModelCacheDir, modelHash)

	return modelDir, nil
}

func getS3ModelCachePath(api *userconfig.API) (string, error) {
	awsClient, err := aws.NewFromEnvS3Path(*api.Predictor.Model)
	if err != nil {
		return "", err
	}

	// TODO fix max file count
	s3Objects, err := awsClient.ListS3PathPrefix(*api.Predictor.Model, pointer.Int64(1000))
	if err != nil {
		return "", err
	}

	if len(s3Objects) == 1000 {
		return "", ErrorTensorFlowDirTooManyFiles(999)
	}

	md5Hash := md5.New()
	for _, obj := range s3Objects {
		if strings.HasSuffix(*obj.Key, "/") {
			continue
		}

		io.WriteString(md5Hash, *obj.ETag)
	}

	modelPathHash := hex.EncodeToString((md5Hash.Sum(nil)))
	return filepath.Join(ModelCacheDir, modelPathHash), nil
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

func S3DownloadDir(api *userconfig.API, modelVersionDir string) (string, error) {
	awsClient, err := aws.NewFromEnvS3Path(*api.Predictor.Model)
	if err != nil {
		return "", err
	}

	s3Objects, err := awsClient.ListS3PathDir(*api.Predictor.Model, pointer.Int64(1000))
	if err != nil {
		return "", err
	}

	if len(s3Objects) == 1000 {
		return "", ErrorTensorFlowDirTooManyFiles(999)
	}

	bucket, fullPathKey, err := aws.SplitS3Path(*api.Predictor.Model)
	if err != nil {
		return "", err
	}

	tfVersion := filepath.Base(*api.Predictor.Model)

	for _, obj := range s3Objects {
		relativePath, err := filepath.Rel(fullPathKey, *obj.Key)
		if err != nil {
			return "", err
		}
		localPath := filepath.Join(modelVersionDir, tfVersion, relativePath)

		fmt.Println(fmt.Sprintf("Downloading from %s to %s (this may take a few minutes)...", *obj.Key, localPath))
		if strings.HasSuffix(*obj.Key, "/") {
			_, err = files.CreateDirIfMissing(localPath)
			if err != nil {
				return "", nil
			}
			continue
		}

		/*
			key: a.zip -> <hash>/<time>/a.zip
			key: a/b   -> <hash>/<time>/contents of b
			key: a/b.zip -> <hash>/<time>/b.zip
		*/
		_, err = files.CreateDirIfMissing(files.ParentDir(localPath))
		if err != nil {
			return "", nil
		}

		// TODO: use s3 download manager here?
		fileBytes, err := awsClient.ReadBytesFromS3(bucket, *obj.Key)
		if err != nil {
			return "", err
		}

		err = files.WriteFile(fileBytes, localPath)
		if err != nil {
			return "", err
		}
	}

	err = files.MakeEmptyFile(filepath.Join(modelVersionDir, "_SUCCESS"))
	if err != nil {
		return "", err
	}

	return modelVersionDir, nil
}

func S3DownloadObj(api *userconfig.API, modelVersionDir string) (string, error) {
	bucket, fullPathKey, err := aws.SplitS3Path(*api.Predictor.Model)
	if err != nil {
		return "", err
	}

	fmt.Println(fmt.Sprintf("Downloading from %s to %s (this may take a few minutes)...", *api.Predictor.Model, modelVersionDir))

	fileKey := filepath.Dir(fullPathKey)

	relativePath, err := filepath.Rel(fileKey, fullPathKey)
	if err != nil {
		return "", err
	}
	localPath := filepath.Join(modelVersionDir, relativePath)

	awsClient, err := aws.NewFromEnvS3Path(*api.Predictor.Model)
	if err != nil {
		return "", err
	}

	// TODO: use s3 download manager here?
	fileBytes, err := awsClient.ReadBytesFromS3(bucket, fullPathKey)
	if err != nil {
		return "", err
	}

	err = files.WriteFile(fileBytes, localPath)
	if err != nil {
		return "", err
	}

	err = files.MakeEmptyFile(filepath.Join(modelVersionDir, "_SUCCESS"))
	if err != nil {
		return "", err
	}

	return localPath, nil
}
