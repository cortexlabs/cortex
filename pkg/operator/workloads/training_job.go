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
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"

	"github.com/cortexlabs/cortex/pkg/api/context"
	"github.com/cortexlabs/cortex/pkg/api/userconfig"
	"github.com/cortexlabs/cortex/pkg/consts"
	"github.com/cortexlabs/cortex/pkg/operator/argo"
	"github.com/cortexlabs/cortex/pkg/operator/aws"
	cc "github.com/cortexlabs/cortex/pkg/operator/cortexconfig"
	"github.com/cortexlabs/cortex/pkg/operator/k8s"
	"github.com/cortexlabs/cortex/pkg/utils/sets/strset"
)

func trainingJobSpec(
	ctx *context.Context,
	modelID string,
	workloadID string,
	tfCompute *userconfig.TFCompute,
) *batchv1.Job {

	resourceList := corev1.ResourceList{}
	if tfCompute.CPU != nil {
		resourceList[corev1.ResourceCPU] = tfCompute.CPU.Quantity
	}
	if tfCompute.Mem != nil {
		resourceList[corev1.ResourceMemory] = tfCompute.Mem.Quantity
	}

	spec := k8s.Job(&k8s.JobSpec{
		Name: workloadID,
		Labels: map[string]string{
			"appName":      ctx.App.Name,
			"workloadType": workloadTypeTrain,
			"workloadID":   workloadID,
		},
		PodSpec: k8s.PodSpec{
			Labels: map[string]string{
				"appName":      ctx.App.Name,
				"workloadType": workloadTypeTrain,
				"workloadID":   workloadID,
				"userFacing":   "true",
			},
			K8sPodSpec: corev1.PodSpec{
				RestartPolicy: "Never",
				Containers: []corev1.Container{
					{
						Name:            "train",
						Image:           cc.TFTrainImage,
						ImagePullPolicy: "Always",
						Args: []string{
							"--workload-id=" + workloadID,
							"--context=" + aws.S3Path(ctx.Key),
							"--cache-dir=" + consts.ContextCacheDir,
							"--model=" + modelID,
						},
						Env:          k8s.AWSCredentials(),
						VolumeMounts: k8s.DefaultVolumeMounts(),
						Resources: corev1.ResourceRequirements{
							Requests: resourceList,
						},
					},
				},
				Volumes:            k8s.DefaultVolumes(),
				ServiceAccountName: "default",
			},
		},
		Namespace: cc.Namespace,
	})
	argo.EnableGC(spec)
	return spec
}

func trainingWorkloadSpecs(ctx *context.Context) ([]*WorkloadSpec, error) {
	modelsToTrain := make(map[string]*userconfig.TFCompute)
	for _, model := range ctx.Models {
		modelCached, err := checkResourceCached(model, ctx)
		if err != nil {
			return nil, err
		}
		if modelCached {
			continue
		}

		if tfCompute, ok := modelsToTrain[model.ID]; ok {
			modelsToTrain[model.ID] = userconfig.MaxTFCompute(tfCompute, model.Compute)
		} else {
			modelsToTrain[model.ID] = model.Compute
		}
	}

	var workloadSpecs []*WorkloadSpec
	for modelID, tfCompute := range modelsToTrain {
		workloadID := generateWorkloadID()
		workloadSpecs = append(workloadSpecs, &WorkloadSpec{
			WorkloadID:       workloadID,
			ResourceIDs:      strset.New(modelID),
			Spec:             trainingJobSpec(ctx, modelID, workloadID, tfCompute),
			K8sAction:        "create",
			SuccessCondition: k8s.JobSuccessCondition,
			FailureCondition: k8s.JobFailureCondition,
			WorkloadType:     workloadTypeTrain,
		})
	}

	return workloadSpecs, nil
}
