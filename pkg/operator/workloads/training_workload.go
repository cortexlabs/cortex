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
	kcore "k8s.io/api/core/v1"
	kresource "k8s.io/apimachinery/pkg/api/resource"

	"github.com/cortexlabs/cortex/pkg/consts"
	"github.com/cortexlabs/cortex/pkg/lib/k8s"
	"github.com/cortexlabs/cortex/pkg/lib/sets/strset"
	"github.com/cortexlabs/cortex/pkg/operator/api/context"
	"github.com/cortexlabs/cortex/pkg/operator/api/userconfig"
	"github.com/cortexlabs/cortex/pkg/operator/config"
)

type TrainingWorkload struct {
	BaseWorkload
}

func populateTrainingWorkloadIDs(ctx *context.Context, latestResourceWorkloadIDs map[string]string) {
	trainingWorkloadIDs := make(map[string]string)

	for _, model := range ctx.Models {
		if model.WorkloadID != "" {
			continue
		}
		if workloadID := latestResourceWorkloadIDs[model.ID]; workloadID != "" {
			model.WorkloadID = workloadID
			continue
		}
		if workloadID, ok := trainingWorkloadIDs[model.ID]; ok {
			// This is a duplicate model ID (different name)
			model.WorkloadID = workloadID
			continue
		}
		model.WorkloadID = generateWorkloadID()
		trainingWorkloadIDs[model.ID] = model.WorkloadID
	}
}

func extractTrainingWorkloads(ctx *context.Context) []Workload {
	workloads := make([]Workload, 0, len(ctx.Models))
	modelIDs := strset.New()

	for _, model := range ctx.Models {
		if !modelIDs.Has(model.ID) {
			workloads = append(workloads, &TrainingWorkload{
				singleBaseWorkload(model, ctx.App.Name, workloadTypeTrain),
			})
			modelIDs.Add(model.ID)
		}
	}

	return workloads
}

func (tw *TrainingWorkload) Start(ctx *context.Context) error {
	var tfCompute *userconfig.TFCompute
	for _, model := range ctx.Models {
		if tw.CreatesResource(model.ID) {
			tfCompute = userconfig.MaxTFCompute(tfCompute, model.Compute)
		}
	}

	resourceList := kcore.ResourceList{}
	limitsList := kcore.ResourceList{}
	resourceList[kcore.ResourceCPU] = tfCompute.CPU.Quantity
	if tfCompute.Mem != nil {
		resourceList[kcore.ResourceMemory] = tfCompute.Mem.Quantity
	}

	trainImage := config.Cortex.TFTrainImage
	if tfCompute.GPU > 0 {
		trainImage = config.Cortex.TFTrainImageGPU
		resourceList["nvidia.com/gpu"] = *kresource.NewQuantity(tfCompute.GPU, kresource.DecimalSI)
		limitsList["nvidia.com/gpu"] = *kresource.NewQuantity(tfCompute.GPU, kresource.DecimalSI)
	}

	spec := &k8s.JobSpec{
		Name: tw.WorkloadID,
		Labels: map[string]string{
			"appName":      ctx.App.Name,
			"workloadType": workloadTypeTrain,
			"workloadID":   tw.WorkloadID,
		},
		PodSpec: k8s.PodSpec{
			Labels: map[string]string{
				"appName":      ctx.App.Name,
				"workloadType": workloadTypeTrain,
				"workloadID":   tw.WorkloadID,
				"userFacing":   "true",
			},
			K8sPodSpec: kcore.PodSpec{
				RestartPolicy: "Never",
				Containers: []kcore.Container{
					{
						Name:            "train",
						Image:           trainImage,
						ImagePullPolicy: kcore.PullAlways,
						Args: []string{
							"--workload-id=" + tw.WorkloadID,
							"--context=" + config.AWS.S3Path(ctx.Key),
							"--cache-dir=" + consts.ContextCacheDir,
							"--model=" + tw.GetSingleResourceID(),
						},
						Env:          k8s.AWSCredentials(),
						VolumeMounts: k8s.DefaultVolumeMounts(),
						Resources: kcore.ResourceRequirements{
							Requests: resourceList,
							Limits:   limitsList,
						},
					},
				},
				Volumes:            k8s.DefaultVolumes(),
				ServiceAccountName: "default",
			},
		},
		Namespace: config.Cortex.Namespace,
	}

	_, err := config.Kubernetes.CreateJob(k8s.Job(spec))
	if err != nil {
		return err
	}
	return nil
}

func (tw *TrainingWorkload) IsStarted(ctx *context.Context) (bool, error) {
	return config.Kubernetes.JobExists(tw.WorkloadID)
}

func (tw *TrainingWorkload) IsRunning(ctx *context.Context) (bool, error) {
	return config.Kubernetes.IsJobRunning(tw.WorkloadID)
}

func (tw *TrainingWorkload) CanRun(ctx *context.Context) (bool, error) {
	return areAllDataDependenciesSucceeded(ctx, tw.GetResourceIDs())
}

func (tw *TrainingWorkload) IsSucceeded(ctx *context.Context) (bool, error) {
	return areAllDataResourcesSucceeded(ctx, tw.GetResourceIDs())
}

func (tw *TrainingWorkload) IsFailed(ctx *context.Context) (bool, error) {
	return areAnyDataResourcesFailed(ctx, tw.GetResourceIDs())
}
