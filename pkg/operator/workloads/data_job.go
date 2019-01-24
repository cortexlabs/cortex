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

package workloads

import (
	"path/filepath"
	"strings"

	sparkop "github.com/GoogleCloudPlatform/spark-on-k8s-operator/pkg/apis/sparkoperator.k8s.io/v1alpha1"

	"github.com/cortexlabs/cortex/pkg/api/context"
	s "github.com/cortexlabs/cortex/pkg/api/strings"
	"github.com/cortexlabs/cortex/pkg/api/userconfig"
	"github.com/cortexlabs/cortex/pkg/operator/argo"
	"github.com/cortexlabs/cortex/pkg/operator/aws"
	"github.com/cortexlabs/cortex/pkg/operator/spark"
	"github.com/cortexlabs/cortex/pkg/utils/errors"
	"github.com/cortexlabs/cortex/pkg/utils/sets/strset"
)

func dataJobSpec(
	ctx *context.Context,
	shouldIngest bool,
	rawFeatures strset.Set,
	aggregates strset.Set,
	transformedFeatures strset.Set,
	trainingDatasets strset.Set,
	workloadID string,
	sparkCompute *userconfig.SparkCompute,
) *sparkop.SparkApplication {

	args := []string{
		"--raw-features=" + strings.Join(rawFeatures.List(), ","),
		"--aggregates=" + strings.Join(aggregates.List(), ","),
		"--transformed-features=" + strings.Join(transformedFeatures.List(), ","),
		"--training-datasets=" + strings.Join(trainingDatasets.List(), ","),
	}
	if shouldIngest {
		args = append(args, "--ingest")
	}
	spec := spark.SparkSpec(workloadID, ctx, workloadTypeData, sparkCompute, args...)
	argo.EnableGC(spec)
	return spec
}

func dataWorkloadSpecs(ctx *context.Context) ([]*WorkloadSpec, error) {
	workloadID := generateWorkloadID()

	rawFileExists, err := aws.IsS3File(filepath.Join(ctx.RawDatasetKey, "_SUCCESS"))
	if err != nil {
		return nil, errors.Wrap(err, ctx.App.Name, "raw dataset")
	}

	var allComputes []*userconfig.SparkCompute

	shouldIngest := !rawFileExists
	if shouldIngest {
		externalDataPath := ctx.Environment.Data.GetExternalPath()
		externalDataExists, err := aws.IsS3aPrefixExternal(externalDataPath)
		if err != nil || !externalDataExists {
			return nil, errors.New(ctx.App.Name, userconfig.Identify(ctx.Environment), userconfig.DataKey, userconfig.PathKey, s.ErrUserDataUnavailable(externalDataPath))
		}
		for _, rawFeature := range ctx.RawFeatures {
			allComputes = append(allComputes, rawFeature.GetCompute())
		}
	}

	rawFeatureIDs := strset.New()
	var rawFeatures []string
	for rawFeatureName, rawFeature := range ctx.RawFeatures {
		isFeatureCached, err := checkResourceCached(rawFeature, ctx)
		if err != nil {
			return nil, err
		}
		if isFeatureCached {
			continue
		}
		rawFeatures = append(rawFeatures, rawFeatureName)
		rawFeatureIDs.Add(rawFeature.GetID())
		allComputes = append(allComputes, rawFeature.GetCompute())
	}

	aggregateIDs := strset.New()
	var aggregates []string
	for aggregateName, aggregate := range ctx.Aggregates {
		isAggregateCached, err := checkResourceCached(aggregate, ctx)
		if err != nil {
			return nil, err
		}
		if isAggregateCached {
			continue
		}
		aggregates = append(aggregates, aggregateName)
		aggregateIDs.Add(aggregate.GetID())
		allComputes = append(allComputes, aggregate.Compute)
	}

	transformedFeatureIDs := strset.New()
	var transformedFeatures []string
	for transformedFeatureName, transformedFeature := range ctx.TransformedFeatures {
		isFeatureCached, err := checkResourceCached(transformedFeature, ctx)
		if err != nil {
			return nil, err
		}
		if isFeatureCached {
			continue
		}
		transformedFeatures = append(transformedFeatures, transformedFeatureName)
		transformedFeatureIDs.Add(transformedFeature.GetID())
		allComputes = append(allComputes, transformedFeature.Compute)
	}

	trainingDatasetIDs := strset.New()
	var trainingDatasets []string
	for modelName, model := range ctx.Models {
		dataset := model.Dataset
		isTrainingDatasetCached, err := checkResourceCached(dataset, ctx)
		if err != nil {
			return nil, err
		}
		if isTrainingDatasetCached {
			continue
		}
		trainingDatasets = append(trainingDatasets, modelName)
		trainingDatasetIDs.Add(dataset.GetID())
		dependencyIDs := ctx.AllComputedResourceDependencies(dataset.GetID())
		for _, transformedFeature := range ctx.TransformedFeatures {
			if _, ok := dependencyIDs[transformedFeature.ID]; ok {
				allComputes = append(allComputes, transformedFeature.Compute)
			}
		}
	}

	resourceIDSet := strset.Union(rawFeatureIDs, aggregateIDs, transformedFeatureIDs, trainingDatasetIDs)

	if !shouldIngest && len(resourceIDSet) == 0 {
		return nil, nil
	}

	sparkCompute := userconfig.MaxSparkCompute(allComputes...)
	spec := dataJobSpec(ctx, shouldIngest, rawFeatureIDs, aggregateIDs, transformedFeatureIDs, trainingDatasetIDs, workloadID, sparkCompute)

	workloadSpec := &WorkloadSpec{
		WorkloadID:       workloadID,
		ResourceIDs:      resourceIDSet,
		Spec:             spec,
		K8sAction:        "create",
		SuccessCondition: spark.SuccessCondition,
		FailureCondition: spark.FailureCondition,
		WorkloadType:     workloadTypeData,
	}
	return []*WorkloadSpec{workloadSpec}, nil
}
