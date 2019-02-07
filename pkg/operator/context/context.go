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

	"github.com/cortexlabs/cortex/pkg/api/context"
	"github.com/cortexlabs/cortex/pkg/api/userconfig"
	"github.com/cortexlabs/cortex/pkg/consts"
	"github.com/cortexlabs/cortex/pkg/operator/aws"
	"github.com/cortexlabs/cortex/pkg/utils/util"
)

func New(
	config *userconfig.Config,
	files map[string][]byte,
	ignoreCache bool,
) (*context.Context, error) {

	userTransformers, err := loadUserTransformers(config.Transformers, files)
	if err != nil {
		return nil, err
	}

	userAggregators, err := loadUserAggregators(config.Aggregators, files)
	if err != nil {
		return nil, err
	}

	err = autoGenerateConfig(config, userAggregators, userTransformers)
	if err != nil {
		return nil, err
	}

	ctx := &context.Context{}

	ctx.CortexConfig = getCortexConfig()

	ctx.App = getApp(config.App)

	datasetVersion, err := getOrSetDatasetVersion(ctx.App.Name, ignoreCache)
	if err != nil {
		return nil, err
	}
	ctx.DatasetVersion = datasetVersion

	ctx.Environment = getEnvironment(config, datasetVersion)

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

	constants, err := loadConstants(config.Constants)
	if err != nil {
		return nil, err
	}
	ctx.Constants = constants

	aggregators, err := getAggregators(config, userAggregators)
	if err != nil {
		return nil, err
	}
	ctx.Aggregators = aggregators

	transformers, err := getTransformers(config, userTransformers)
	if err != nil {
		return nil, err
	}
	ctx.Transformers = transformers

	rawFeatures, err := getRawFeatures(config, ctx.Environment)
	if err != nil {
		return nil, err
	}
	ctx.RawFeatures = rawFeatures

	aggregates, err := getAggregates(config, constants, rawFeatures, userAggregators, ctx.Root)
	if err != nil {
		return nil, err
	}
	ctx.Aggregates = aggregates

	transformedFeatures, err := getTransformedFeatures(config, constants, rawFeatures, ctx.Aggregates, userTransformers, ctx.Root)
	if err != nil {
		return nil, err
	}
	ctx.TransformedFeatures = transformedFeatures

	models, err := getModels(config, aggregates, ctx.Features(), files, ctx.Root)
	if err != nil {
		return nil, err
	}
	ctx.Models = models

	apis, err := getAPIs(config, ctx.Models)
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
	ids = append(ids, ctx.CortexConfig.ID)
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
	return util.HashStr(strings.Join(ids, ""))
}

func DownloadContext(ctxID string, appName string) (*context.Context, error) {
	s3Key := ctxKey(ctxID, appName)
	var ctxSerial context.ContextSerial

	if err := aws.ReadMsgpackFromS3(&ctxSerial, s3Key); err != nil {
		return nil, err
	}

	return ctxSerial.FromSerial()
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
