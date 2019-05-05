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

package userconfig

import (
	"math/rand"

	cr "github.com/cortexlabs/cortex/pkg/lib/configreader"
	"github.com/cortexlabs/cortex/pkg/lib/errors"
	"github.com/cortexlabs/cortex/pkg/lib/pointer"
	"github.com/cortexlabs/cortex/pkg/lib/slices"
	"github.com/cortexlabs/cortex/pkg/operator/api/resource"
)

type Models []*Model

type Model struct {
	ResourceFields
	Type               ModelType                `json:"type" yaml:"type"`
	Path               string                   `json:"path" yaml:"path"`
	TargetColumn       string                   `json:"target_column" yaml:"target_column"`
	PredictionKey      string                   `json:"prediction_key" yaml:"prediction_key"`
	FeatureColumns     []string                 `json:"feature_columns" yaml:"feature_columns"`
	TrainingColumns    []string                 `json:"training_columns" yaml:"training_columns"`
	Aggregates         []string                 `json:"aggregates"  yaml:"aggregates"`
	Hparams            map[string]interface{}   `json:"hparams" yaml:"hparams"`
	DataPartitionRatio *ModelDataPartitionRatio `json:"data_partition_ratio" yaml:"data_partition_ratio"`
	Training           *ModelTraining           `json:"training" yaml:"training"`
	Evaluation         *ModelEvaluation         `json:"evaluation" yaml:"evaluation"`
	Compute            *TFCompute               `json:"compute" yaml:"compute"`
	DatasetCompute     *SparkCompute            `json:"dataset_compute" yaml:"dataset_compute"`
	Tags               Tags                     `json:"tags" yaml:"tags"`
}

var modelValidation = &cr.StructValidation{
	StructFieldValidations: []*cr.StructFieldValidation{
		{
			StructField: "Name",
			StringValidation: &cr.StringValidation{
				Required:                   true,
				AlphaNumericDashUnderscore: true,
			},
		},
		{
			StructField:      "Path",
			StringValidation: &cr.StringValidation{},
			DefaultField:     "Name",
			DefaultFieldFunc: func(name interface{}) interface{} {
				return "implementations/models/" + name.(string) + ".py"
			},
		},
		{
			StructField: "Type",
			StringValidation: &cr.StringValidation{
				Default:       ClassificationModelType.String(),
				AllowedValues: ModelTypeStrings(),
			},
			Parser: func(str string) (interface{}, error) {
				return ModelTypeFromString(str), nil
			},
		},
		{
			StructField: "TargetColumn",
			StringValidation: &cr.StringValidation{
				Required: true,
			},
		},
		{
			StructField: "PredictionKey",
			StringValidation: &cr.StringValidation{
				Default:    "",
				AllowEmpty: true,
			},
		},
		{
			StructField: "FeatureColumns",
			StringListValidation: &cr.StringListValidation{
				Required:     true,
				DisallowDups: true,
			},
		},
		{
			StructField: "TrainingColumns",
			StringListValidation: &cr.StringListValidation{
				AllowEmpty:   true,
				DisallowDups: true,
				Default:      make([]string, 0),
			},
		},
		{
			StructField: "Aggregates",
			StringListValidation: &cr.StringListValidation{
				AllowEmpty: true,
				Default:    make([]string, 0),
			},
		},
		{
			StructField: "Hparams",
			InterfaceMapValidation: &cr.InterfaceMapValidation{
				AllowEmpty: true,
				Default:    make(map[string]interface{}),
			},
		},
		{
			StructField:      "DataPartitionRatio",
			StructValidation: modelDataPartitionRatioValidation,
		},
		{
			StructField:      "Training",
			StructValidation: modelTrainingValidation,
		},
		{
			StructField:      "Evaluation",
			StructValidation: modelEvaluationValidation,
		},
		tfComputeFieldValidation,
		sparkComputeFieldValidation("DatasetCompute"),
		tagsFieldValidation,
		typeFieldValidation,
	},
}

type ModelDataPartitionRatio struct {
	Training   *float64 `json:"training"`
	Evaluation *float64 `json:"evaluation"`
}

var modelDataPartitionRatioValidation = &cr.StructValidation{
	StructFieldValidations: []*cr.StructFieldValidation{
		{
			StructField: "Training",
			Float64PtrValidation: &cr.Float64PtrValidation{
				GreaterThan: pointer.Float64(0),
			},
		},
		{
			StructField: "Evaluation",
			Float64PtrValidation: &cr.Float64PtrValidation{
				GreaterThan: pointer.Float64(0),
			},
		},
	},
}

type ModelTraining struct {
	BatchSize                 int64  `json:"batch_size" yaml:"batch_size"`
	NumSteps                  *int64 `json:"num_steps" yaml:"num_steps"`
	NumEpochs                 *int64 `json:"num_epochs" yaml:"num_epochs"`
	Shuffle                   bool   `json:"shuffle" yaml:"shuffle"`
	TfRandomSeed              int64  `json:"tf_random_seed" yaml:"tf_random_seed"`
	TfRandomizeSeed           bool   `json:"tf_randomize_seed" yaml:"tf_randomize_seed"`
	SaveSummarySteps          int64  `json:"save_summary_steps" yaml:"save_summary_steps"`
	SaveCheckpointsSecs       *int64 `json:"save_checkpoints_secs" yaml:"save_checkpoints_secs"`
	SaveCheckpointsSteps      *int64 `json:"save_checkpoints_steps" yaml:"save_checkpoints_steps"`
	LogStepCountSteps         int64  `json:"log_step_count_steps" yaml:"log_step_count_steps"`
	KeepCheckpointMax         int64  `json:"keep_checkpoint_max" yaml:"keep_checkpoint_max"`
	KeepCheckpointEveryNHours int64  `json:"keep_checkpoint_every_n_hours" yaml:"keep_checkpoint_every_n_hours"`
}

var modelTrainingValidation = &cr.StructValidation{
	StructFieldValidations: []*cr.StructFieldValidation{
		{
			StructField: "BatchSize",
			Int64Validation: &cr.Int64Validation{
				GreaterThan: pointer.Int64(0),
				Default:     40,
			},
		},
		{
			StructField: "NumSteps",
			Int64PtrValidation: &cr.Int64PtrValidation{
				GreaterThan: pointer.Int64(0),
			},
		},
		{
			StructField: "NumEpochs",
			Int64PtrValidation: &cr.Int64PtrValidation{
				GreaterThan: pointer.Int64(0),
			},
		},
		{
			StructField: "Shuffle",
			BoolValidation: &cr.BoolValidation{
				Default: true,
			},
		},
		{
			StructField: "TfRandomizeSeed",
			BoolValidation: &cr.BoolValidation{
				Default: false,
			},
		},
		{
			StructField:  "TfRandomSeed",
			DefaultField: "TfRandomizeSeed",
			DefaultFieldFunc: func(randomize interface{}) interface{} {
				if randomize.(bool) == true {
					return rand.Int63()
				}
				return int64(1788)
			},
			Int64Validation: &cr.Int64Validation{},
		},
		{
			StructField: "SaveSummarySteps",
			Int64Validation: &cr.Int64Validation{
				GreaterThan: pointer.Int64(0),
				Default:     100,
			},
		},
		{
			StructField: "SaveCheckpointsSecs",
			Int64PtrValidation: &cr.Int64PtrValidation{
				GreaterThan: pointer.Int64(0),
			},
		},
		{
			StructField: "SaveCheckpointsSteps",
			Int64PtrValidation: &cr.Int64PtrValidation{
				GreaterThan: pointer.Int64(0),
			},
		},
		{
			StructField: "LogStepCountSteps",
			Int64Validation: &cr.Int64Validation{
				GreaterThan: pointer.Int64(0),
				Default:     100,
			},
		},
		{
			StructField: "KeepCheckpointMax",
			Int64Validation: &cr.Int64Validation{
				GreaterThan: pointer.Int64(0),
				Default:     3,
			},
		},
		{
			StructField: "KeepCheckpointEveryNHours",
			Int64Validation: &cr.Int64Validation{
				GreaterThan: pointer.Int64(0),
				Default:     10000,
			},
		},
	},
}

type ModelEvaluation struct {
	BatchSize      int64  `json:"batch_size" yaml:"batch_size"`
	NumSteps       *int64 `json:"num_steps" yaml:"num_steps"`
	NumEpochs      *int64 `json:"num_epochs" yaml:"num_epochs"`
	Shuffle        bool   `json:"shuffle" yaml:"shuffle"`
	StartDelaySecs int64  `json:"start_delay_secs" yaml:"start_delay_secs"`
	ThrottleSecs   int64  `json:"throttle_secs" yaml:"throttle_secs"`
}

var modelEvaluationValidation = &cr.StructValidation{
	StructFieldValidations: []*cr.StructFieldValidation{
		{
			StructField: "BatchSize",
			Int64Validation: &cr.Int64Validation{
				GreaterThan: pointer.Int64(0),
				Default:     40,
			},
		},
		{
			StructField: "NumSteps",
			Int64PtrValidation: &cr.Int64PtrValidation{
				GreaterThan: pointer.Int64(0),
			},
		},
		{
			StructField: "NumEpochs",
			Int64PtrValidation: &cr.Int64PtrValidation{
				GreaterThan: pointer.Int64(0),
			},
		},
		{
			StructField: "Shuffle",
			BoolValidation: &cr.BoolValidation{
				Default: false,
			},
		},
		{
			StructField: "StartDelaySecs",
			Int64Validation: &cr.Int64Validation{
				GreaterThan: pointer.Int64(0),
				Default:     120,
			},
		},
		{
			StructField: "ThrottleSecs",
			Int64Validation: &cr.Int64Validation{
				GreaterThan: pointer.Int64(0),
				Default:     600,
			},
		},
	},
}

func (models Models) Validate() error {
	for _, model := range models {
		if err := model.Validate(); err != nil {
			return err
		}
	}

	resources := make([]Resource, len(models))
	for i, res := range models {
		resources[i] = res
	}

	dups := FindDuplicateResourceName(resources...)
	if len(dups) > 0 {
		return ErrorDuplicateResourceName(dups...)
	}

	return nil
}

func (model *Model) Validate() error {
	if model.DataPartitionRatio.Training == nil && model.DataPartitionRatio.Evaluation == nil {
		model.DataPartitionRatio.Training = pointer.Float64(0.8)
		model.DataPartitionRatio.Evaluation = pointer.Float64(0.2)
	} else if model.DataPartitionRatio.Training == nil || model.DataPartitionRatio.Evaluation == nil {
		return errors.Wrap(ErrorSpecifyAllOrNone(TrainingKey, EvaluationKey), Identify(model), DataPartitionRatioKey)
	}

	if model.Training.SaveCheckpointsSecs == nil && model.Training.SaveCheckpointsSteps == nil {
		model.Training.SaveCheckpointsSecs = pointer.Int64(600)
	} else if model.Training.SaveCheckpointsSecs != nil && model.Training.SaveCheckpointsSteps != nil {
		return errors.Wrap(ErrorSpecifyOnlyOne(SaveCheckpointSecsKey, SaveCheckpointStepsKey), Identify(model), TrainingKey)
	}

	if model.Training.NumSteps == nil && model.Training.NumEpochs == nil {
		model.Training.NumSteps = pointer.Int64(1000)
	} else if model.Training.NumSteps != nil && model.Training.NumEpochs != nil {
		return errors.Wrap(ErrorSpecifyOnlyOne(NumEpochsKey, NumStepsKey), Identify(model), TrainingKey)
	}

	if model.Evaluation.NumSteps == nil && model.Evaluation.NumEpochs == nil {
		model.Evaluation.NumSteps = pointer.Int64(100)
	} else if model.Evaluation.NumSteps != nil && model.Evaluation.NumEpochs != nil {
		return errors.Wrap(ErrorSpecifyOnlyOne(NumEpochsKey, NumStepsKey), Identify(model), EvaluationKey)
	}

	for _, trainingColumn := range model.TrainingColumns {
		if slices.HasString(model.FeatureColumns, trainingColumn) {
			return errors.Wrap(ErrorDuplicateResourceValue(trainingColumn, TrainingColumnsKey, FeatureColumnsKey), Identify(model))
		}
	}

	return nil
}

func (model *Model) AllColumnNames() []string {
	return slices.MergeStrSlices(model.FeatureColumns, model.TrainingColumns, []string{model.TargetColumn})
}

func (model *Model) GetResourceType() resource.Type {
	return resource.ModelType
}

func (models Models) Names() []string {
	names := make([]string, len(models))
	for i, model := range models {
		names[i] = model.Name
	}
	return names
}
