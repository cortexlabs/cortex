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

	corev1 "k8s.io/api/core/v1"

	"github.com/cortexlabs/cortex/pkg/api/context"
	"github.com/cortexlabs/cortex/pkg/api/resource"
	"github.com/cortexlabs/cortex/pkg/api/userconfig"
	"github.com/cortexlabs/cortex/pkg/operator/k8s"
	"github.com/cortexlabs/cortex/pkg/utils/errors"
	"github.com/cortexlabs/cortex/pkg/utils/sets/strset"
)

func GetCurrentAPIStatuses(
	ctx *context.Context, dataStatuses map[string]*resource.DataStatus,
) (map[string]*resource.APIStatus, error) {

	failedWorkloadIDs, err := getFailedArgoWorkloadIDs(ctx.App.Name)
	if err != nil {
		return nil, err
	}

	podList, err := k8s.ListPodsByLabels(map[string]string{
		"workloadType": WorkloadTypeAPI,
		"appName":      ctx.App.Name,
		"userFacing":   "true",
	})
	if err != nil {
		return nil, errors.Wrap(err, "api statuses", ctx.App.Name)
	}

	replicaCountsMap := getReplicaCountsMap(podList, ctx)

	currentResourceWorkloadIDs := ctx.APIResourceWorkloadIDs()

	savedStatuses, err := calculateAPISavedStatuses(podList, ctx.App.Name)
	if err != nil {
		return nil, err
	}

	apiStatuses := make(map[string]*resource.APIStatus)
	for _, savedStatus := range savedStatuses {
		// This handles the rare case when there are multiple workloads for a single resource
		if apiStatuses[savedStatus.ResourceID] != nil {
			if savedStatus.WorkloadID != currentResourceWorkloadIDs[savedStatus.ResourceID] {
				continue
			}
		}

		apiStatuses[savedStatus.ResourceID] = &resource.APIStatus{
			APISavedStatus: *savedStatus,
		}
	}

	currentAPIResourceIDs := strset.New()
	for _, api := range ctx.APIs {
		resourceID := api.ID
		if apiStatuses[resourceID] == nil || apiStatuses[resourceID].WorkloadID != api.WorkloadID {
			apiStatuses[resourceID] = &resource.APIStatus{
				APISavedStatus: resource.APISavedStatus{
					BaseSavedStatus: resource.BaseSavedStatus{
						ResourceID:   resourceID,
						ResourceType: resource.APIType,
						WorkloadID:   api.WorkloadID,
						AppName:      ctx.App.Name,
					},
					APIName: api.Name,
				},
			}
		}
		apiStatuses[resourceID].RequestedReplicas = api.Compute.Replicas
		currentAPIResourceIDs.Add(resourceID)
	}

	for resourceID, apiStatus := range apiStatuses {
		apiStatus.Path = context.APIPath(apiStatus.APIName, apiStatus.AppName)
		apiStatus.ReplicaCounts = replicaCountsMap[resourceID]
		apiStatus.Code = apiStatusCode(apiStatus, failedWorkloadIDs)
	}

	for _, apiStatus := range apiStatuses {
		if currentAPIResourceIDs.Has(apiStatus.ResourceID) {
			updateAPIStatusCodeByParents(apiStatus, dataStatuses, ctx)
		}
	}

	setInsufficientComputeAPIStatusCodes(apiStatuses, ctx)

	return apiStatuses, nil
}

func getReplicaCountsMap(podList []corev1.Pod, ctx *context.Context) map[string]resource.ReplicaCounts {
	replicaCountsMap := make(map[string]resource.ReplicaCounts)

	ctxAPIComputeIDMap := make(map[string]string)
	for _, api := range ctx.APIs {
		ctxAPIComputeIDMap[api.ID] = api.Compute.IDWithoutReplicas()
	}

	for _, pod := range podList {
		resourceID := pod.Labels["resourceID"]
		cpu, mem, gpu := APIPodCompute(pod.Spec.Containers)
		podAPICompute := userconfig.APICompute{
			CPU: cpu,
			Mem: mem,
			GPU: gpu,
		}
		podAPIComputeID := podAPICompute.IDWithoutReplicas()
		podStatus := k8s.GetPodStatus(&pod)

		replicaCounts := replicaCountsMap[resourceID]

		ctxAPIComputeID, isAPIInCtx := ctxAPIComputeIDMap[resourceID]
		computeMatches := false
		if isAPIInCtx && ctxAPIComputeID == podAPIComputeID {
			computeMatches = true
		}

		if podStatus == k8s.PodStatusRunning {
			switch {
			case isAPIInCtx && computeMatches:
				replicaCounts.ReadyUpdated++
			case isAPIInCtx && !computeMatches:
				replicaCounts.ReadyStaleCompute++
			case !isAPIInCtx:
				replicaCounts.ReadyStaleResource++
			}
		}
		if podStatus == k8s.PodStatusFailed {
			switch {
			case isAPIInCtx && computeMatches:
				replicaCounts.FailedUpdated++
			case isAPIInCtx && !computeMatches:
				replicaCounts.FailedStaleCompute++
			case !isAPIInCtx:
				replicaCounts.FailedStaleResource++
			}
		}

		replicaCountsMap[resourceID] = replicaCounts
	}

	return replicaCountsMap
}

func apiStatusCode(apiStatus *resource.APIStatus, failedWorkloadIDs strset.Set) resource.StatusCode {
	if failedWorkloadIDs.Has(apiStatus.WorkloadID) {
		return resource.StatusAPIError
	}
	if apiStatus.RequestedReplicas == 0 {
		if apiStatus.TotalReady() > 0 {
			return resource.StatusAPIStopping
		}
		return resource.StatusAPIStopped
	}
	if apiStatus.ReadyUpdated == apiStatus.RequestedReplicas {
		return resource.StatusAPIReady
	}
	if apiStatus.FailedUpdated > 0 {
		return resource.StatusAPIError
	}
	if apiStatus.TotalReady() > 0 {
		return resource.StatusAPIUpdating
	}

	return resource.StatusPending
}

func updateAPIStatusCodeByParents(apiStatus *resource.APIStatus, dataStatuses map[string]*resource.DataStatus, ctx *context.Context) {
	if apiStatus.Code != resource.StatusPending {
		return
	}

	allDependencies := ctx.AllComputedResourceDependencies(apiStatus.ResourceID)
	numSucceeded := 0
	parentSkipped := false
	for dependency := range allDependencies {
		switch dataStatuses[dependency].Code {
		case resource.StatusDataKilled:
			apiStatus.Code = resource.StatusParentKilled
			return
		case resource.StatusDataFailed:
			apiStatus.Code = resource.StatusParentFailed
			return
		case resource.StatusSkipped:
			parentSkipped = true
		case resource.StatusDataSucceeded:
			numSucceeded++
		}
	}

	if parentSkipped {
		apiStatus.Code = resource.StatusSkipped
		return
	}

	if numSucceeded == len(allDependencies) {
		apiStatus.Code = resource.StatusAPIUpdating
	}
}

func GetAPIGroupStatuses(apiStatuses map[string]*resource.APIStatus, ctx *context.Context) (map[string]*resource.APIGroupStatus, error) {
	apiGroupStatuses := make(map[string]*resource.APIGroupStatus)

	statusMap := make(map[string][]*resource.APIStatus)
	for _, apiStatus := range apiStatuses {
		apiName := apiStatus.APIName
		statusMap[apiName] = append(statusMap[apiName], apiStatus)
	}

	deployments, err := deploymentMap(ctx.App.Name)
	if err != nil {
		return nil, errors.Wrap(err, "api group statuses")
	}

	for apiName, apiStatuses := range statusMap {
		apiGroupStatuses[apiName] = &resource.APIGroupStatus{
			APIName:      apiName,
			Start:        k8s.DeploymentStartTime(deployments[apiName]),
			ActiveStatus: getActiveAPIStatus(apiStatuses, ctx),
			Code:         apiGroupStatusCode(apiStatuses, ctx),
		}
	}

	return apiGroupStatuses, nil
}

// Get the most recently ready API status, or the ctx API status if it is ready
func getActiveAPIStatus(apiStatuses []*resource.APIStatus, ctx *context.Context) *resource.APIStatus {
	if len(apiStatuses) == 0 {
		return nil
	}

	apiName := apiStatuses[0].APIName
	ctxAPIID := ""
	if ctxAPI := ctx.APIs[apiName]; ctxAPI != nil {
		ctxAPIID = ctxAPI.ID
	}

	var latestTime time.Time
	var latestAPIStatus *resource.APIStatus = nil

	for _, apiStatus := range apiStatuses {
		if apiStatus.TotalReady() == 0 {
			continue
		}
		if apiStatus.ResourceID == ctxAPIID {
			return apiStatus
		}
		if apiStatus.Start != nil && (*apiStatus.Start).After(latestTime) {
			latestAPIStatus = apiStatus
			latestTime = *apiStatus.Start
		}
	}
	return latestAPIStatus
}

func apiGroupStatusCode(apiStatuses []*resource.APIStatus, ctx *context.Context) resource.StatusCode {
	if len(apiStatuses) == 0 {
		return resource.StatusAPIStopped
	}

	apiName := apiStatuses[0].APIName
	ctxAPI := ctx.APIs[apiName]

	if ctxAPI == nil {
		for _, apiStatus := range apiStatuses {
			if apiStatus.TotalReady() > 0 {
				return resource.StatusAPIStopping
			}
		}
		return resource.StatusAPIStopped
	}

	var ctxAPIStatus *resource.APIStatus
	nonCtxPodsReady := false
	for _, apiStatus := range apiStatuses {
		if apiStatus.ResourceID == ctxAPI.ID {
			ctxAPIStatus = apiStatus
		} else if apiStatus.TotalReady() > 0 {
			nonCtxPodsReady = true
		}
	}

	if !nonCtxPodsReady {
		return ctxAPIStatus.Code
	}

	switch ctxAPIStatus.Code {
	case resource.StatusUnknown:
		return resource.StatusUnknown
	case resource.StatusPending:
		return resource.StatusAPIGroupPendingUpdate
	case resource.StatusPendingCompute:
		return resource.StatusPendingCompute
	case resource.StatusWaiting:
		return resource.StatusAPIUpdating
	case resource.StatusSkipped:
		return resource.StatusAPIGroupUpdateSkipped
	case resource.StatusParentFailed:
		return resource.StatusAPIGroupParentFailed
	case resource.StatusParentKilled:
		return resource.StatusAPIGroupParentKilled
	case resource.StatusAPIUpdating:
		return resource.StatusAPIUpdating
	case resource.StatusAPIReady:
		return resource.StatusAPIReady
	case resource.StatusAPIStopping:
		return resource.StatusAPIStopping
	case resource.StatusAPIStopped:
		return resource.StatusAPIGroupPendingUpdate
	case resource.StatusAPIError:
		return resource.StatusAPIError
	}

	return resource.StatusUnknown
}

func setInsufficientComputeAPIStatusCodes(apiStatuses map[string]*resource.APIStatus, ctx *context.Context) error {
	stalledPods, err := k8s.StalledPods()
	if err != nil {
		return err
	}
	stalledWorkloads := strset.New()
	for _, pod := range stalledPods {
		stalledWorkloads.Add(pod.Labels["workloadID"])
	}

	for _, apiStatus := range apiStatuses {
		if apiStatus.Code == resource.StatusPending || apiStatus.Code == resource.StatusWaiting || apiStatus.Code == resource.StatusAPIUpdating {
			if _, ok := stalledWorkloads[apiStatus.WorkloadID]; ok {
				apiStatus.Code = resource.StatusPendingCompute
			}
		}
	}

	return nil
}
