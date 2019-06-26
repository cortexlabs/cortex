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
	"fmt"
	"math/rand"
	"strings"

	"github.com/cortexlabs/yaml"

	cr "github.com/cortexlabs/cortex/pkg/lib/configreader"
	"github.com/cortexlabs/cortex/pkg/lib/errors"
	"github.com/cortexlabs/cortex/pkg/lib/pointer"
	s "github.com/cortexlabs/cortex/pkg/lib/strings"
	"github.com/cortexlabs/cortex/pkg/operator/api/resource"
)

type Models []*Model

type Model struct {
	ResourceFields
	Estimator          string                   `json:"estimator" yaml:"estimator"`
	EstimatorPath      *string                  `json:"estimator_path" yaml:"estimator_path"`
	TargetColumn       string                   `json:"target_column" yaml:"target_column"`
	Input              interface{}              `json:"input" yaml:"input"`
	TrainingInput      interface{}              `json:"training_input" yaml:"training_input"`
	Hparams            interface{}              `json:"hparams"  yaml:"hparams"`
	PredictionKey      string                   `json:"prediction_key" yaml:"prediction_key"`
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
			StructField: "Estimator",
			StringValidation: &cr.StringValidation{
				AllowEmpty:                           true,
				AlphaNumericDashDotUnderscoreOrEmpty: true,
			},
		},
		{
			StructField:         "EstimatorPath",
			StringPtrValidation: &cr.StringPtrValidation{},
		},
		{
			StructField: "TargetColumn",
			StringValidation: &cr.StringValidation{
				Required:               true,
				RequireCortexResources: true,
			},
		},
		{
			StructField: "Input",
			InterfaceValidation: &cr.InterfaceValidation{
				Required:             true,
				AllowCortexResources: true,
			},
		},
		{
			StructField: "TrainingInput",
			InterfaceValidation: &cr.InterfaceValidation{
				Required:             false,
				AllowCortexResources: true,
			},
		},
		{
			StructField: "Hparams",
			InterfaceValidation: &cr.InterfaceValidation{
				Required: false,
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

func (model *Model) UserConfigStr() string {
	var sb strings.Builder
	sb.WriteString(model.ResourceFields.UserConfigStr())
	if model.EstimatorPath != nil {
		sb.WriteString(fmt.Sprintf("%s: %s\n", EstimatorPathKey, *model.EstimatorPath))
	} else {
		sb.WriteString(fmt.Sprintf("%s: %s\n", EstimatorKey, model.Estimator))
	}
	sb.WriteString(fmt.Sprintf("%s: %s\n", TargetColumnKey, yaml.UnescapeAtSymbol(model.TargetColumn)))
	sb.WriteString(fmt.Sprintf("%s: %s\n", InputKey, s.Obj(model.Input)))
	if model.TrainingInput != nil {
		sb.WriteString(fmt.Sprintf("%s: %s\n", TrainingInputKey, s.Obj(model.TrainingInput)))
	}
	if model.Hparams != nil {
		sb.WriteString(fmt.Sprintf("%s: %s\n", HparamsKey, s.Obj(model.Hparams)))
	}
	if model.PredictionKey != "" {
		sb.WriteString(fmt.Sprintf("%s: %s\n", PredictionKeyKey, model.PredictionKey))
	}
	if model.DataPartitionRatio != nil {
		sb.WriteString(fmt.Sprintf("%s:\n", DataPartitionRatioKey))
		sb.WriteString(s.Indent(model.DataPartitionRatio.UserConfigStr(), "  "))
	}
	if model.Training != nil {
		sb.WriteString(fmt.Sprintf("%s:\n", TrainingKey))
		sb.WriteString(s.Indent(model.Training.UserConfigStr(), "  "))
	}
	if model.Evaluation != nil {
		sb.WriteString(fmt.Sprintf("%s:\n", EvaluationKey))
		sb.WriteString(s.Indent(model.Evaluation.UserConfigStr(), "  "))
	}
	if model.Compute != nil {
		if tfComputeStr := model.Compute.UserConfigStr(); tfComputeStr != "" {
			sb.WriteString(fmt.Sprintf("%s:\n", ComputeKey))
			sb.WriteString(s.Indent(tfComputeStr, "  "))
		}
	}
	if model.DatasetCompute != nil {
		sb.WriteString(fmt.Sprintf("%s:\n", DatasetComputeKey))
		sb.WriteString(s.Indent(model.DatasetCompute.UserConfigStr(), "  "))
	}
	return sb.String()
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

func (mdpr *ModelDataPartitionRatio) UserConfigStr() string {
	var sb strings.Builder
	if mdpr.Training != nil {
		sb.WriteString(fmt.Sprintf("%s: %s\n", TrainingKey, s.Float64(*mdpr.Training)))
	}
	if mdpr.Evaluation != nil {
		sb.WriteString(fmt.Sprintf("%s: %s\n", EvaluationKey, s.Float64(*mdpr.Evaluation)))
	}
	return sb.String()
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

func (mt *ModelTraining) UserConfigStr() string {
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("%s: %s\n", BatchSizeKey, s.Int64(mt.BatchSize)))
	if mt.NumSteps != nil {
		sb.WriteString(fmt.Sprintf("%s: %s\n", NumStepsKey, s.Int64(*mt.NumSteps)))
	}
	if mt.NumEpochs != nil {
		sb.WriteString(fmt.Sprintf("%s: %s\n", NumEpochsKey, s.Int64(*mt.NumEpochs)))
	}
	sb.WriteString(fmt.Sprintf("%s: %s\n", ShuffleKey, s.Bool(mt.Shuffle)))
	sb.WriteString(fmt.Sprintf("%s: %s\n", TfRandomSeedKey, s.Int64(mt.TfRandomSeed)))
	sb.WriteString(fmt.Sprintf("%s: %s\n", SaveSummaryStepsKey, s.Int64(mt.SaveSummarySteps)))
	if mt.SaveCheckpointsSecs != nil {
		sb.WriteString(fmt.Sprintf("%s: %s\n", SaveCheckpointsSecsKey, s.Int64(*mt.SaveCheckpointsSecs)))
	}
	if mt.SaveCheckpointsSteps != nil {
		sb.WriteString(fmt.Sprintf("%s: %s\n", SaveCheckpointsStepsKey, s.Int64(*mt.SaveCheckpointsSteps)))
	}
	sb.WriteString(fmt.Sprintf("%s: %s\n", LogStepCountStepsKey, s.Int64(mt.LogStepCountSteps)))
	sb.WriteString(fmt.Sprintf("%s: %s\n", KeepCheckpointEveryNHoursKey, s.Int64(mt.KeepCheckpointEveryNHours)))
	sb.WriteString(fmt.Sprintf("%s: %s\n", KeepCheckpointMaxKey, s.Int64(mt.KeepCheckpointMax)))
	return sb.String()
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

func (me *ModelEvaluation) UserConfigStr() string {
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("%s: %s\n", BatchSizeKey, s.Int64(me.BatchSize)))
	if me.NumSteps != nil {
		sb.WriteString(fmt.Sprintf("%s: %s\n", NumStepsKey, s.Int64(*me.NumSteps)))
	}
	if me.NumEpochs != nil {
		sb.WriteString(fmt.Sprintf("%s: %s\n", NumEpochsKey, s.Int64(*me.NumEpochs)))
	}
	sb.WriteString(fmt.Sprintf("%s: %s\n", ShuffleKey, s.Bool(me.Shuffle)))
	sb.WriteString(fmt.Sprintf("%s: %s\n", StartDelaySecsKey, s.Int64(me.StartDelaySecs)))
	sb.WriteString(fmt.Sprintf("%s: %s\n", ThrottleSecsKey, s.Int64(me.ThrottleSecs)))
	return sb.String()
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
		return errors.Wrap(ErrorSpecifyOnlyOne(SaveCheckpointsSecsKey, SaveCheckpointsStepsKey), Identify(model), TrainingKey)
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

	if model.EstimatorPath == nil && model.Estimator == "" {
		return errors.Wrap(ErrorSpecifyOnlyOneMissing(EstimatorKey, EstimatorPathKey), Identify(model))
	}

	if model.EstimatorPath != nil && model.Estimator != "" {
		return errors.Wrap(ErrorSpecifyOnlyOne(EstimatorKey, EstimatorPathKey), Identify(model))
	}

	if model.Estimator != "" && model.PredictionKey != "" {
		return ErrorPredictionKeyOnModelWithEstimator()
	}

	return nil
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
