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

package spec

import (
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/cortexlabs/cortex/pkg/lib/aws"
	"github.com/cortexlabs/cortex/pkg/lib/errors"
	"github.com/cortexlabs/cortex/pkg/lib/files"
	"github.com/cortexlabs/cortex/pkg/lib/pointer"
	"github.com/cortexlabs/cortex/pkg/lib/sets/strset"
	s "github.com/cortexlabs/cortex/pkg/lib/strings"
	"github.com/cortexlabs/cortex/pkg/types/userconfig"
)

func surgeOrUnavailableValidator(str string) (string, error) {
	if strings.HasSuffix(str, "%") {
		parsed, ok := s.ParseInt32(strings.TrimSuffix(str, "%"))
		if !ok {
			return "", ErrorInvalidSurgeOrUnavailable(str)
		}
		if parsed < 0 || parsed > 100 {
			return "", ErrorInvalidSurgeOrUnavailable(str)
		}
	} else {
		parsed, ok := s.ParseInt32(str)
		if !ok {
			return "", ErrorInvalidSurgeOrUnavailable(str)
		}
		if parsed < 0 {
			return "", ErrorInvalidSurgeOrUnavailable(str)
		}
	}

	return str, nil
}

// Verifies if modelName is found in models slice.
func isModelNameIn(models []userconfig.ModelResource, modelName string) bool {
	for _, model := range models {
		if model.Name == modelName {
			return true
		}
	}
	return false
}

// Verifies if m has been initialized.
func isMultiModelFieldSet(m *userconfig.MultiModels) bool {
	if len(m.Paths) == 0 && m.Dir == nil && m.CacheSize == nil && m.DiskCacheSize == nil && m.SignatureKey == nil {
		return false
	}
	return true
}

func modelResourceToCurated(modelResources []userconfig.ModelResource) []CuratedModelResource {
	models := []CuratedModelResource{}
	for _, model := range modelResources {
		models = append(models, CuratedModelResource{
			ModelResource: userconfig.ModelResource{
				Name:         model.Name,
				ModelPath:    model.ModelPath,
				SignatureKey: model.SignatureKey,
			},
		})
	}

	return models
}

// Retrieves the objects found in the path directory.
//
// The model name is determined from the objects' names found in the path directory minus the extension if there's one.
// Path can either be an S3 path or a local system path - in the latter case, the returned paths will be in absolute form.
func retrieveModelsResourcesFromPath(path string, projectFiles ProjectFiles, awsClient *aws.Client) ([]userconfig.ModelResource, error) {
	models := []userconfig.ModelResource{}

	if aws.IsValidS3Path(path) {
		awsClientForBucket, err := aws.NewFromClientS3Path(path, awsClient)
		if err != nil {
			return models, err
		}

		if isDir, err := awsClientForBucket.IsS3PathDir(path); err != nil {
			return models, err
		} else if isDir {
			modelPaths, err := awsClientForBucket.GetNLevelsDeepFromS3Path(path, 1, true, nil)
			if err != nil {
				return models, err
			}
			var bucket string
			bucket, _, err = aws.SplitS3Path(path)
			if err != nil {
				return models, err
			}

			for _, modelPath := range modelPaths {
				modelName := strings.Split(filepath.Base(modelPath), ".")[0]
				if !isModelNameIn(models, modelName) {
					models = append(models, userconfig.ModelResource{
						Name:      modelName,
						ModelPath: aws.S3Path(bucket, modelPath),
					})
				} else {
					return []userconfig.ModelResource{}, ErrorS3ModelNameDuplicate(path, modelName)
				}
			}
		} else {
			return models, ErrorS3DirNotFound(path)
		}

	} else {
		var err error
		if strings.HasPrefix(path, "~/") {
			path, err = files.EscapeTilde(path)
			if err != nil {
				return models, err
			}
		} else {
			path = files.RelToAbsPath(path, projectFiles.ProjectDir())
		}

		var fi os.FileInfo
		fi, err = os.Stat(path)
		if err != nil {
			return models, ErrorInvalidPath(path)
		}
		if !fi.Mode().IsDir() {
			return models, ErrorInvalidDirPath(path)
		}

		var file *os.File
		file, err = os.Open(path)
		if err != nil {
			return models, err
		}

		var modelObjects []string
		modelObjects, err = file.Readdirnames(0)
		if err != nil {
			return models, err
		}

		for _, modelObject := range modelObjects {
			modelName := strings.Split(modelObject, ".")[0]
			if !isModelNameIn(models, modelName) {
				models = append(models, userconfig.ModelResource{
					Name:      modelName,
					ModelPath: filepath.Join(path, modelObject),
				})
			} else {
				return []userconfig.ModelResource{}, ErrorS3ModelNameDuplicate(path, modelName)
			}
		}
	}

	return models, nil
}

func getTFServingExportFromS3Path(path string, isNeuronExport bool, awsClientForBucket *aws.Client) (string, error) {
	if isValidTensorFlowS3Directory(path, awsClientForBucket) {
		return path, nil
	}

	bucket, _, err := aws.SplitS3Path(path)
	if err != nil {
		return "", err
	}

	objects, err := awsClientForBucket.ListS3PathDir(path, false, pointer.Int64(1000))
	if err != nil {
		return "", err
	} else if len(objects) == 0 {
		return "", errors.Wrap(ErrorInvalidTensorFlowModelPath(), path)
	}

	highestVersion := int64(0)
	var highestPath string
	for _, object := range objects {
		if !strings.HasSuffix(*object.Key, "saved_model.pb") {
			continue
		}

		keyParts := strings.Split(*object.Key, "/")
		versionStr := keyParts[len(keyParts)-1]
		version, err := strconv.ParseInt(versionStr, 10, 64)
		if err != nil {
			version = 0
		}

		possiblePath := "s3://" + filepath.Join(bucket, filepath.Join(keyParts[:len(keyParts)-1]...))

		if version >= highestVersion {
			if isNeuronExport && isValidNeuronTensorFlowS3Directory(possiblePath, awsClientForBucket) {
				highestVersion = version
				highestPath = possiblePath
			}
			if !isNeuronExport && isValidTensorFlowS3Directory(possiblePath, awsClientForBucket) {
				highestVersion = version
				highestPath = possiblePath
			}
		}
	}

	return highestPath, nil
}

// isValidTensorFlowS3Directory checks that the path contains a valid S3 directory for TensorFlow models
// Must contain the following structure:
// - 1523423423/ (version prefix, usually a timestamp)
// 		- saved_model.pb
//		- variables/
//			- variables.index
//			- variables.data-00000-of-00001 (there are a variable number of these files)
func isValidTensorFlowS3Directory(path string, awsClientForBucket *aws.Client) bool {
	if valid, err := awsClientForBucket.IsS3PathFile(
		aws.JoinS3Path(path, "saved_model.pb"),
		aws.JoinS3Path(path, "variables/variables.index"),
	); err != nil || !valid {
		return false
	}

	if valid, err := awsClientForBucket.IsS3PathPrefix(
		aws.JoinS3Path(path, "variables/variables.data-00000-of"),
	); err != nil || !valid {
		return false
	}
	return true
}

// isValidNeuronTensorFlowS3Directory checks that the path contains a valid S3 directory for Neuron TensorFlow models
// Must contain the following structure:
// - 1523423423/ (version prefix, usually a timestamp)
// 		- saved_model.pb
func isValidNeuronTensorFlowS3Directory(path string, awsClient *aws.Client) bool {
	if valid, err := awsClient.IsS3PathFile(
		aws.JoinS3Path(path, "saved_model.pb"),
	); err != nil || !valid {
		return false
	}

	return true
}

func GetTFServingExportFromLocalPath(path string) (string, error) {
	if err := files.CheckDir(path); err != nil {
		return "", err
	}
	paths, err := files.ListDirRecursive(path, false, files.IgnoreHiddenFiles, files.IgnoreHiddenFolders)
	if err != nil {
		return "", err
	}

	if len(paths) == 0 {
		return "", ErrorDirIsEmpty(path)
	}

	highestVersion := int64(0)
	var highestPath string

	for _, path := range paths {
		if strings.HasSuffix(path, "saved_model.pb") {
			possiblePath := filepath.Dir(path)

			versionStr := filepath.Base(possiblePath)
			version, err := strconv.ParseInt(versionStr, 10, 64)
			if err != nil {
				version = 0
			}

			validTFDirectory, err := IsValidTensorFlowLocalDirectory(possiblePath)
			if err != nil {
				return "", err
			}
			if version > highestVersion && validTFDirectory {
				highestVersion = version
				highestPath = possiblePath
			}
		}
	}

	return highestPath, nil
}

func IsValidTensorFlowLocalDirectory(path string) (bool, error) {
	paths, err := files.ListDirRecursive(path, true, files.IgnoreHiddenFiles, files.IgnoreHiddenFolders)
	if err != nil {
		return false, err
	}
	pathSet := strset.New(paths...)

	if !(pathSet.Has("saved_model.pb") && pathSet.Has("variables/variables.index")) {
		return false, nil
	}

	for _, path := range paths {
		if strings.HasPrefix(path, "variables/variables.data-00000-of") {
			return true, nil
		}
	}

	return false, nil
}

func FindDuplicateNames(apis []userconfig.API) []userconfig.API {
	names := make(map[string][]userconfig.API)

	for _, api := range apis {
		names[api.Name] = append(names[api.Name], api)
	}

	for name := range names {
		if len(names[name]) > 1 {
			return names[name]
		}
	}

	return nil
}

func checkDuplicateModelNames(modelResources []*userconfig.ModelResource) error {
	names := strset.New()

	for _, modelResource := range modelResources {
		if names.Has(modelResource.Name) {
			return ErrorDuplicateModelNames(modelResource.Name)
		}
		names.Add(modelResource.Name)
	}

	return nil
}
