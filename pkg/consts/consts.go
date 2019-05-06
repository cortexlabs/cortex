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

package consts

import (
	"regexp"
)

var (
	CortexVersion = "master" // CORTEX_VERSION

	SingleTypeStrRegex   = regexp.MustCompile(`"(INT|FLOAT|STRING|BOOL)(_COLUMN)?"`)
	CompoundTypeStrRegex = regexp.MustCompile(`"(INT|FLOAT|STRING|BOOL)(_COLUMN)?(\|(INT|FLOAT|STRING|BOOL)(_COLUMN)?)+"`)

	ContextCacheDir    = "/mnt/context"
	EmptyDirMountPath  = "/mnt"
	EmptyDirVolumeName = "mnt"

	CortexConfigPath = "/configs/cortex"
	CortexConfigName = "cortex-config"

	RequirementsTxt = "requirements.txt"
	PackageDir      = "packages"

	AppsDir               = "apps"
	APIsDir               = "apis"
	DataDir               = "data"
	RawDataDir            = "data_raw"
	TrainingDataDir       = "data_training"
	AggregatorsDir        = "aggregators"
	AggregatesDir         = "aggregates"
	TransformersDir       = "transformers"
	ModelImplsDir         = "model_implementations"
	PythonPackagesDir     = "python_packages"
	ModelsDir             = "models"
	ConstantsDir          = "constants"
	ContextsDir           = "contexts"
	ResourceStatusesDir   = "resource_statuses"
	WorkloadSpecsDir      = "workload_specs"
	LogPrefixesDir        = "log_prefixes"
	RawColumnsDir         = "raw_columns"
	TransformedColumnsDir = "transformed_columns"

	TelemetryURL = "https://telemetry.cortexlabs.dev"
)
