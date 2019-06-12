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

const (
	// Shared
	UnknownKey    = "unknown"
	NameKey       = "name"
	KindKey       = "kind"
	InputKey      = "input"
	ComputeKey    = "compute"
	TypeKey       = "type"
	PathKey       = "path"
	OutputTypeKey = "output_type"
	TagsKey       = "tags"

	// input schema options
	OptionalOptKey = "_optional"
	DefaultOptKey  = "_default"
	MinCountOptKey = "_min_count"
	MaxCountOptKey = "_max_count"

	// environment
	DataKey           = "data"
	SchemaKey         = "schema"
	LogLevelKey       = "log_level"
	LimitKey          = "limit"
	NumRowsKey        = "num_rows"
	FractionOfRowsKey = "fraction_of_rows"
	RandomizeKey      = "randomize"
	RandomSeedKey     = "random_seed"

	// templates / embeds
	TemplateKey = "template"
	YAMLKey     = "yaml"
	ArgsKey     = "args"

	// constants
	ValueKey = "value"

	// raw columns
	RequiredKey = "required"
	MinKey      = "min"
	MaxKey      = "max"
	ValuesKey   = "values"

	// aggregator / aggregate
	AggregatorKey     = "aggregator"
	AggregatorPathKey = "aggregator_path"

	// transformer / transformed_column
	TransformerKey     = "transformer"
	TransformerPathKey = "transformer_path"

	// estimator / model
	EstimatorKey                 = "estimator"
	EstimatorPathKey             = "estimator_path"
	TrainingInputKey             = "training_input"
	HparamsKey                   = "hparams"
	TargetColumnKey              = "target_column"
	PredictionKeyKey             = "prediction_key"
	DataPartitionRatioKey        = "data_partition_ratio"
	TrainingKey                  = "training"
	EvaluationKey                = "evaluation"
	BatchSizeKey                 = "batch_size"
	NumStepsKey                  = "num_steps"
	NumEpochsKey                 = "num_epochs"
	ShuffleKey                   = "shuffle"
	TfRandomSeedKey              = "tf_random_seed"
	TfRandomizeSeedKey           = "tf_randomize_seed"
	SaveSummaryStepsKey          = "save_summary_steps"
	SaveCheckpointsSecsKey       = "save_checkpoints_secs"
	SaveCheckpointsStepsKey      = "save_checkpoints_steps"
	LogStepCountStepsKey         = "log_step_count_steps"
	KeepCheckpointMaxKey         = "keep_checkpoint_max"
	KeepCheckpointEveryNHoursKey = "keep_checkpoint_every_n_hours"
	StartDelaySecsKey            = "start_delay_secs"
	ThrottleSecsKey              = "throttle_secs"
	DatasetComputeKey            = "dataset_compute"

	// API
	ModelKey     = "model"
	ModelNameKey = "model_name"
)
