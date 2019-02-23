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
	"sort"

	"github.com/cortexlabs/cortex/pkg/api/resource"
	s "github.com/cortexlabs/cortex/pkg/api/strings"
	"github.com/cortexlabs/cortex/pkg/api/userconfig"
	"github.com/cortexlabs/cortex/pkg/utils/errors"
	"github.com/cortexlabs/cortex/pkg/utils/sets/strset"
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
	userconfig.ResourceConfigFields
	*ComputedResourceFields
	ModelName   string `json:"model_name"`
	TrainKey    string `json:"train_key"`
	EvalKey     string `json:"eval_key"`
	MetadataKey string `json:"metadata_key"`
}

func (trainingDataset *TrainingDataset) GetResourceType() resource.Type {
	return resource.TrainingDatasetType
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

func (ctx *Context) RawColumnInputNames(model *Model) []string {
	rawColumnInputNames := strset.New()
	for _, colName := range model.FeatureColumns {
		col := ctx.GetColumn(colName)
		rawColumnInputNames.Add(col.GetInputRawColumnNames()...)
	}
	list := rawColumnInputNames.List()
	sort.Strings(list)
	return list
}
