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

	"github.com/cortexlabs/cortex/pkg/api/resource"
	cr "github.com/cortexlabs/cortex/pkg/utils/configreader"
	"github.com/cortexlabs/cortex/pkg/utils/errors"
	"github.com/cortexlabs/cortex/pkg/utils/util"
)

type Models []*Model

type Model struct {
	Name               string                   `json:"name" yaml:"name"`
	Type               string                   `json:"type" yaml:"type"`
	Path               string                   `json:"path" yaml:"path"`
	Target             string                   `json:"target" yaml:"target"`
	PredictionKey      string                   `json:"prediction_key" yaml:"prediction_key"`
	Features           []string                 `json:"features" yaml:"features"`
	TrainingFeatures   []string                 `json:"training_features" yaml:"training_features"`
	Aggregates         []string                 `json:"aggregates"  yaml:"aggregates"`
	Hparams            map[string]interface{}   `json:"hparams" yaml:"hparams"`
	DataPartitionRatio *ModelDataPartitionRatio `json:"data_partition_ratio" yaml:"data_partition_ratio"`
	Training           *ModelTraining           `json:"training" yaml:"training"`
	Evaluation         *ModelEvaluation         `json:"evaluation" yaml:"evaluation"`
	Misc               *ModelMisc               `json:"misc" yaml:"misc"`
	Compute            *TFCompute               `json:"compute" yaml:"compute"`
	Tags               Tags                     `json:"tags" yaml:"tags"`
}

var modelValidation = &cr.StructValidation{
	StructFieldValidations: []*cr.StructFieldValidation{
		&cr.StructFieldValidation{
			StructField: "Name",
			StringValidation: &cr.StringValidation{
				Required:                   true,
				AlphaNumericDashUnderscore: true,
			},
		},
		&cr.StructFieldValidation{
			StructField:      "Path",
			StringValidation: &cr.StringValidation{},
			DefaultField:     "Name",
			DefaultFieldFunc: func(name interface{}) interface{} {
				return "implementations/models/" + name.(string) + ".py"
			},
		},
		&cr.StructFieldValidation{
			StructField: "Type",
			StringValidation: &cr.StringValidation{
				Default:       "classification",
				AllowedValues: []string{"classification", "regression"},
			},
		},
		&cr.StructFieldValidation{
			StructField: "Target",
			StringValidation: &cr.StringValidation{
				Required: true,
			},
		},
		&cr.StructFieldValidation{
			StructField: "PredictionKey",
			StringValidation: &cr.StringValidation{
				Default:    "",
				AllowEmpty: true,
			},
		},
		&cr.StructFieldValidation{
			StructField: "Features",
			StringListValidation: &cr.StringListValidation{
				Required:     true,
				DisallowDups: true,
			},
		},
		&cr.StructFieldValidation{
			StructField: "TrainingFeatures",
			StringListValidation: &cr.StringListValidation{
				AllowEmpty:   true,
				DisallowDups: true,
				Default:      make([]string, 0),
			},
		},
		&cr.StructFieldValidation{
			StructField: "Aggregates",
			StringListValidation: &cr.StringListValidation{
				AllowEmpty: true,
				Default:    make([]string, 0),
			},
		},
		&cr.StructFieldValidation{
			StructField: "Hparams",
			InterfaceMapValidation: &cr.InterfaceMapValidation{
				AllowEmpty: true,
				Default:    make(map[string]interface{}),
			},
		},
		&cr.StructFieldValidation{
			StructField:      "DataPartitionRatio",
			StructValidation: modelDataPartitionRatioValidation,
		},
		&cr.StructFieldValidation{
			StructField:      "Training",
			StructValidation: modelTrainingValidation,
		},
		&cr.StructFieldValidation{
			StructField:      "Evaluation",
			StructValidation: modelEvaluationValidation,
		},
		&cr.StructFieldValidation{
			StructField:      "Misc",
			StructValidation: modelMiscValidation,
		},
		tfComputeFieldValidation,
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
		&cr.StructFieldValidation{
			StructField: "Training",
			Float64PtrValidation: &cr.Float64PtrValidation{
				GreaterThan: util.Float64Ptr(0),
			},
		},
		&cr.StructFieldValidation{
			StructField: "Evaluation",
			Float64PtrValidation: &cr.Float64PtrValidation{
				GreaterThan: util.Float64Ptr(0),
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
		&cr.StructFieldValidation{
			StructField: "BatchSize",
			Int64Validation: &cr.Int64Validation{
				GreaterThan: util.Int64Ptr(0),
				Default:     40,
			},
		},
		&cr.StructFieldValidation{
			StructField: "NumSteps",
			Int64PtrValidation: &cr.Int64PtrValidation{
				GreaterThan: util.Int64Ptr(0),
			},
		},
		&cr.StructFieldValidation{
			StructField: "NumEpochs",
			Int64PtrValidation: &cr.Int64PtrValidation{
				GreaterThan: util.Int64Ptr(0),
			},
		},
		&cr.StructFieldValidation{
			StructField: "Shuffle",
			BoolValidation: &cr.BoolValidation{
				Default: true,
			},
		},
		&cr.StructFieldValidation{
			StructField: "TfRandomizeSeed",
			BoolValidation: &cr.BoolValidation{
				Default: false,
			},
		},
		&cr.StructFieldValidation{
			StructField:  "TfRandomSeed",
			DefaultField: "TfRandomizeSeed",
			DefaultFieldFunc: func(randomize interface{}) interface{} {
				if randomize.(bool) == true {
					return rand.Int63()
				} else {
					return int64(1788)
				}
			},
			Int64Validation: &cr.Int64Validation{},
		},
		&cr.StructFieldValidation{
			StructField: "SaveSummarySteps",
			Int64Validation: &cr.Int64Validation{
				GreaterThan: util.Int64Ptr(0),
				Default:     100,
			},
		},
		&cr.StructFieldValidation{
			StructField: "SaveCheckpointsSecs",
			Int64PtrValidation: &cr.Int64PtrValidation{
				GreaterThan: util.Int64Ptr(0),
			},
		},
		&cr.StructFieldValidation{
			StructField: "SaveCheckpointsSteps",
			Int64PtrValidation: &cr.Int64PtrValidation{
				GreaterThan: util.Int64Ptr(0),
			},
		},
		&cr.StructFieldValidation{
			StructField: "LogStepCountSteps",
			Int64Validation: &cr.Int64Validation{
				GreaterThan: util.Int64Ptr(0),
				Default:     100,
			},
		},
		&cr.StructFieldValidation{
			StructField: "KeepCheckpointMax",
			Int64Validation: &cr.Int64Validation{
				GreaterThan: util.Int64Ptr(0),
				Default:     3,
			},
		},
		&cr.StructFieldValidation{
			StructField: "KeepCheckpointEveryNHours",
			Int64Validation: &cr.Int64Validation{
				GreaterThan: util.Int64Ptr(0),
				Default:     10000,
			},
		},
	},
}

type ModelEvaluation struct {
	BatchSize      int64 `json:"batch_size" yaml:"batch_size"`
	NumSteps       int64 `json:"num_steps" yaml:"num_steps"`
	Shuffle        bool  `json:"shuffle" yaml:"shuffle"`
	StartDelaySecs int64 `json:"start_delay_secs" yaml:"start_delay_secs"`
	ThrottleSecs   int64 `json:"throttle_secs" yaml:"throttle_secs"`
}

var modelEvaluationValidation = &cr.StructValidation{
	StructFieldValidations: []*cr.StructFieldValidation{
		&cr.StructFieldValidation{
			StructField: "BatchSize",
			Int64Validation: &cr.Int64Validation{
				GreaterThan: util.Int64Ptr(0),
				Default:     40,
			},
		},
		&cr.StructFieldValidation{
			StructField: "NumSteps",
			Int64Validation: &cr.Int64Validation{
				GreaterThan: util.Int64Ptr(0),
				Default:     100,
			},
		},
		&cr.StructFieldValidation{
			StructField: "Shuffle",
			BoolValidation: &cr.BoolValidation{
				Default: false,
			},
		},
		&cr.StructFieldValidation{
			StructField: "StartDelaySecs",
			Int64Validation: &cr.Int64Validation{
				GreaterThan: util.Int64Ptr(0),
				Default:     120,
			},
		},
		&cr.StructFieldValidation{
			StructField: "ThrottleSecs",
			Int64Validation: &cr.Int64Validation{
				GreaterThan: util.Int64Ptr(0),
				Default:     600,
			},
		},
	},
}

type ModelMisc struct {
	Verbosity string `json:"verbosity" yaml:"verbosity"`
}

var modelMiscValidation = &cr.StructValidation{
	StructFieldValidations: []*cr.StructFieldValidation{
		&cr.StructFieldValidation{
			StructField: "Verbosity",
			StringValidation: &cr.StringValidation{
				Default:       "INFO",
				AllowedValues: []string{"DEBUG", "INFO", "WARN", "ERROR", "FATAL"},
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

	dups := util.FindDuplicateStrs(models.Names())
	if len(dups) > 0 {
		return ErrorDuplicateConfigName(dups[0], resource.ModelType)
	}

	return nil
}

func (model *Model) Validate() error {
	if model.DataPartitionRatio.Training == nil && model.DataPartitionRatio.Evaluation == nil {
		model.DataPartitionRatio.Training = util.Float64Ptr(0.8)
		model.DataPartitionRatio.Evaluation = util.Float64Ptr(0.2)
	} else if model.DataPartitionRatio.Training == nil || model.DataPartitionRatio.Evaluation == nil {
		return errors.Wrap(ErrorSpecifyAllOrNone(TrainingKey, EvaluationKey), Identify(model), DataPartitionRatioKey)
	}

	if model.Training.SaveCheckpointsSecs == nil && model.Training.SaveCheckpointsSteps == nil {
		model.Training.SaveCheckpointsSecs = util.Int64Ptr(600)
	} else if model.Training.SaveCheckpointsSecs != nil && model.Training.SaveCheckpointsSteps != nil {
		return errors.Wrap(ErrorSpecifyOnlyOne(SaveCheckpointSecsKey, SaveCheckpointStepsKey), Identify(model), TrainingKey)
	}

	if model.Training.NumSteps == nil && model.Training.NumEpochs == nil {
		model.Training.NumSteps = util.Int64Ptr(1000)
	} else if model.Training.NumSteps != nil && model.Training.NumEpochs != nil {
		return errors.Wrap(ErrorSpecifyOnlyOne(NumEpochsKey, NumStepsKey), Identify(model), TrainingKey)
	}

	for _, trainingFeature := range model.TrainingFeatures {
		if util.IsStrInSlice(trainingFeature, model.Features) {
			return errors.Wrap(ErrorDuplicateResourceValue(trainingFeature, TrainingFeaturesKey, FeaturesKey), Identify(model))
		}
	}

	return nil
}

func (model *Model) AllFeatureNames() []string {
	return util.MergeStrSlices(model.Features, model.TrainingFeatures, []string{model.Target})
}

func (model *Model) GetName() string {
	return model.Name
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
