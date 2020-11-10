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

func checkDuplicateModelNames(models []CuratedModelResource) error {
	names := strset.New()

	for _, model := range models {
		if names.Has(model.Name) {
			return ErrorDuplicateModelNames(model.Name)
		}
		names.Add(model.Name)
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

func modelResourceToCurated(modelResources []userconfig.ModelResource, projectDir string) ([]CuratedModelResource, error) {
	models := []CuratedModelResource{}
	for _, model := range modelResources {
		isS3Path := strings.HasPrefix(model.ModelPath, "s3://")
		if !isS3Path {
			model.ModelPath = files.RelToAbsPath(model.ModelPath, projectDir)
		}

		model.ModelPath = s.EnsureSuffix(model.ModelPath, "/")

		models = append(models, CuratedModelResource{
			ModelResource: &userconfig.ModelResource{
				Name:         model.Name,
				ModelPath:    model.ModelPath,
				SignatureKey: model.SignatureKey,
			},
			S3Path: isS3Path,
		})
	}

	return models, nil
}

// List the model objects found in the S3/local path directory.
//
// The model name is determined from the objects' names found in the path directory.
// Path can either be an S3 path or a local system path - in the latter case, the returned paths will be in absolute form.
func listModelResourcesFromPath(path string, projectFiles ProjectFiles, awsClient *aws.Client) ([]userconfig.ModelResource, error) {
	models := []userconfig.ModelResource{}

	if aws.IsValidS3Path(path) {
		awsClientForBucket, err := aws.NewFromClientS3Path(path, awsClient)
		if err != nil {
			return nil, err
		}

		isDir, err := awsClientForBucket.IsS3PathDir(path)
		if err != nil {
			return nil, err
		}

		if !isDir {
			return nil, ErrorS3DirNotFound(path)
		}

		modelPaths, _, err := awsClientForBucket.GetNLevelsDeepFromS3Path(path, 1, false, pointer.Int64(20000))
		if err != nil {
			return nil, err
		}
		var bucket string
		bucket, _, err = aws.SplitS3Path(path)
		if err != nil {
			return nil, err
		}

		for _, modelPath := range modelPaths {
			models = append(models, userconfig.ModelResource{
				Name:      filepath.Base(modelPath),
				ModelPath: aws.S3Path(bucket, modelPath),
			})
		}

	} else {
		path = files.RelToAbsPath(path, projectFiles.ProjectDir())

		if err := files.CheckDir(path); err != nil {
			return nil, err
		}

		modelObjects, err := files.ListDir(path, true)
		if err != nil {
			return nil, err
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

// getPythonVersionsFromS3Path checks that the path contains a valid S3 directory for versioned Python models:
// - model-name
// 		- 1523423423/ (version prefix, usually a timestamp)
// 			- *
// 		- 2434389194/ (version prefix, usually a timestamp)
//			- *
// 		...
func getPythonVersionsFromS3Path(modelPath string, awsClientForBucket *aws.Client) ([]int64, error) {
	isDir, err := awsClientForBucket.IsS3PathDir(modelPath)
	if err != nil {
		return nil, err
	}
	if !isDir {
		return nil, ErrorModelPathNotDirectory(modelPath)
	}

	modelSubPaths, allModelSubPaths, err := awsClientForBucket.GetNLevelsDeepFromS3Path(modelPath, 1, false, pointer.Int64(1000))
	if err != nil {
		return nil, err
	}

	if len(modelSubPaths) == 0 {
		return nil, ErrorS3DirIsEmpty(modelPath)
	}

	versions := []int64{}
	for _, modelSubPath := range modelSubPaths {
		keyParts := strings.Split(modelSubPath, "/")
		versionStr := keyParts[len(keyParts)-1]
		version, err := strconv.ParseInt(versionStr, 10, 64)
		if err != nil {
			return nil, ErrorInvalidPythonModelPath(modelPath, allModelSubPaths)
		}

		modelVersionPath := aws.JoinS3Path(modelPath, versionStr)
		if err := validatePythonS3ModelDir(modelVersionPath, awsClientForBucket); err != nil {
			if errors.GetKind(err) == ErrDirIsEmpty {
				continue
			}
			return nil, errors.Append(err, "\n\n"+ErrorInvalidPythonModelPath(modelPath, allModelSubPaths).Error())
		}
		versions = append(versions, version)
	}

	return slices.UniqueInt64(versions), nil
}

func validatePythonS3ModelDir(modelPath string, awsClientForBucket *aws.Client) error {
	isDir, err := awsClientForBucket.IsS3PathDir(modelPath)
	if err != nil {
		return err
	}
	if !isDir {
		return ErrorModelPathNotDirectory(modelPath)
	}

	modelSubPaths, _, err := awsClientForBucket.GetNLevelsDeepFromS3Path(modelPath, 1, false, pointer.Int64(1000))
	if err != nil {
		return err
	}

	if len(modelSubPaths) == 0 {
		return ErrorS3DirIsEmpty(modelPath)
	}

	return nil
}

// getPythonVersionsFromLocalPath checks that the path contains a valid local directory for versioned Python models:
// - model-name
// 		- 1523423423/ (version prefix, usually a timestamp)
// 			- *
// 		- 2434389194/ (version prefix, usually a timestamp)
//			- *
// 		...
func getPythonVersionsFromLocalPath(modelPath string) ([]int64, error) {
	if !files.IsDir(modelPath) {
		return nil, ErrorModelPathNotDirectory(modelPath)
	}

	modelSubPaths, err := files.ListDirRecursive(modelPath, false)
	if err != nil {
		return nil, err
	} else if len(modelSubPaths) == 0 {
		return nil, ErrorDirIsEmpty(modelPath)
	}

	basePathLength := len(slices.RemoveEmpties(strings.Split(modelPath, "/")))
	versions := []int64{}
	for _, modelSubPath := range modelSubPaths {
		pathParts := slices.RemoveEmpties(strings.Split(modelSubPath, "/"))
		versionStr := pathParts[basePathLength]
		version, err := strconv.ParseInt(versionStr, 10, 64)
		if err != nil {
			return nil, ErrorInvalidPythonModelPath(modelPath, modelSubPaths)
		}

		modelVersionPath := filepath.Join(modelPath, versionStr)
		if err := validatePythonLocalModelDir(modelVersionPath); err != nil {
			if errors.GetKind(err) == ErrDirIsEmpty {
				continue
			}
			return nil, errors.Append(err, "\n\n"+ErrorInvalidPythonModelPath(modelPath, modelSubPaths).Error())
		}
		versions = append(versions, version)
	}

	return slices.UniqueInt64(versions), nil
}

func validatePythonLocalModelDir(modelPath string) error {
	if !files.IsDir(modelPath) {
		return ErrorModelPathNotDirectory(modelPath)
	}

	if objects, err := files.ListDir(modelPath, false); err != nil {
		return err
	} else if len(objects) == 0 {
		return ErrorDirIsEmpty(modelPath)
	}

	return nil
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
func getTFServingVersionsFromS3Path(modelPath string, isNeuronExport bool, awsClientForBucket *aws.Client) ([]int64, error) {
	isDir, err := awsClientForBucket.IsS3PathDir(modelPath)
	if err != nil {
		return nil, err
	}
	if !isDir {
		return nil, ErrorModelPathNotDirectory(modelPath)
	}

	modelSubPaths, allModelSubPaths, err := awsClientForBucket.GetNLevelsDeepFromS3Path(modelPath, 1, false, pointer.Int64(1000))
	if err != nil {
		return nil, err
	}

	if len(modelSubPaths) == 0 {
		return nil, ErrorS3DirIsEmpty(modelPath)
	}

	versions := []int64{}
	for _, modelSubPath := range modelSubPaths {
		keyParts := strings.Split(modelSubPath, "/")
		versionStr := keyParts[len(keyParts)-1]
		version, err := strconv.ParseInt(versionStr, 10, 64)
		if err != nil {
			return nil, ErrorInvalidTensorFlowModelPath(modelPath, isNeuronExport, allModelSubPaths)
		}

		modelVersionPath := aws.JoinS3Path(modelPath, versionStr)
		if err := validateTFServingS3ModelDir(modelVersionPath, isNeuronExport, awsClientForBucket); err != nil {
			if errors.GetKind(err) == ErrDirIsEmpty {
				continue
			}
			if errors.GetKind(err) != ErrInvalidTensorFlowModelPath {
				return nil, errors.Append(err, "\n\n"+ErrorInvalidTensorFlowModelPath(modelPath, isNeuronExport, allModelSubPaths).Error())
			}
			return nil, ErrorInvalidTensorFlowModelPath(modelPath, isNeuronExport, allModelSubPaths)
		}
		versions = append(versions, version)
	}

	return slices.UniqueInt64(versions), nil
}

func validateTFServingS3ModelDir(modelPath string, isNeuronExport bool, awsClientForBucket *aws.Client) error {
	isDir, err := awsClientForBucket.IsS3PathDir(modelPath)
	if err != nil {
		return err
	}
	if !isDir {
		return ErrorModelPathNotDirectory(modelPath)
	}

	modelSubPaths, allModelSubPaths, err := awsClientForBucket.GetNLevelsDeepFromS3Path(modelPath, 1, false, pointer.Int64(1000))
	if err != nil {
		return err
	}

	if len(modelSubPaths) == 0 {
		return ErrorS3DirIsEmpty(modelPath)
	}

	if isNeuronExport {
		if !isValidNeuronTensorFlowS3Directory(modelPath, awsClientForBucket) {
			return ErrorInvalidTensorFlowModelPath(modelPath, isNeuronExport, allModelSubPaths)
		}
	} else {
		if !isValidTensorFlowS3Directory(modelPath, awsClientForBucket) {
			return ErrorInvalidTensorFlowModelPath(modelPath, isNeuronExport, allModelSubPaths)
		}
	}

	return nil
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

// getTFServingVersionsFromLocalPath checks that the path contains a valid local directory for TensorFlow models:
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
func getTFServingVersionsFromLocalPath(modelPath string) ([]int64, error) {
	if !files.IsDir(modelPath) {
		return nil, ErrorModelPathNotDirectory(modelPath)
	}

	modelSubPaths, err := files.ListDirRecursive(modelPath, false)
	if err != nil {
		return nil, err
	} else if len(modelSubPaths) == 0 {
		return nil, ErrorDirIsEmpty(modelPath)
	}

	basePathLength := len(slices.RemoveEmpties(strings.Split(modelPath, "/")))
	versions := []int64{}

	for _, modelSubPath := range modelSubPaths {
		pathParts := slices.RemoveEmpties(strings.Split(modelSubPath, "/"))
		versionStr := pathParts[basePathLength]
		version, err := strconv.ParseInt(versionStr, 10, 64)
		if err != nil {
			return nil, ErrorInvalidTensorFlowModelPath(modelPath, false, modelSubPaths)
		}

		modelVersionPath := filepath.Join(modelPath, versionStr)
		if err := validateTFServingLocalModelDir(modelVersionPath); err != nil {
			if errors.GetKind(err) == ErrDirIsEmpty {
				continue
			}
			return nil, errors.Append(err, "\n\n"+ErrorInvalidTensorFlowModelPath(modelPath, false, modelSubPaths).Error())
		}

		versions = append(versions, version)
	}

	return slices.UniqueInt64(versions), nil
}

func validateTFServingLocalModelDir(modelPath string) error {
	if !files.IsDir(modelPath) {
		return ErrorModelPathNotDirectory(modelPath)
	}

	var versionObjects []string
	var err error
	if versionObjects, err = files.ListDir(modelPath, false); err != nil {
		return err
	} else if len(versionObjects) == 0 {
		return ErrorDirIsEmpty(modelPath)
	}

	if yes, err := isValidTensorFlowLocalDirectory(modelPath); !yes || err != nil {
		return ErrorInvalidTensorFlowModelPath(modelPath, false, versionObjects)
	}

	return nil
}

// isValidTensorFlowLocalDirectory checks that the path contains a valid local directory for TensorFlow models
// Must contain the following structure:
// - 1523423423/ (version prefix, usually a timestamp)
// 		- saved_model.pb
//		- variables/
//			- variables.index
//			- variables.data-00000-of-00001 (there are a variable number of these files)
func isValidTensorFlowLocalDirectory(path string) (bool, error) {
	paths, err := files.ListDirRecursive(path, true)
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
// 			- <model-name>.onnx
// 		- 2434389194/ (version prefix, usually a timestamp)
//			- <model-name>.onnx
// 		...
func getONNXVersionsFromS3Path(modelPath string, awsClientForBucket *aws.Client) ([]int64, error) {
	isDir, err := awsClientForBucket.IsS3PathDir(modelPath)
	if err != nil {
		return nil, err
	}
	if !isDir {
		return nil, ErrorModelPathNotDirectory(modelPath)
	}

	modelSubPaths, allModelSubPaths, err := awsClientForBucket.GetNLevelsDeepFromS3Path(modelPath, 1, false, pointer.Int64(1000))
	if err != nil {
		return nil, err
	}

	if len(modelSubPaths) == 0 {
		return nil, ErrorS3DirIsEmpty(modelPath)
	}

	versions := []int64{}
	for _, modelSubPath := range modelSubPaths {
		keyParts := strings.Split(modelSubPath, "/")
		versionStr := keyParts[len(keyParts)-1]
		version, err := strconv.ParseInt(versionStr, 10, 64)
		if err != nil {
			return nil, ErrorInvalidONNXModelPath(modelPath, allModelSubPaths)
		}

		modelVersionPath := aws.JoinS3Path(modelPath, versionStr)
		if err := validateONNXS3ModelDir(modelVersionPath, awsClientForBucket); err != nil {
			if errors.GetKind(err) == ErrDirIsEmpty {
				continue
			}
			if errors.GetKind(err) != ErrInvalidONNXModelPath {
				return nil, errors.Append(err, "\n\n"+ErrorInvalidONNXModelPath(modelPath, allModelSubPaths).Error())
			}
			return nil, ErrorInvalidONNXModelPath(modelPath, allModelSubPaths)
		}

		versions = append(versions, version)
	}

	return slices.UniqueInt64(versions), nil
}

func validateONNXS3ModelDir(modelPath string, awsClientForBucket *aws.Client) error {
	isDir, err := awsClientForBucket.IsS3PathDir(modelPath)
	if err != nil {
		return err
	}
	if !isDir {
		return ErrorModelPathNotDirectory(modelPath)
	}

	bucket, _, err := aws.SplitS3Path(modelPath)
	if err != nil {
		return err
	}

	modelSubPaths, allModelSubPaths, err := awsClientForBucket.GetNLevelsDeepFromS3Path(modelPath, 1, false, pointer.Int64(1000))
	if err != nil {
		return err
	}

	if len(modelSubPaths) == 0 {
		return ErrorS3DirIsEmpty(modelPath)
	}

	numONNXFiles := 0
	for _, modelSubPath := range modelSubPaths {
		if !strings.HasSuffix(modelSubPath, ".onnx") {
			return ErrorInvalidONNXModelPath(modelPath, allModelSubPaths)
		}
		if isFile, err := awsClientForBucket.IsS3PathFile(aws.S3Path(bucket, modelSubPath)); err != nil {
			return errors.Wrap(err, modelPath)
		} else if !isFile {
			return ErrorInvalidONNXModelPath(modelPath, allModelSubPaths)
		}
		numONNXFiles++
	}

	if numONNXFiles > 1 {
		return ErrorInvalidONNXModelPath(modelPath, allModelSubPaths)
	}

	return nil
}

// getONNXVersionsFromLocalPath checks that the path contains a valid local directory for versioned ONNX models:
// - model-name
// 		- 1523423423/ (version prefix, usually a timestamp)
// 			- <model-name>.onnx
// 		- 2434389194/ (version prefix, usually a timestamp)
//			- <model-name>.onnx
// 		...
func getONNXVersionsFromLocalPath(modelPath string) ([]int64, error) {
	if !files.IsDir(modelPath) {
		return nil, ErrorModelPathNotDirectory(modelPath)
	}

	modelSubPaths, err := files.ListDirRecursive(modelPath, false)
	if err != nil {
		return nil, err
	} else if len(modelSubPaths) == 0 {
		return nil, ErrorDirIsEmpty(modelPath)
	}

	basePathLength := len(slices.RemoveEmpties(strings.Split(modelPath, "/")))
	versions := []int64{}

	for _, modelSubPath := range modelSubPaths {
		pathParts := slices.RemoveEmpties(strings.Split(modelSubPath, "/"))
		versionStr := pathParts[basePathLength]
		version, err := strconv.ParseInt(versionStr, 10, 64)
		if err != nil {
			return nil, ErrorInvalidONNXModelPath(modelPath, modelSubPaths)
		}

		modelVersionPath := filepath.Join(modelPath, versionStr)
		if err := validateONNXLocalModelDir(modelVersionPath); err != nil {
			if errors.GetKind(err) == ErrDirIsEmpty {
				continue
			}
			return nil, errors.Append(err, "\n\n"+ErrorInvalidONNXModelPath(modelPath, modelSubPaths).Error())
		}

		versions = append(versions, version)
	}

	return slices.UniqueInt64(versions), nil
}

func validateONNXLocalModelDir(modelPath string) error {
	if !files.IsDir(modelPath) {
		return ErrorModelPathNotDirectory(modelPath)
	}

	var versionObjects []string
	var err error
	if versionObjects, err = files.ListDir(modelPath, false); err != nil {
		return err
	} else if len(versionObjects) == 0 {
		return ErrorDirIsEmpty(modelPath)
	}

	numONNXFiles := 0
	for _, versionObject := range versionObjects {
		if !strings.HasSuffix(versionObject, ".onnx") || !files.IsFile(versionObject) {
			return ErrorInvalidONNXModelPath(modelPath, versionObjects)
		}
		numONNXFiles++
	}

	if numONNXFiles > 1 {
		return ErrorInvalidONNXModelPath(modelPath, versionObjects)
	}

	return nil
}

func verifyTotalWeight(apis []*userconfig.TrafficSplit) error {
	totalWeight := int32(0)
	for _, api := range apis {
		totalWeight += api.Weight
	}
	if totalWeight == 100 {
		return nil
	}
	return errors.Wrap(ErrorIncorrectTrafficSplitterWeightTotal(totalWeight), userconfig.APIsKey)
}

// areTrafficSplitterAPIsUnique gives error if the same API is used multiple times in TrafficSplitter
func areTrafficSplitterAPIsUnique(apis []*userconfig.TrafficSplit) error {
	names := make(map[string][]userconfig.TrafficSplit)
	for _, api := range apis {
		names[api.Name] = append(names[api.Name], *api)
	}
	var notUniqueAPIs []string
	for name := range names {
		if len(names[name]) > 1 {
			notUniqueAPIs = append(notUniqueAPIs, names[name][0].Name)
		}
	}
	if len(notUniqueAPIs) > 0 {
		return errors.Wrap(ErrorTrafficSplitterAPIsNotUnique(notUniqueAPIs), userconfig.APIsKey)
	}
	return nil
}
