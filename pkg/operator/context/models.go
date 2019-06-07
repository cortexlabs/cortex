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

	"github.com/cortexlabs/yaml"

	"github.com/cortexlabs/cortex/pkg/consts"
	"github.com/cortexlabs/cortex/pkg/lib/errors"
	"github.com/cortexlabs/cortex/pkg/lib/hash"
	s "github.com/cortexlabs/cortex/pkg/lib/strings"
	"github.com/cortexlabs/cortex/pkg/operator/api/context"
	"github.com/cortexlabs/cortex/pkg/operator/api/resource"
	"github.com/cortexlabs/cortex/pkg/operator/api/userconfig"
)

func getModels(
	config *userconfig.Config,
	constants context.Constants,
	columns context.Columns,
	aggregates context.Aggregates,
	transformedColumns context.TransformedColumns,
	aggregators context.Aggregators,
	transformers context.Transformers,
	estimators context.Estimators,
	root string,
) (context.Models, error) {

	models := context.Models{}

	for _, modelConfig := range config.Models {
		estimator := estimators[modelConfig.Estimator]

		var validInputResources []context.Resource
		for _, res := range constants {
			validInputResources = append(validInputResources, res)
		}
		for _, res := range columns {
			validInputResources = append(validInputResources, res)
		}
		for _, res := range aggregates {
			validInputResources = append(validInputResources, res)
		}
		for _, res := range transformedColumns {
			validInputResources = append(validInputResources, res)
		}

		// Input
		castedInput, inputID, err := ValidateInput(
			modelConfig.Input,
			estimator.Input,
			[]resource.Type{resource.RawColumnType, resource.TransformedColumnType, resource.ConstantType, resource.AggregateType, resource.TransformedColumnType},
			validInputResources,
			config.Resources,
			aggregators,
			transformers,
		)
		if err != nil {
			return nil, errors.Wrap(err, userconfig.Identify(modelConfig), userconfig.InputKey)
		}
		modelConfig.Input = castedInput

		// TrainingInput
		castedTrainingInput, trainingInputID, err := ValidateInput(
			modelConfig.TrainingInput,
			estimator.TrainingInput,
			[]resource.Type{resource.RawColumnType, resource.TransformedColumnType, resource.ConstantType, resource.AggregateType, resource.TransformedColumnType},
			validInputResources,
			config.Resources,
			aggregators,
			transformers,
		)
		if err != nil {
			return nil, errors.Wrap(err, userconfig.Identify(modelConfig), userconfig.TrainingInputKey)
		}
		modelConfig.TrainingInput = castedTrainingInput

		// Hparams
		if estimator.Hparams != nil {
			castedHparams, err := userconfig.CastInputValue(modelConfig.Hparams, estimator.Hparams)
			if err != nil {
				return nil, errors.Wrap(err, userconfig.Identify(modelConfig), userconfig.HparamsKey)
			}
			modelConfig.Hparams = castedHparams
		}

		// TargetColumn
		targetColumnName, _ := yaml.ExtractAtSymbolText(modelConfig.TargetColumn)
		targetColumn := columns[targetColumnName]
		if targetColumn == nil {
			return nil, errors.Wrap(userconfig.ErrorUndefinedResource(targetColumnName, resource.RawColumnType, resource.TransformedColumnType), userconfig.Identify(modelConfig), userconfig.TargetColumnKey)
		}
		if targetColumn.GetColumnType() != userconfig.IntegerColumnType && targetColumn.GetColumnType() != userconfig.FloatColumnType {
			return nil, userconfig.ErrorTargetColumnIntOrFloat()
		}
		if estimator.TargetColumn != userconfig.InferredColumnType {
			if targetColumn.GetColumnType() != estimator.TargetColumn {
				return nil, errors.Wrap(userconfig.ErrorUnsupportedOutputType(targetColumn.GetColumnType(), estimator.TargetColumn), userconfig.Identify(modelConfig), userconfig.TargetColumnKey)
			}
		}

		// Model ID
		var buf bytes.Buffer
		buf.WriteString(inputID)
		buf.WriteString(trainingInputID)
		buf.WriteString(targetColumn.GetID())
		buf.WriteString(s.Obj(modelConfig.Hparams))
		buf.WriteString(s.Obj(modelConfig.DataPartitionRatio))
		buf.WriteString(s.Obj(modelConfig.Training))
		buf.WriteString(s.Obj(modelConfig.Evaluation))
		buf.WriteString(estimator.ID)
		modelID := hash.Bytes(buf.Bytes())

		// Dataset ID
		buf.Reset()
		buf.WriteString(s.Obj(modelConfig.DataPartitionRatio))
		combinedInput := []interface{}{modelConfig.Input, modelConfig.TrainingInput, modelConfig.TargetColumn}
		var columnSlice []context.Resource
		for _, res := range columns {
			columnSlice = append(columnSlice, res)
		}
		for _, col := range context.ExtractCortexResources(combinedInput, columnSlice, resource.RawColumnType, resource.TransformedColumnType) {
			buf.WriteString(col.GetID())
		}
		datasetID := hash.Bytes(buf.Bytes())

		datasetRoot := filepath.Join(root, consts.TrainingDataDir, datasetID)
		trainingDatasetName := strings.Join([]string{
			modelConfig.Name,
			resource.TrainingDatasetType.String(),
		}, "/")

		models[modelConfig.Name] = &context.Model{
			ComputedResourceFields: &context.ComputedResourceFields{
				ResourceFields: &context.ResourceFields{
					ID:           modelID,
					ResourceType: resource.ModelType,
				},
			},
			Model: modelConfig,
			Key:   filepath.Join(root, consts.ModelsDir, modelID+".zip"),
			Dataset: &context.TrainingDataset{
				ResourceFields: userconfig.ResourceFields{
					Name:     trainingDatasetName,
					FilePath: modelConfig.FilePath,
					Embed:    modelConfig.Embed,
				},
				ComputedResourceFields: &context.ComputedResourceFields{
					ResourceFields: &context.ResourceFields{
						ID:           datasetID,
						ResourceType: resource.TrainingDatasetType,
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
