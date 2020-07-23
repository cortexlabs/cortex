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
	"github.com/cortexlabs/cortex/pkg/lib/slices"
	s "github.com/cortexlabs/cortex/pkg/lib/strings"
	"github.com/cortexlabs/cortex/pkg/types/userconfig"
)

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

// Returns absolute path of "path" based on "basedir".
func absolutePath(path, basedir string) (string, error) {
	var err error
	if strings.HasPrefix(path, "~/") {
		path, err = files.EscapeTilde(path)
		if err != nil {
			return "", err
		}
	} else {
		path = files.RelToAbsPath(path, basedir)
	}

	return path, nil
}

func modelResourceToCurated(modelResources []userconfig.ModelResource, projectFiles ProjectFiles) ([]CuratedModelResource, error) {
	models := []CuratedModelResource{}
	var err error
	for _, model := range modelResources {
		isS3Path := strings.HasPrefix(model.ModelPath, "s3://")
		if !isS3Path {
			model.ModelPath, err = absolutePath(model.ModelPath, projectFiles.ProjectDir())
			if err != nil {
				return []CuratedModelResource{}, err
			}
		}
		models = append(models, CuratedModelResource{
			ModelResource: userconfig.ModelResource{
				Name:         model.Name,
				ModelPath:    model.ModelPath,
				SignatureKey: model.SignatureKey,
			},
			S3Path: isS3Path,
		})
	}

	return models, nil
}

// Retrieves the model objects found in the S3/local path directory.
//
// The model name is determined from the objects' names found in the path directory.
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
			modelPaths, err := awsClientForBucket.GetNLevelsDeepFromS3Path(path, 1, false, pointer.Int64(20000))
			if err != nil {
				return models, err
			}
			var bucket string
			bucket, _, err = aws.SplitS3Path(path)
			if err != nil {
				return models, err
			}

			for _, modelPath := range modelPaths {
				models = append(models, userconfig.ModelResource{
					Name:      filepath.Base(modelPath),
					ModelPath: aws.S3Path(bucket, modelPath),
				})
			}
		} else {
			return models, ErrorS3DirNotFound(path)
		}

	} else {
		var err error
		path, err = absolutePath(path, projectFiles.ProjectDir())
		if err != nil {
			return models, err
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

		for _, modelName := range modelObjects {
			models = append(models, userconfig.ModelResource{
				Name:      modelName,
				ModelPath: filepath.Join(path, modelName),
			})
		}
	}

	return models, nil
}

// getTFServingVersionsFromS3Path checks that the path contains a valid S3 directory for (Neuron) TensorFlow models:
//
// For TensorFlow models:
// - model-name
// 		- 1523423423/ (version prefix, usually a timestamp)
//			- saved_model.pb
// 			- variables/
//				- variables.index
//				- variables.data-00000-of-00001 (there are a variable number of these files)
// 		- 2434389194/ (version prefix, usually a timestamp)
// 			- saved_model.pb
//			- variables/
//				- variables.index
//				- variables.data-00000-of-00001 (there are a variable number of these files)
//   ...
//
// For Neuron TensorFlow models:
// - model-name
// 		- 1523423423/ (version prefix, usually a timestamp)
// 			- saved_model.pb
// 		- 2434389194/ (version prefix, usually a timestamp)
//			- saved_model.pb
// 		...
//
func getTFServingVersionsFromS3Path(path string, isNeuronExport bool, awsClientForBucket *aws.Client) ([]int64, error) {
	bucket, _, err := aws.SplitS3Path(path)
	if err != nil {
		return []int64{}, err
	}

	objects, err := awsClientForBucket.ListS3PathDir(path, false, pointer.Int64(1000))
	if err != nil {
		return []int64{}, err
	} else if len(objects) == 0 {
		return []int64{}, errors.Wrap(ErrorInvalidTensorFlowModelPath(), path)
	}

	versions := []int64{}
	for _, object := range objects {
		if !strings.HasSuffix(*object.Key, "saved_model.pb") {
			continue
		}

		keyParts := strings.Split(*object.Key, "/")
		versionStr := keyParts[len(keyParts)-1]
		version, err := strconv.ParseInt(versionStr, 10, 64)
		if err != nil {
			return []int64{}, err
		}

		versionedPath := "s3://" + filepath.Join(bucket, filepath.Join(keyParts[:len(keyParts)-1]...))
		if (isNeuronExport && isValidNeuronTensorFlowS3Directory(versionedPath, awsClientForBucket)) ||
			(!isNeuronExport && isValidTensorFlowS3Directory(versionedPath, awsClientForBucket)) {
			versions = append(versions, version)
		}
	}

	return versions, nil
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

// GetTFServingVersionsFromLocalPath checks that the path contains a valid local directory for TensorFlow models:
// - model-name
// 		- 1523423423/ (version prefix, usually a timestamp)
//			- saved_model.pb
// 			- variables/
//				- variables.index
//				- variables.data-00000-of-00001 (there are a variable number of these files)
// 		- 2434389194/ (version prefix, usually a timestamp)
// 			- saved_model.pb
//			- variables/
//				- variables.index
//				- variables.data-00000-of-00001 (there are a variable number of these files)
//   ...
func GetTFServingVersionsFromLocalPath(path string) ([]int64, error) {
	paths, err := files.ListDirRecursive(path, false, files.IgnoreHiddenFiles, files.IgnoreHiddenFolders)
	if err != nil {
		return []int64{}, err
	} else if len(paths) == 0 {
		return []int64{}, ErrorDirIsEmpty(path)
	}

	versions := []int64{}
	for _, path := range paths {
		if strings.HasSuffix(path, "saved_model.pb") {
			possiblePath := filepath.Dir(path)

			versionStr := filepath.Base(possiblePath)
			version, err := strconv.ParseInt(versionStr, 10, 64)
			if err != nil {
				return []int64{}, err
			}

			validTFDirectory, err := IsValidTensorFlowLocalDirectory(possiblePath)
			if err != nil {
				return []int64{}, err
			}
			if validTFDirectory {
				versions = append(versions, version)
			}
		}
	}

	return versions, nil
}

// IsValidTensorFlowLocalDirectory checks that the path contains a valid local directory for TensorFlow models
// Must contain the following structure:
// - 1523423423/ (version prefix, usually a timestamp)
// 		- saved_model.pb
//		- variables/
//			- variables.index
//			- variables.data-00000-of-00001 (there are a variable number of these files)
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

// getONNXVersionsFromS3Path checks that the path contains a valid S3 directory for versioned ONNX models:
// - model-name
// 		- 1523423423/ (version prefix, usually a timestamp)
// 			- *.onnx
// 		- 2434389194/ (version prefix, usually a timestamp)
//			- *.onnx
// 		...
func getONNXVersionsFromS3Path(path string, awsClientForBucket *aws.Client) ([]int64, error) {
	objects, err := awsClientForBucket.ListS3PathDir(path, false, pointer.Int64(1000))
	if err != nil {
		return []int64{}, err
	} else if len(objects) == 0 {
		return []int64{}, errors.Wrap(ErrorInvalidONNXModelPath(), path)
	}

	versions := []int64{}
	for _, object := range objects {
		if !strings.HasSuffix(*object.Key, ".onnx") {
			continue
		}

		keyParts := strings.Split(*object.Key, "/")
		versionStr := keyParts[len(keyParts)-1]
		version, err := strconv.ParseInt(versionStr, 10, 64)
		if err != nil {
			return []int64{}, err
		}

		versions = append(versions, version)
	}

	return versions, nil
}

// GetONNXVersionsFromLocalPath checks that the path contains a valid local directory for versioned ONNX models:
// - model-name
// 		- 1523423423/ (version prefix, usually a timestamp)
// 			- *.onnx
// 		- 2434389194/ (version prefix, usually a timestamp)
//			- *.onnx
// 		...
func GetONNXVersionsFromLocalPath(path string) ([]int64, error) {
	paths, err := files.ListDirRecursive(path, false, files.IgnoreHiddenFiles, files.IgnoreHiddenFolders)
	if err != nil {
		return []int64{}, err
	} else if len(paths) == 0 {
		return []int64{}, ErrorDirIsEmpty(path)
	}

	versions := []int64{}
	for _, path := range paths {
		if strings.HasSuffix(path, ".onnx") {
			possiblePath := filepath.Dir(path)

			versionStr := filepath.Base(possiblePath)
			version, err := strconv.ParseInt(versionStr, 10, 64)
			if err != nil {
				return []int64{}, err
			}

			versions = append(versions, version)
		}
	}

	return versions, nil
}

// getPythonVersionsFromS3Path checks that the path contains a valid S3 directory for versioned Python models:
// - model-name
// 		- 1523423423/ (version prefix, usually a timestamp)
// 			- *
// 		- 2434389194/ (version prefix, usually a timestamp)
//			- *
// 		...
func getPythonVersionsFromS3Path(path string, awsClientForBucket *aws.Client) ([]int64, error) {
	objects, err := awsClientForBucket.GetNLevelsDeepFromS3Path(path, 1, false, pointer.Int64(1000))
	if err != nil {
		return []int64{}, err
	} else if len(objects) == 0 {
		return []int64{}, ErrorNoVersionsFoundForPythonModelPath(path)
	}

	versions := []int64{}
	for _, object := range objects {
		keyParts := strings.Split(object, "/")
		versionStr := keyParts[len(keyParts)-1]
		version, err := strconv.ParseInt(versionStr, 10, 64)
		if err != nil {
			return []int64{}, ErrorInvalidPythonModelPath(path)
		}

		modelVersionPath := aws.JoinS3Path(path, versionStr)
		if yes, err := awsClientForBucket.IsS3PathDir(modelVersionPath); err != nil {
			return []int64{}, err
		} else if !yes {
			return []int64{}, ErrorPythonModelVersionPathMustBeDir(path, aws.JoinS3Path(path, versionStr))
		}

		versions = append(versions, version)
	}

	return slices.UniqueInt64(versions), nil
}

// GetPythonVersionsFromLocalPath checks that the path contains a valid local directory for versioned Python models:
// - model-name
// 		- 1523423423/ (version prefix, usually a timestamp)
// 			- *
// 		- 2434389194/ (version prefix, usually a timestamp)
//			- *
// 		...
func GetPythonVersionsFromLocalPath(path string) ([]int64, error) {
	if !files.IsDir(path) {
		return []int64{}, ErrorInvalidDirPath(path)
	}
	dirPaths, err := files.ListDirRecursive(path, false, files.IgnoreHiddenFiles, files.IgnoreHiddenFolders)
	if err != nil {
		return []int64{}, err
	} else if len(dirPaths) == 0 {
		return []int64{}, ErrorNoVersionsFoundForPythonModelPath(path)
	}

	basePathLength := len(strings.Split(path, "/"))
	versions := []int64{}
	for _, dirPath := range dirPaths {
		pathParts := strings.Split(dirPath, "/")
		versionStr := pathParts[basePathLength]
		version, err := strconv.ParseInt(versionStr, 10, 64)
		if err != nil {
			return []int64{}, ErrorInvalidPythonModelPath(path)
		}

		modelVersionPath := filepath.Join(path, versionStr)
		if !files.IsDir(modelVersionPath) {
			return []int64{}, ErrorPythonModelVersionPathMustBeDir(path, modelVersionPath)
		}

		if objects, err := files.ListDir(modelVersionPath, false); err != nil {
			return []int64{}, err
		} else if len(objects) == 0 {
			continue
		}

		versions = append(versions, version)
	}

	return slices.UniqueInt64(versions), nil
}
