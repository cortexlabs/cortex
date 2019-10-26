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
	"time"

	kapps "k8s.io/api/apps/v1"
	kautoscaling "k8s.io/api/autoscaling/v2beta2"
	kcore "k8s.io/api/core/v1"

	"github.com/cortexlabs/cortex/pkg/consts"
	"github.com/cortexlabs/cortex/pkg/lib/k8s"
	"github.com/cortexlabs/cortex/pkg/operator/api/context"
	"github.com/cortexlabs/cortex/pkg/operator/config"
)

type HPAWorkload struct {
	BaseWorkload
	APIID string
}

func extractHPAWorkloads(ctx *context.Context) []Workload {
	workloads := make([]Workload, 0, len(ctx.APIs))

	for _, api := range ctx.APIs {
		workloads = append(workloads, &HPAWorkload{
			BaseWorkload: emptyBaseWorkload(ctx.App.Name, api.WorkloadID, workloadTypeHPA), // HPA doesn't produce any resources
			APIID:        api.ID,
		})
	}

	return workloads
}

func (hw *HPAWorkload) Start(ctx *context.Context) error {
	api := ctx.APIs.OneByID(hw.APIID)

	_, err := config.Kubernetes.ApplyHPA(hpaSpec(ctx, api))
	if err != nil {
		return err
	}

	return nil
}

func (hw *HPAWorkload) IsSucceeded(ctx *context.Context) (bool, error) {
	api := ctx.APIs.OneByID(hw.APIID)
	k8sDeloymentName := internalAPIName(api.Name, ctx.App.Name)

	hpa, err := config.Kubernetes.GetHPA(k8sDeloymentName)
	if err != nil {
		return false, err
	}

	return k8s.IsHPAUpToDate(hpa, api.Compute.MinReplicas, api.Compute.MaxReplicas, api.Compute.TargetCPUUtilization), nil
}

func (hw *HPAWorkload) IsRunning(ctx *context.Context) (bool, error) {
	return false, nil
}

func (hw *HPAWorkload) IsStarted(ctx *context.Context) (bool, error) {
	return hw.IsSucceeded(ctx)
}

func (hw *HPAWorkload) CanRun(ctx *context.Context) (bool, error) {
	api := ctx.APIs.OneByID(hw.APIID)
	k8sDeloymentName := internalAPIName(api.Name, ctx.App.Name)

	k8sDeployment, err := config.Kubernetes.GetDeployment(k8sDeloymentName)
	if err != nil {
		return false, err
	}
	if k8sDeployment == nil || k8sDeployment.Labels["resourceID"] != api.ID || k8sDeployment.DeletionTimestamp != nil {
		return false, nil
	}

	if doesAPIComputeNeedsUpdating(api, k8sDeployment) {
		return false, nil
	}

	updatedReplicas, err := numUpdatedReadyReplicas(ctx, api)
	if err != nil {
		return false, err
	}
	requestedReplicas := getRequestedReplicasFromDeployment(api, k8sDeployment, nil)
	if updatedReplicas < requestedReplicas {
		return false, nil
	}

	for _, condition := range k8sDeployment.Status.Conditions {
		if condition.Type == kapps.DeploymentProgressing &&
			condition.Status == kcore.ConditionTrue &&
			!condition.LastUpdateTime.IsZero() &&
			time.Now().After(condition.LastUpdateTime.Add(35*time.Second)) { // the metrics poll interval is 30 seconds, so 35 should be safe
			return true, nil
		}
	}

	return false, nil
}

func (hw *HPAWorkload) IsFailed(ctx *context.Context) (bool, error) {
	return false, nil
}

func hpaSpec(ctx *context.Context, api *context.API) *kautoscaling.HorizontalPodAutoscaler {
	return k8s.HPA(&k8s.HPASpec{
		DeploymentName:       internalAPIName(api.Name, ctx.App.Name),
		MinReplicas:          api.Compute.MinReplicas,
		MaxReplicas:          api.Compute.MaxReplicas,
		TargetCPUUtilization: api.Compute.TargetCPUUtilization,
		Labels: map[string]string{
			"appName":      ctx.App.Name,
			"workloadType": workloadTypeAPI,
			"apiName":      api.Name,
		},
		Namespace: consts.K8sNamespace,
	})
}
