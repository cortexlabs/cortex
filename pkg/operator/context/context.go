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
	"path/filepath"
	"sort"
	"strings"

	"github.com/cortexlabs/cortex/pkg/consts"
	"github.com/cortexlabs/cortex/pkg/lib/configreader"
	"github.com/cortexlabs/cortex/pkg/lib/errors"
	"github.com/cortexlabs/cortex/pkg/lib/files"
	"github.com/cortexlabs/cortex/pkg/lib/hash"
	"github.com/cortexlabs/cortex/pkg/lib/pointer"
	"github.com/cortexlabs/cortex/pkg/lib/sets/strset"
	"github.com/cortexlabs/cortex/pkg/operator/api/context"
	"github.com/cortexlabs/cortex/pkg/operator/api/userconfig"
	"github.com/cortexlabs/cortex/pkg/operator/config"
)

var (
	builtinAggregators   = make(map[string]*context.Aggregator)
	uploadedAggregators  = strset.New()
	builtinTransformers  = make(map[string]*context.Transformer)
	uploadedTransformers = strset.New()

	OperatorTransformersDir = configreader.MustStringFromEnv(
		"CONST_OPERATOR_TRANSFORMERS_DIR",
		&configreader.StringValidation{Default: "/src/transformers"},
	)
	OperatorAggregatorsDir = configreader.MustStringFromEnv(
		"CONST_OPERATOR_AGGREGATORS_DIR",
		&configreader.StringValidation{Default: "/src/aggregators"},
	)
)

func Init() error {
	aggregatorConfigPath := filepath.Join(OperatorAggregatorsDir, "aggregators.yaml")
	aggregatorConfig, err := userconfig.NewPartialPath(aggregatorConfigPath)
	if err != nil {
		return err
	}

	for _, aggregatorConfig := range aggregatorConfig.Aggregators {
		implPath := filepath.Join(OperatorAggregatorsDir, aggregatorConfig.Path)
		impl, err := files.ReadFileBytes(implPath)
		if err != nil {
			return errors.Wrap(err, userconfig.Identify(aggregatorConfig))
		}
		aggregator, err := newAggregator(*aggregatorConfig, impl, pointer.String("cortex"), nil)
		if err != nil {
			return err
		}
		builtinAggregators["cortex."+aggregatorConfig.Name] = aggregator
	}

	transformerConfigPath := filepath.Join(OperatorTransformersDir, "transformers.yaml")
	transformerConfig, err := userconfig.NewPartialPath(transformerConfigPath)
	if err != nil {
		return err
	}

	for _, transConfig := range transformerConfig.Transformers {
		implPath := filepath.Join(OperatorTransformersDir, transConfig.Path)
		impl, err := files.ReadFileBytes(implPath)
		if err != nil {
			return errors.Wrap(err, userconfig.Identify(transConfig))
		}
		transformer, err := newTransformer(*transConfig, impl, pointer.String("cortex"), nil)
		if err != nil {
			return err
		}
		builtinTransformers["cortex."+transConfig.Name] = transformer
	}

	return nil
}

func New(
	userconf *userconfig.Config,
	files map[string][]byte,
	ignoreCache bool,
) (*context.Context, error) {
	ctx := &context.Context{}

	ctx.CortexConfig = config.Cortex

	ctx.App = getApp(userconf.App)

	datasetVersion, err := getOrSetDatasetVersion(ctx.App.Name, ignoreCache)
	if err != nil {
		return nil, err
	}
	ctx.DatasetVersion = datasetVersion

	ctx.Environment = getEnvironment(userconf, datasetVersion)

	ctx.Root = filepath.Join(
		consts.AppsDir,
		ctx.App.Name,
		consts.DataDir,
		ctx.DatasetVersion,
		ctx.Environment.ID,
	)
	ctx.RawDataset = context.RawDataset{
		Key:         filepath.Join(ctx.Root, consts.RawDataDir, "raw.parquet"),
		MetadataKey: filepath.Join(ctx.Root, consts.RawDataDir, "metadata.json"),
	}

	ctx.StatusPrefix = StatusPrefix(ctx.App.Name)

	pythonPackages, err := loadPythonPackages(files, ctx.DatasetVersion)
	if err != nil {
		return nil, err
	}
	ctx.PythonPackages = pythonPackages

	userTransformers, err := loadUserTransformers(userconf, files, pythonPackages)
	if err != nil {
		return nil, err
	}

	userAggregators, err := loadUserAggregators(userconf, files, pythonPackages)
	if err != nil {
		return nil, err
	}

	err = autoGenerateConfig(userconf, userAggregators, userTransformers)
	if err != nil {
		return nil, err
	}

	constants, err := loadConstants(userconf.Constants)
	if err != nil {
		return nil, err
	}
	ctx.Constants = constants

	aggregators, err := getAggregators(userconf, userAggregators)
	if err != nil {
		return nil, err
	}
	ctx.Aggregators = aggregators

	transformers, err := getTransformers(userconf, userTransformers)
	if err != nil {
		return nil, err
	}
	ctx.Transformers = transformers

	rawColumns, err := getRawColumns(userconf, ctx.Environment)
	if err != nil {
		return nil, err
	}
	ctx.RawColumns = rawColumns

	aggregates, err := getAggregates(userconf, constants, rawColumns, userAggregators, ctx.Root)
	if err != nil {
		return nil, err
	}
	ctx.Aggregates = aggregates

	transformedColumns, err := getTransformedColumns(userconf, constants, rawColumns, ctx.Aggregates, userTransformers, ctx.Root)
	if err != nil {
		return nil, err
	}
	ctx.TransformedColumns = transformedColumns

	models, err := getModels(userconf, aggregates, ctx.Columns(), files, ctx.Root, pythonPackages)
	if err != nil {
		return nil, err
	}
	ctx.Models = models

	apis, err := getAPIs(userconf, ctx.Models)
	if err != nil {
		return nil, err
	}
	ctx.APIs = apis

	err = ctx.Validate()
	if err != nil {
		return nil, err
	}

	ctx.ID = calculateID(ctx)
	ctx.Key = ctxKey(ctx.ID, ctx.App.Name)
	return ctx, nil
}

func ctxKey(ctxID string, appName string) string {
	return filepath.Join(
		consts.AppsDir,
		appName,
		consts.ContextsDir,
		ctxID+".msgpack",
	)
}

func calculateID(ctx *context.Context) string {
	ids := []string{}
	ids = append(ids, config.Cortex.ID)
	ids = append(ids, ctx.DatasetVersion)
	ids = append(ids, ctx.Root)
	ids = append(ids, ctx.RawDataset.Key)
	ids = append(ids, ctx.StatusPrefix)
	ids = append(ids, ctx.App.ID)
	ids = append(ids, ctx.Environment.ID)

	for _, resource := range ctx.AllResources() {
		ids = append(ids, resource.GetID())
	}

	sort.Strings(ids)
	return hash.String(strings.Join(ids, ""))
}

func DownloadContext(ctxID string, appName string) (*context.Context, error) {
	s3Key := ctxKey(ctxID, appName)
	var serial context.Serial

	if err := config.AWS.ReadMsgpackFromS3(&serial, s3Key); err != nil {
		return nil, err
	}

	return serial.ContextFromSerial()
}

func StatusPrefix(appName string) string {
	return filepath.Join(
		consts.AppsDir,
		appName,
		consts.ResourceStatusesDir,
	)
}

func StatusKey(resourceID string, workloadID string, appName string) string {
	return filepath.Join(
		StatusPrefix(appName),
		resourceID,
		workloadID,
	)
}

func LatestWorkloadIDKey(resourceID string, appName string) string {
	return filepath.Join(
		StatusPrefix(appName),
		resourceID,
		"latest",
	)
}

func WorkloadSpecKey(workloadID string, appName string) string {
	return filepath.Join(
		consts.AppsDir,
		appName,
		consts.WorkloadSpecsDir,
		workloadID,
	)
}
