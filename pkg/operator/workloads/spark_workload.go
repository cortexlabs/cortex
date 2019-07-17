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

	"github.com/cortexlabs/cortex/pkg/consts"
	"github.com/cortexlabs/cortex/pkg/lib/errors"
	"github.com/cortexlabs/cortex/pkg/lib/pointer"
	"github.com/cortexlabs/cortex/pkg/lib/sets/strset"
	"github.com/cortexlabs/cortex/pkg/lib/spark"
	s "github.com/cortexlabs/cortex/pkg/lib/strings"
	"github.com/cortexlabs/cortex/pkg/operator/api/context"
	"github.com/cortexlabs/cortex/pkg/operator/api/userconfig"
	"github.com/cortexlabs/cortex/pkg/operator/config"
)

type SparkWorkload struct {
	BaseWorkload
}

func sparkResources(ctx *context.Context) []context.ComputedResource {
	var sparkResources []context.ComputedResource
	for _, rawColumn := range ctx.RawColumns {
		sparkResources = append(sparkResources, rawColumn)
	}
	for _, aggregate := range ctx.Aggregates {
		sparkResources = append(sparkResources, aggregate)
	}
	for _, transformedColumn := range ctx.TransformedColumns {
		sparkResources = append(sparkResources, transformedColumn)
	}
	for _, model := range ctx.Models {
		sparkResources = append(sparkResources, model.Dataset)
	}
	return sparkResources
}

func populateSparkWorkloadIDs(ctx *context.Context, latestResourceWorkloadIDs map[string]string) {
	sparkWorkloadID := generateWorkloadID()
	for _, res := range sparkResources(ctx) {
		if res.GetWorkloadID() != "" {
			continue
		}
		if workloadID := latestResourceWorkloadIDs[res.GetID()]; workloadID != "" {
			res.SetWorkloadID(workloadID)
			continue
		}
		res.SetWorkloadID(sparkWorkloadID)
	}
}

func extractSparkWorkloads(ctx *context.Context) []Workload {
	workloadMap := make(map[string]*SparkWorkload)

	for _, res := range sparkResources(ctx) {
		if _, ok := workloadMap[res.GetWorkloadID()]; !ok {
			workloadMap[res.GetWorkloadID()] = &SparkWorkload{
				emptyBaseWorkload(ctx.App.Name, res.GetWorkloadID(), workloadTypeSpark),
			}
		}
		workloadMap[res.GetWorkloadID()].AddResource(res)
	}

	workloads := make([]Workload, 0, len(workloadMap))
	for _, workload := range workloadMap {
		workloads = append(workloads, workload)
	}
	return workloads
}

func (sw *SparkWorkload) Start(ctx *context.Context) error {
	rawDatasetExists, err := config.AWS.IsS3File(filepath.Join(ctx.RawDataset.Key, "_SUCCESS"))
	if err != nil {
		return errors.Wrap(err, ctx.App.Name, "raw dataset")
	}
	shouldIngest := !rawDatasetExists

	rawColumns := strset.New()
	aggregates := strset.New()
	transformedColumns := strset.New()
	trainingDatasets := strset.New()

	var sparkCompute *userconfig.SparkCompute

	if shouldIngest {
		for _, rawColumn := range ctx.RawColumns {
			sparkCompute = userconfig.MaxSparkCompute(sparkCompute, rawColumn.GetCompute())
		}
	}

	for _, rawColumn := range ctx.RawColumns {
		if sw.CreatesResource(rawColumn.GetID()) {
			rawColumns.Add(rawColumn.GetID())
			sparkCompute = userconfig.MaxSparkCompute(sparkCompute, rawColumn.GetCompute())
		}
	}
	for _, aggregate := range ctx.Aggregates {
		if sw.CreatesResource(aggregate.ID) {
			aggregates.Add(aggregate.ID)
			sparkCompute = userconfig.MaxSparkCompute(sparkCompute, aggregate.Compute)
		}
	}
	for _, transformedColumn := range ctx.TransformedColumns {
		if sw.CreatesResource(transformedColumn.ID) {
			transformedColumns.Add(transformedColumn.ID)
			sparkCompute = userconfig.MaxSparkCompute(sparkCompute, transformedColumn.Compute)
		}
	}
	for _, model := range ctx.Models {
		dataset := model.Dataset
		if sw.CreatesResource(dataset.ID) {
			trainingDatasets.Add(dataset.ID)
			sparkCompute = userconfig.MaxSparkCompute(sparkCompute, model.DatasetCompute)

			dependencyIDs := ctx.AllComputedResourceDependencies(dataset.ID)
			for _, transformedColumn := range ctx.TransformedColumns {
				if _, ok := dependencyIDs[transformedColumn.ID]; ok {
					sparkCompute = userconfig.MaxSparkCompute(sparkCompute, transformedColumn.Compute)
				}
			}
		}
	}

	args := []string{
		"--raw-columns=" + strings.Join(rawColumns.Slice(), ","),
		"--aggregates=" + strings.Join(aggregates.Slice(), ","),
		"--transformed-columns=" + strings.Join(transformedColumns.Slice(), ","),
		"--training-datasets=" + strings.Join(trainingDatasets.Slice(), ","),
	}
	if shouldIngest {
		args = append(args, "--ingest")
	}

	spec := sparkSpec(sw.WorkloadID, ctx, workloadTypeSpark, sparkCompute, args...)
	_, err = config.Spark.Create(spec)
	if err != nil {
		return err
	}

	return nil
}

func sparkSpec(
	workloadID string,
	ctx *context.Context,
	workloadType string,
	sparkCompute *userconfig.SparkCompute,
	args ...string,
) *sparkop.SparkApplication {

	var driverMemOverhead *string
	if sparkCompute.DriverMemOverhead != nil {
		driverMemOverhead = pointer.String(s.Int64(sparkCompute.DriverMemOverhead.ToKi()) + "k")
	}
	var executorMemOverhead *string
	if sparkCompute.ExecutorMemOverhead != nil {
		executorMemOverhead = pointer.String(s.Int64(sparkCompute.ExecutorMemOverhead.ToKi()) + "k")
	}
	var memOverheadFactor *string
	if sparkCompute.MemOverheadFactor != nil {
		memOverheadFactor = pointer.String(s.Float64(*sparkCompute.MemOverheadFactor))
	}

	return spark.App(&spark.Spec{
		Name:      workloadID,
		Namespace: config.Cortex.Namespace,
		Labels: map[string]string{
			"workloadID":   workloadID,
			"workloadType": workloadType,
			"appName":      ctx.App.Name,
		},
		Spec: sparkop.SparkApplicationSpec{
			Type:                 sparkop.PythonApplicationType,
			PythonVersion:        pointer.String("3"),
			Mode:                 sparkop.ClusterMode,
			Image:                &config.Cortex.SparkImage,
			ImagePullPolicy:      pointer.String("Always"),
			MainApplicationFile:  pointer.String("local:///src/cortex/spark_job/spark_job.py"),
			RestartPolicy:        sparkop.RestartPolicy{Type: sparkop.Never},
			MemoryOverheadFactor: memOverheadFactor,
			Arguments: []string{
				strings.TrimSpace(
					" --workload-id=" + workloadID +
						" --context=" + config.AWS.S3Path(ctx.Key) +
						" --cache-dir=" + consts.ContextCacheDir +
						" " + strings.Join(args, " ")),
			},
			Deps: sparkop.Dependencies{
				PyFiles: []string{"local:///src/cortex/spark_job/spark_util.py", "local:///src/cortex/lib/*.py"},
			},
			Driver: sparkop.DriverSpec{
				SparkPodSpec: sparkop.SparkPodSpec{
					Cores:          pointer.Float32(sparkCompute.DriverCPU.ToFloat32()),
					Memory:         pointer.String(s.Int64(sparkCompute.DriverMem.ToKi()) + "k"),
					MemoryOverhead: driverMemOverhead,
					Labels: map[string]string{
						"workloadID":   workloadID,
						"workloadType": workloadType,
						"appName":      ctx.App.Name,
						"userFacing":   "true",
					},
					EnvSecretKeyRefs: map[string]sparkop.NameKey{
						"AWS_ACCESS_KEY_ID": {
							Name: "aws-credentials",
							Key:  "AWS_ACCESS_KEY_ID",
						},
						"AWS_SECRET_ACCESS_KEY": {
							Name: "aws-credentials",
							Key:  "AWS_SECRET_ACCESS_KEY",
						},
					},
					EnvVars: map[string]string{
						"CORTEX_SPARK_VERBOSITY": ctx.Environment.LogLevel.Spark,
						"CORTEX_CONTEXT_S3_PATH": config.AWS.S3Path(ctx.Key),
						"CORTEX_WORKLOAD_ID":     workloadID,
						"CORTEX_CACHE_DIR":       consts.ContextCacheDir,
					},
				},
				PodName:        &workloadID,
				ServiceAccount: pointer.String("spark"),
			},
			Executor: sparkop.ExecutorSpec{
				SparkPodSpec: sparkop.SparkPodSpec{
					Cores:          pointer.Float32(sparkCompute.ExecutorCPU.ToFloat32()),
					Memory:         pointer.String(s.Int64(sparkCompute.ExecutorMem.ToKi()) + "k"),
					MemoryOverhead: executorMemOverhead,
					Labels: map[string]string{
						"workloadID":   workloadID,
						"workloadType": workloadType,
						"appName":      ctx.App.Name,
					},
					EnvSecretKeyRefs: map[string]sparkop.NameKey{
						"AWS_ACCESS_KEY_ID": {
							Name: "aws-credentials",
							Key:  "AWS_ACCESS_KEY_ID",
						},
						"AWS_SECRET_ACCESS_KEY": {
							Name: "aws-credentials",
							Key:  "AWS_SECRET_ACCESS_KEY",
						},
					},
					EnvVars: map[string]string{
						"CORTEX_SPARK_VERBOSITY": ctx.Environment.LogLevel.Spark,
						"CORTEX_CONTEXT_S3_PATH": config.AWS.S3Path(ctx.Key),
						"CORTEX_WORKLOAD_ID":     workloadID,
						"CORTEX_CACHE_DIR":       consts.ContextCacheDir,
					},
				},
				Instances: &sparkCompute.Executors,
			},
		},
	})
}

func (sw *SparkWorkload) IsRunning(ctx *context.Context) (bool, error) {
	return config.Spark.IsRunning(sw.WorkloadID)
}

func (sw *SparkWorkload) CanRun(ctx *context.Context) (bool, error) {
	return areDataDependenciesSucceeded(ctx, sw.GetResourceIDs())
}

func (sw *SparkWorkload) IsSucceeded(ctx *context.Context) (bool, error) {
	return areDataResourcesSucceeded(ctx, sw.GetResourceIDs())
}
