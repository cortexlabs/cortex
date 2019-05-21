/*
Copyright 2019 Cortex Labs, Inc.

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

package context

import (
	"bytes"
	"path/filepath"
	"strings"

	"github.com/cortexlabs/cortex/pkg/consts"
	"github.com/cortexlabs/cortex/pkg/lib/errors"
	"github.com/cortexlabs/cortex/pkg/lib/hash"
	"github.com/cortexlabs/cortex/pkg/lib/sets/strset"
	s "github.com/cortexlabs/cortex/pkg/lib/strings"
	"github.com/cortexlabs/cortex/pkg/operator/api/context"
	"github.com/cortexlabs/cortex/pkg/operator/api/resource"
	"github.com/cortexlabs/cortex/pkg/operator/api/userconfig"
	"github.com/cortexlabs/cortex/pkg/operator/config"
)

var uploadedModels = strset.New()

func getModels(
	config *userconfig.Config,
	aggregates context.Aggregates,
	columns context.Columns,
	impls map[string][]byte,
	root string,
	pythonPackages context.PythonPackages,
) (context.Models, error) {

	models := context.Models{}

	for _, modelConfig := range config.Models {
		modelImplID, modelImplKey, err := getModelImplID(modelConfig.Path, impls)
		if err != nil {
			return nil, errors.Wrap(err, userconfig.Identify(modelConfig), userconfig.PathKey)
		}

		targetDataType := columns[modelConfig.TargetColumn].GetType()
		err = context.ValidateModelTargetType(targetDataType, modelConfig.Type)
		if err != nil {
			return nil, errors.Wrap(err, userconfig.Identify(modelConfig))
		}

		var buf bytes.Buffer
		buf.WriteString(modelConfig.Type.String())
		buf.WriteString(modelImplID)
		for _, pythonPackage := range pythonPackages {
			buf.WriteString(pythonPackage.GetID())
		}
		buf.WriteString(modelConfig.PredictionKey)
		buf.WriteString(s.Obj(modelConfig.Hparams))
		buf.WriteString(s.Obj(modelConfig.DataPartitionRatio))
		buf.WriteString(s.Obj(modelConfig.Training))
		buf.WriteString(s.Obj(modelConfig.Evaluation))
		buf.WriteString(columns.IDWithTags(modelConfig.AllColumnNames())) // A change in tags can invalidate the model

		for _, aggregate := range modelConfig.Aggregates {
			buf.WriteString(aggregates[aggregate].GetID())
		}
		buf.WriteString(modelConfig.Tags.ID())

		modelID := hash.Bytes(buf.Bytes())

		buf.Reset()
		buf.WriteString(s.Obj(modelConfig.DataPartitionRatio))
		buf.WriteString(columns.ID(modelConfig.AllColumnNames()))
		datasetID := hash.Bytes(buf.Bytes())
		buf.WriteString(columns.IDWithTags(modelConfig.AllColumnNames()))
		datasetIDWithTags := hash.Bytes(buf.Bytes())

		datasetRoot := filepath.Join(root, consts.TrainingDataDir, datasetID)
		trainingDatasetName := strings.Join([]string{
			modelConfig.Name,
			resource.TrainingDatasetType.String(),
		}, "/")

		models[modelConfig.Name] = &context.Model{
			ComputedResourceFields: &context.ComputedResourceFields{
				ResourceFields: &context.ResourceFields{
					ID:           modelID,
					IDWithTags:   modelID,
					ResourceType: resource.ModelType,
					MetadataKey:  filepath.Join(root, consts.ModelsDir, modelID+"_metadata.json"),
				},
			},
			Model:   modelConfig,
			Key:     filepath.Join(root, consts.ModelsDir, modelID+".zip"),
			ImplID:  modelImplID,
			ImplKey: modelImplKey,
			Dataset: &context.TrainingDataset{
				ResourceFields: userconfig.ResourceFields{
					Name:     trainingDatasetName,
					FilePath: modelConfig.FilePath,
					Embed:    modelConfig.Embed,
				},
				ComputedResourceFields: &context.ComputedResourceFields{
					ResourceFields: &context.ResourceFields{
						ID:           datasetID,
						IDWithTags:   datasetIDWithTags,
						ResourceType: resource.TrainingDatasetType,
						MetadataKey:  filepath.Join(datasetRoot, "metadata.json"),
					},
				},
				ModelName: modelConfig.Name,
				TrainKey:  filepath.Join(datasetRoot, "train.tfrecord"),
				EvalKey:   filepath.Join(datasetRoot, "eval.tfrecord"),
			},
		}
	}

	return models, nil
}

func getModelImplID(implPath string, impls map[string][]byte) (string, string, error) {
	impl, ok := impls[implPath]
	if !ok {
		return "", "", ErrorImplDoesNotExist(implPath)
	}
	modelImplID := hash.Bytes(impl)
	modelImplKey, err := uploadModelImpl(modelImplID, impl)
	if err != nil {
		return "", "", errors.Wrap(err, implPath)
	}
	return modelImplID, modelImplKey, nil
}

func uploadModelImpl(modelImplID string, impl []byte) (string, error) {
	modelImplKey := filepath.Join(
		consts.ModelImplsDir,
		modelImplID+".py",
	)

	if uploadedModels.Has(modelImplID) {
		return modelImplKey, nil
	}

	isUploaded, err := config.AWS.IsS3File(modelImplKey)
	if err != nil {
		return "", errors.Wrap(err, "upload")
	}

	if !isUploaded {
		err = config.AWS.UploadBytesToS3(impl, modelImplKey)
		if err != nil {
			return "", errors.Wrap(err, "upload")
		}
	}

	uploadedModels.Add(modelImplID)
	return modelImplKey, nil
}
