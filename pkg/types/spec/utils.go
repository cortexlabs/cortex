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
	"github.com/cortexlabs/cortex/pkg/lib/gcp"
	"github.com/cortexlabs/cortex/pkg/lib/pointer"
	"github.com/cortexlabs/cortex/pkg/lib/sets/strset"
	"github.com/cortexlabs/cortex/pkg/lib/slices"
	s "github.com/cortexlabs/cortex/pkg/lib/strings"
	"github.com/cortexlabs/cortex/pkg/types/userconfig"
)

type modelValidator func(paths []string, prefix string, versionedPrefix *string) error

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

func getErrorForPredictorType(api userconfig.API, modelPrefix string, modelPaths []string) error {
	switch api.Predictor.Type {
	case userconfig.PythonPredictorType:
		return ErrorInvalidPythonModelPath(modelPrefix, modelPaths)
	case userconfig.ONNXPredictorType:
		return ErrorInvalidONNXModelPath(modelPrefix, modelPaths)
	case userconfig.TensorFlowPredictorType:
		return ErrorInvalidTensorFlowModelPath(modelPrefix, api.Compute.Inf > 0, modelPaths)
	}
	return nil
}

func validateDirModels(modelPath string, api userconfig.API, projectDir string, awsClient *aws.Client, gcpClient *gcp.Client, extraValidators []modelValidator) ([]CuratedModelResource, error) {
	var bucket string
	var dirPrefix string
	var modelDirPaths []string
	var err error

	modelPath = s.EnsureSuffix(modelPath, "/")

	s3Path := strings.HasPrefix(modelPath, "s3://")
	gcsPath := strings.HasPrefix(modelPath, "gs://")
	localPath := !s3Path && !gcsPath

	if s3Path {
		awsClientForBucket, err := aws.NewFromClientS3Path(modelPath, awsClient)
		if err != nil {
			return nil, err
		}

		bucket, dirPrefix, err = aws.SplitS3Path(modelPath)
		if err != nil {
			return nil, err
		}

		s3Objects, err := awsClientForBucket.ListS3PathDir(modelPath, false, nil)
		if err != nil {
			return nil, err
		}
		modelDirPaths = aws.ConvertS3ObjectsToKeys(s3Objects...)
	}
	if gcsPath {
		bucket, dirPrefix, err = gcp.SplitGCSPath(modelPath)
		if err != nil {
			return nil, err
		}

		modelDirPaths, err = gcpClient.ListGCSPathDir(modelPath, nil)
		if err != nil {
			return nil, err
		}
	}
	if localPath {
		expandedLocalPath := files.RelToAbsPath(modelPath, projectDir)
		dirPrefix = s.EnsureSuffix(expandedLocalPath, "/")

		err := files.CheckDir(dirPrefix)
		if err != nil {
			return nil, err
		}

		modelDirPaths, err = files.ListDirRecursive(dirPrefix, false, nil...)
		if err != nil {
			return nil, err
		}
	}
	if len(modelDirPaths) == 0 {
		return nil, getErrorForPredictorType(api, dirPrefix, modelDirPaths)
	}

	modelNames := []string{}
	modelDirPathLength := len(slices.RemoveEmpties(strings.Split(dirPrefix, "/")))
	for _, path := range modelDirPaths {
		splitPath := strings.Split(path, "/")
		modelNames = append(modelNames, splitPath[modelDirPathLength])
	}
	modelNames = slices.UniqueStrings(modelNames)

	modelResources := make([]CuratedModelResource, len(modelNames))
	for i, modelName := range modelNames {
		modelPrefix := filepath.Join(dirPrefix, modelName)
		modelPrefix = s.EnsureSuffix(modelPrefix, "/")

		modelStructureType := determineBaseModelStructure(modelDirPaths, modelPrefix)
		if modelStructureType == userconfig.UnknownModelStructureType {
			return nil, errors.Wrap(getErrorForPredictorType(api, modelPrefix, nil), modelName)
		}

		var versions []string
		if modelStructureType == userconfig.VersionedModelType {
			versions = getModelVersionsFromPaths(modelDirPaths, modelPrefix)
			for _, version := range versions {
				versionedModelPrefix := filepath.Join(modelPrefix, version)
				versionedModelPrefix = s.EnsureSuffix(versionedModelPrefix, "/")

				for _, validator := range extraValidators {
					err := validator(modelDirPaths, modelPrefix, pointer.String(versionedModelPrefix))
					if err != nil {
						return nil, errors.Wrap(err, modelName)
					}
				}
			}
		} else {
			for _, validator := range extraValidators {
				err := validator(modelDirPaths, modelPrefix, nil)
				if err != nil {
					return nil, errors.Wrap(err, modelName)
				}
			}
		}

		intVersions, err := slices.StringToInt64(versions)
		if err != nil {
			return nil, errors.Wrap(err, modelName)
		}

		fullModelPath := ""
		if s3Path {
			fullModelPath = s.EnsureSuffix(aws.S3Path(bucket, modelPrefix), "/")
		}
		if gcsPath {
			fullModelPath = s.EnsureSuffix(gcp.GCSPath(bucket, modelPrefix), "/")
		}
		if localPath {
			fullModelPath = s.EnsureSuffix(modelPrefix, "/")
		}

		modelResources[i] = CuratedModelResource{
			ModelResource: &userconfig.ModelResource{
				Name:         modelName,
				ModelPath:    fullModelPath,
				SignatureKey: api.Predictor.SignatureKey,
			},
			S3Path:    s3Path,
			GCSPath:   gcsPath,
			LocalPath: localPath,
			Versions:  intVersions,
		}
	}

	return modelResources, nil
}

func validateModels(models []userconfig.ModelResource, api userconfig.API, projectDir string, awsClient *aws.Client, gcpClient *gcp.Client, extraValidators []modelValidator) ([]CuratedModelResource, error) {
	var bucket string
	var modelPrefix string
	var modelPaths []string
	var err error

	modelResources := make([]CuratedModelResource, len(models))
	for i, model := range models {
		modelPath := s.EnsureSuffix(model.ModelPath, "/")

		s3Path := strings.HasPrefix(model.ModelPath, "s3://")
		gcsPath := strings.HasPrefix(model.ModelPath, "gs://")
		localPath := !s3Path && !gcsPath

		if s3Path {
			awsClientForBucket, err := aws.NewFromClientS3Path(model.ModelPath, awsClient)
			if err != nil {
				return nil, errors.Wrap(err, model.Name)
			}

			bucket, modelPrefix, err = aws.SplitS3Path(model.ModelPath)
			if err != nil {
				return nil, errors.Wrap(err, model.Name)
			}
			modelPrefix = s.EnsureSuffix(modelPrefix, "/")

			s3Objects, err := awsClientForBucket.ListS3PathDir(modelPath, false, nil)
			if err != nil {
				return nil, errors.Wrap(err, model.Name)
			}
			modelPaths = aws.ConvertS3ObjectsToKeys(s3Objects...)
		}

		if gcsPath {
			bucket, modelPrefix, err = gcp.SplitGCSPath(model.ModelPath)
			if err != nil {
				return nil, errors.Wrap(err, model.Name)
			}
			modelPrefix = s.EnsureSuffix(modelPrefix, "/")

			modelPaths, err = gcpClient.ListGCSPathDir(modelPath, nil)
			if err != nil {
				return nil, errors.Wrap(err, model.Name)
			}
		}

		if localPath {
			expandedLocalPath := files.RelToAbsPath(model.ModelPath, projectDir)
			modelPrefix = s.EnsureSuffix(expandedLocalPath, "/")

			err := files.CheckDir(modelPrefix)
			if err != nil {
				return nil, errors.Wrap(err, model.Name)
			}

			modelPaths, err = files.ListDirRecursive(modelPrefix, false, nil...)
			if err != nil {
				return nil, errors.Wrap(err, model.Name)
			}
		}
		if len(modelPaths) == 0 {
			return nil, errors.Wrap(getErrorForPredictorType(api, modelPrefix, modelPaths), model.Name)
		}

		modelStructureType := determineBaseModelStructure(modelPaths, modelPrefix)
		if modelStructureType == userconfig.UnknownModelStructureType {
			return nil, errors.Wrap(ErrorInvalidPythonModelPath(modelPath, []string{}), model.Name)
		}

		var versions []string
		if modelStructureType == userconfig.VersionedModelType {
			versions = getModelVersionsFromPaths(modelPaths, modelPrefix)
			for _, version := range versions {
				versionedModelPrefix := filepath.Join(modelPrefix, version)
				versionedModelPrefix = s.EnsureSuffix(versionedModelPrefix, "/")

				for _, validator := range extraValidators {
					err := validator(modelPaths, modelPrefix, pointer.String(versionedModelPrefix))
					if err != nil {
						return nil, errors.Wrap(err, model.Name)
					}
				}
			}
		} else {
			for _, validator := range extraValidators {
				err := validator(modelPaths, modelPrefix, nil)
				if err != nil {
					return nil, errors.Wrap(err, model.Name)
				}
			}
		}

		intVersions, err := slices.StringToInt64(versions)
		if err != nil {
			return nil, errors.Wrap(err, model.Name)
		}

		var signatureKey *string
		if model.SignatureKey != nil {
			signatureKey = model.SignatureKey
		} else if api.Predictor.Models != nil && api.Predictor.Models.SignatureKey != nil {
			signatureKey = api.Predictor.SignatureKey
		}

		fullModelPath := ""
		if s3Path {
			fullModelPath = s.EnsureSuffix(aws.S3Path(bucket, modelPrefix), "/")
		}
		if gcsPath {
			fullModelPath = s.EnsureSuffix(gcp.GCSPath(bucket, modelPrefix), "/")
		}
		if localPath {
			fullModelPath = s.EnsureSuffix(modelPrefix, "/")
		}

		modelResources[i] = CuratedModelResource{
			ModelResource: &userconfig.ModelResource{
				Name:         model.Name,
				ModelPath:    fullModelPath,
				SignatureKey: signatureKey,
			},
			S3Path:    s3Path,
			GCSPath:   gcsPath,
			LocalPath: localPath,
			Versions:  intVersions,
		}
	}

	return modelResources, nil
}

func onnxModelValidator(paths []string, prefix string, versionedPrefix *string) error {
	var filteredFilePaths []string
	if versionedPrefix != nil {
		filteredFilePaths = files.FilterPathsWithDirPrefix(paths, *versionedPrefix)
	} else {
		filteredFilePaths = files.FilterPathsWithDirPrefix(paths, prefix)
	}

	errFunc := func() error {
		if versionedPrefix != nil {
			return ErrorInvalidONNXModelPath(prefix, files.FilterPathsWithDirPrefix(paths, prefix))
		}
		return ErrorInvalidONNXModelPath(prefix, filteredFilePaths)
	}

	if len(filteredFilePaths) != 1 {
		return errFunc()
	}

	if !strings.HasSuffix(filteredFilePaths[0], ".onnx") {
		return errFunc()
	}

	return nil
}

func tensorflowModelValidator(paths []string, prefix string, versionedPrefix *string) error {
	var filteredFilePaths []string
	if versionedPrefix != nil {
		filteredFilePaths = files.FilterPathsWithDirPrefix(paths, *versionedPrefix)
	} else {
		filteredFilePaths = files.FilterPathsWithDirPrefix(paths, prefix)
	}

	errFunc := func() error {
		if versionedPrefix != nil {
			return ErrorInvalidTensorFlowModelPath(prefix, false, files.FilterPathsWithDirPrefix(paths, prefix))
		}
		return ErrorInvalidTensorFlowModelPath(prefix, false, filteredFilePaths)
	}

	filesToHave := []string{"saved_model.pb", "variables/variables.index"}
	for _, fileToHave := range filesToHave {
		var fullFilePath string
		if versionedPrefix != nil {
			fullFilePath = filepath.Join(*versionedPrefix, fileToHave)
		} else {
			fullFilePath = filepath.Join(prefix, fileToHave)
		}
		if !slices.HasString(filteredFilePaths, fullFilePath) {
			return errFunc()
		}
	}

	filesWithPrefix := []string{"variables/variables.data-00000-of"}
	for _, fileWithPrefix := range filesWithPrefix {
		var prefixPath string
		if versionedPrefix != nil {
			prefixPath = filepath.Join(*versionedPrefix, fileWithPrefix)
		} else {
			prefixPath = filepath.Join(prefix, fileWithPrefix)
		}
		prefixPaths := slices.FilterStrs(filteredFilePaths, func(path string) bool {
			return strings.HasPrefix(path, prefixPath)
		})
		if len(prefixPaths) == 0 {
			return errFunc()
		}
	}

	return nil
}

func tensorflowNeuronModelValidator(paths []string, prefix string, versionedPrefix *string) error {
	var filteredFilePaths []string
	if versionedPrefix != nil {
		filteredFilePaths = files.FilterPathsWithDirPrefix(paths, *versionedPrefix)
	} else {
		filteredFilePaths = files.FilterPathsWithDirPrefix(paths, prefix)
	}

	errFunc := func() error {
		if versionedPrefix != nil {
			return ErrorInvalidTensorFlowModelPath(prefix, true, files.FilterPathsWithDirPrefix(paths, prefix))
		}
		return ErrorInvalidTensorFlowModelPath(prefix, true, filteredFilePaths)
	}

	filesToHave := []string{"saved_model.pb"}
	for _, fileToHave := range filesToHave {
		var fullFilePath string
		if versionedPrefix != nil {
			fullFilePath = filepath.Join(*versionedPrefix, fileToHave)
		} else {
			fullFilePath = filepath.Join(prefix, fileToHave)
		}
		if !slices.HasString(filteredFilePaths, fullFilePath) {
			return errFunc()
		}
	}

	return nil
}

func determineBaseModelStructure(paths []string, prefix string) userconfig.ModelStructureType {
	filteredPaths := files.FilterPathsWithDirPrefix(paths, prefix)
	prefixLength := len(slices.RemoveEmpties(strings.Split(prefix, "/")))

	numFailedVersionChecks := 0
	numPassedVersionChecks := 0
	for _, path := range filteredPaths {
		splitPath := slices.RemoveEmpties(strings.Split(path, "/"))
		versionStr := splitPath[prefixLength]
		_, err := strconv.ParseInt(versionStr, 10, 64)
		if err != nil {
			numFailedVersionChecks++
			continue
		}
		numPassedVersionChecks++
		if len(splitPath) == prefixLength {
			return userconfig.UnknownModelStructureType
		}
	}

	if numFailedVersionChecks > 0 && numPassedVersionChecks > 0 {
		return userconfig.UnknownModelStructureType
	}
	if numFailedVersionChecks > 0 {
		return userconfig.NonVersionedModelType
	}
	if numPassedVersionChecks > 0 {
		return userconfig.VersionedModelType
	}
	return userconfig.UnknownModelStructureType
}

func getModelVersionsFromPaths(paths []string, prefix string) []string {
	filteredPaths := files.FilterPathsWithDirPrefix(paths, prefix)
	prefixLength := len(slices.RemoveEmpties(strings.Split(prefix, "/")))

	versions := []string{}
	for _, path := range filteredPaths {
		splitPath := slices.RemoveEmpties(strings.Split(path, "/"))
		versions = append(versions, splitPath[prefixLength])
	}

	return slices.UniqueStrings(versions)
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
