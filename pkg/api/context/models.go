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
	"github.com/cortexlabs/cortex/pkg/api/resource"
	s "github.com/cortexlabs/cortex/pkg/api/strings"
	"github.com/cortexlabs/cortex/pkg/api/userconfig"
	"github.com/cortexlabs/cortex/pkg/utils/errors"
)

type Models map[string]*Model
type TrainingDatasets map[string]*TrainingDataset

type Model struct {
	*userconfig.Model
	*ComputedResourceFields
	Key     string           `json:"key"`
	ImplID  string           `json:"impl_id"`
	ImplKey string           `json:"impl_key"`
	Dataset *TrainingDataset `json:"dataset"`
}

type TrainingDataset struct {
	*ComputedResourceFields
	Name        string            `json:"name"`
	ModelName   string            `json:"model_name"`
	TrainKey    string            `json:"train_key"`
	EvalKey     string            `json:"eval_key"`
	MetadataKey string            `json:"metadata_key"`
	FilePath    string            `json:"file_path"`
	Embed       *userconfig.Embed `json:"embed"`
}

func (trainingDataset *TrainingDataset) GetName() string {
	return trainingDataset.Name
}

func (trainingDataset *TrainingDataset) GetResourceType() resource.Type {
	return resource.TrainingDatasetType
}

func (trainingDataset *TrainingDataset) GetFilePath() string {
	return trainingDataset.FilePath
}

func (trainingDataset *TrainingDataset) GetEmbed() *userconfig.Embed {
	return trainingDataset.Embed
}

func (models Models) OneByID(id string) *Model {
	for _, model := range models {
		if model.ID == id {
			return model
		}
	}
	return nil
}

func (ctx *Context) OneTrainingDatasetByID(id string) *TrainingDataset {
	for _, model := range ctx.Models {
		if model.Dataset.ID == id {
			return model.Dataset
		}
	}
	return nil
}

func (models Models) GetTrainingDatasets() TrainingDatasets {
	trainingDatasets := make(map[string]*TrainingDataset, len(models))
	for _, model := range models {
		trainingDatasets[model.Dataset.Name] = model.Dataset
	}
	return trainingDatasets
}

func ValidateModelTargetType(targetDataTypeStr string, modelType string) error {
	targetType := userconfig.ColumnTypeFromString(targetDataTypeStr)
	switch modelType {
	case "classification":
		if targetType != userconfig.IntegerColumnType {
			return errors.New(s.ErrClassificationTargetType)
		}
		return nil
	case "regression":
		if targetType != userconfig.IntegerColumnType && targetType != userconfig.FloatColumnType {
			return errors.New(s.ErrRegressionTargetType)
		}
		return nil
	}

	return errors.New(s.ErrInvalidStr(modelType, "classification", "regression")) // unexpected
}
