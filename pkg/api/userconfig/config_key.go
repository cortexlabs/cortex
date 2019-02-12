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
	UnknownKey         = "unknown"
	NameKey            = "name"
	KindKey            = "kind"
	DataKey            = "data"
	SchemaKey          = "schema"
	ColumnsKey         = "columns"
	FeatureColumnsKey  = "feature_columns"
	TrainingColumnsKey = "training_columns"
	TargetColumnKey    = "target_column"
	AggregatesKey      = "aggregates"
	ModelNameKey       = "model_name"
	InputsKey          = "inputs"
	ArgsKey            = "args"
	TypeKey            = "type"
	AggregatorKey      = "aggregator"
	TransformerKey     = "transformer"
	PathKey            = "path"
	ValueKey           = "value"

	// environment
	SubsetKey   = "subset"
	LimitKey    = "limit"
	FractionKey = "fraction"

	// model
	NumEpochsKey           = "num_epochs"
	NumStepsKey            = "num_steps"
	SaveCheckpointSecsKey  = "save_checkpoints_secs"
	SaveCheckpointStepsKey = "save_checkpoints_steps"
	DataPartitionRatioKey  = "data_partition_ratio"
	TrainingKey            = "training"
	EvaluationKey          = "evaluation"
)
