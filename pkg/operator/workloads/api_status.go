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
	kcore "k8s.io/api/core/v1"

	"github.com/cortexlabs/cortex/pkg/lib/errors"
	"github.com/cortexlabs/cortex/pkg/lib/k8s"
	"github.com/cortexlabs/cortex/pkg/lib/sets/strset"
	"github.com/cortexlabs/cortex/pkg/operator/api/context"
	"github.com/cortexlabs/cortex/pkg/operator/api/resource"
	"github.com/cortexlabs/cortex/pkg/operator/config"
)

func GetCurrentAPIAndGroupStatuses(
	dataStatuses map[string]*resource.DataStatus,
	ctx *context.Context,
) (map[string]*resource.APIStatus, map[string]*resource.APIGroupStatus, error) {
	deployments, err := apiDeploymentMap(ctx.App.Name)
	if err != nil {
		return nil, nil, err
	}

	apiStatuses, err := getCurrentAPIStatuses(dataStatuses, deployments, ctx)
	if err != nil {
		return nil, nil, err
	}

	apiGroupStatuses, err := getAPIGroupStatuses(apiStatuses, deployments, ctx)
	if err != nil {
		return nil, nil, err
	}

	return apiStatuses, apiGroupStatuses, nil
}

func getCurrentAPIStatuses(
	dataStatuses map[string]*resource.DataStatus,
	deployments map[string]*kapps.Deployment, // api.Name -> deployment
	ctx *context.Context,
) (map[string]*resource.APIStatus, error) {

	podList, err := config.Kubernetes.ListPodsByLabels(map[string]string{
		"workloadType": workloadTypeAPI,
		"appName":      ctx.App.Name,
		"userFacing":   "true",
	})
	if err != nil {
		return nil, errors.Wrap(err, "api statuses", ctx.App.Name)
	}

	replicaCountsMap := getReplicaCountsMap(podList, deployments, ctx)

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
		apiStatuses[resourceID].MinReplicas = api.Compute.MinReplicas
		apiStatuses[resourceID].MaxReplicas = api.Compute.MaxReplicas
		apiStatuses[resourceID].InitReplicas = api.Compute.InitReplicas
		apiStatuses[resourceID].TargetCPUUtilization = api.Compute.TargetCPUUtilization
		currentAPIResourceIDs.Add(resourceID)
	}

	for resourceID, apiStatus := range apiStatuses {
		apiStatus.Path = context.APIPath(apiStatus.APIName, apiStatus.AppName)
		apiStatus.ReplicaCounts = replicaCountsMap[resourceID]
		apiStatus.Code = apiStatusCode(apiStatus)
	}

	for _, apiStatus := range apiStatuses {
		if currentAPIResourceIDs.Has(apiStatus.ResourceID) {
			updateAPIStatusCodeByParents(apiStatus, dataStatuses, ctx)
		}
	}

	setInsufficientComputeAPIStatusCodes(apiStatuses, ctx)

	return apiStatuses, nil
}

func getReplicaCountsMap(
	podList []kcore.Pod,
	deployments map[string]*kapps.Deployment, // api.Name -> deployment
	ctx *context.Context,
) map[string]resource.ReplicaCounts {

	apiComputeIDMap := make(map[string]string)
	for _, api := range ctx.APIs {
		apiComputeIDMap[api.ID] = api.Compute.IDWithoutReplicas()
	}
	for _, deployment := range deployments {
		resourceID := deployment.Labels["resourceID"]
		if _, ok := apiComputeIDMap[resourceID]; !ok {
			apiComputeIDMap[resourceID] = APIPodComputeID(deployment.Spec.Template.Spec.Containers)
		}
	}

	replicaCountsMap := make(map[string]resource.ReplicaCounts)
	for _, pod := range podList {
		resourceID := pod.Labels["resourceID"]
		podAPIComputeID := APIPodComputeID(pod.Spec.Containers)
		podStatus := k8s.GetPodStatus(&pod)

		replicaCounts := replicaCountsMap[resourceID]

		computeMatches := false
		ctxAPIComputeID, ok := apiComputeIDMap[resourceID]
		if ok && ctxAPIComputeID == podAPIComputeID {
			computeMatches = true
		}

		if podStatus == k8s.PodStatusRunning {
			if computeMatches {
				replicaCounts.ReadyUpdatedCompute++
			} else {
				replicaCounts.ReadyStaleCompute++
			}
		}
		if podStatus == k8s.PodStatusFailed {
			if computeMatches {
				replicaCounts.FailedUpdatedCompute++
			} else {
				replicaCounts.FailedStaleCompute++
			}
		}

		replicaCountsMap[resourceID] = replicaCounts
	}

	for _, deployment := range deployments {
		if deployment.Spec.Replicas == nil {
			continue
		}
		resourceID := deployment.Labels["resourceID"]
		replicaCounts := replicaCountsMap[resourceID]
		replicaCounts.K8sRequested = *deployment.Spec.Replicas
		replicaCountsMap[resourceID] = replicaCounts
	}

	return replicaCountsMap
}

func apiStatusCode(apiStatus *resource.APIStatus) resource.StatusCode {
	if apiStatus.MaxReplicas == 0 {
		if apiStatus.TotalReady() > 0 {
			return resource.StatusStopping
		}
		return resource.StatusStopped
	}

	if apiStatus.TotalReady() > 0 {
		return resource.StatusLive
	}

	if apiStatus.TotalFailed() > 0 {
		return resource.StatusError
	}

	if apiStatus.K8sRequested != 0 {
		return resource.StatusCreating
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
		case resource.StatusKilled, resource.StatusKilledOOM:
			apiStatus.Code = resource.StatusParentKilled
			return
		case resource.StatusError:
			apiStatus.Code = resource.StatusParentFailed
			return
		case resource.StatusSkipped:
			parentSkipped = true
		case resource.StatusSucceeded:
			numSucceeded++
		}
	}

	if parentSkipped {
		apiStatus.Code = resource.StatusSkipped
		return
	}

	if numSucceeded == len(allDependencies) {
		apiStatus.Code = resource.StatusCreating
	}
}

func getAPIGroupStatuses(
	apiStatuses map[string]*resource.APIStatus,
	deployments map[string]*kapps.Deployment, // api.Name -> deployment
	ctx *context.Context,
) (map[string]*resource.APIGroupStatus, error) {

	apiGroupStatuses := make(map[string]*resource.APIGroupStatus)

	statusMap := make(map[string][]*resource.APIStatus)
	for _, apiStatus := range apiStatuses {
		apiName := apiStatus.APIName
		statusMap[apiName] = append(statusMap[apiName], apiStatus)
	}

	for apiName, apiStatuses := range statusMap {
		apiGroupStatuses[apiName] = &resource.APIGroupStatus{
			APIName:              apiName,
			Start:                k8s.DeploymentStartTime(deployments[apiName]),
			ActiveStatus:         getActiveAPIStatus(apiStatuses, ctx),
			Code:                 apiGroupStatusCode(apiStatuses, ctx),
			GroupedReplicaCounts: getGroupedReplicaCounts(apiStatuses, ctx),
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
	var latestAPIStatus *resource.APIStatus

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
		return resource.StatusStopped
	}

	apiName := apiStatuses[0].APIName
	ctxAPI := ctx.APIs[apiName]

	if ctxAPI == nil {
		for _, apiStatus := range apiStatuses {
			if apiStatus.TotalReady() > 0 {
				return resource.StatusStopping
			}
		}
		return resource.StatusStopped
	}

	for _, apiStatus := range apiStatuses {
		if apiStatus.TotalReady() > 0 {
			return resource.StatusLive
		}
	}

	for _, apiStatus := range apiStatuses {
		if apiStatus.ResourceID == ctxAPI.ID {
			return apiStatus.Code
		}
	}

	return resource.StatusUnknown
}

func getGroupedReplicaCounts(apiStatuses []*resource.APIStatus, ctx *context.Context) resource.GroupedReplicaCounts {
	groupedReplicaCounts := resource.GroupedReplicaCounts{}

	if len(apiStatuses) == 0 {
		return groupedReplicaCounts
	}

	apiName := apiStatuses[0].APIName
	ctxAPI := ctx.APIs[apiName]

	for _, apiStatus := range apiStatuses {
		if ctxAPI != nil && apiStatus.ResourceID == ctxAPI.ID {
			groupedReplicaCounts.ReadyUpdated = apiStatus.ReadyUpdatedCompute
			groupedReplicaCounts.ReadyStaleCompute = apiStatus.ReadyStaleCompute
			groupedReplicaCounts.FailedUpdated = apiStatus.FailedUpdatedCompute
			groupedReplicaCounts.FailedStaleCompute = apiStatus.FailedStaleCompute

			groupedReplicaCounts.Requested = ctxAPI.Compute.InitReplicas
			if apiStatus.K8sRequested > 0 {
				groupedReplicaCounts.Requested = apiStatus.K8sRequested
			}
			if groupedReplicaCounts.Requested < ctxAPI.Compute.MinReplicas {
				groupedReplicaCounts.Requested = ctxAPI.Compute.MinReplicas
			}
		} else {
			groupedReplicaCounts.ReadyStaleModel += apiStatus.TotalReady()
			groupedReplicaCounts.FailedStaleModel += apiStatus.TotalFailed()
		}
	}

	return groupedReplicaCounts
}

func setInsufficientComputeAPIStatusCodes(apiStatuses map[string]*resource.APIStatus, ctx *context.Context) error {
	stalledPods, err := config.Kubernetes.StalledPods()
	if err != nil {
		return err
	}
	stalledWorkloads := strset.New()
	for _, pod := range stalledPods {
		stalledWorkloads.Add(pod.Labels["workloadID"])
	}

	for _, apiStatus := range apiStatuses {
		if apiStatus.Code == resource.StatusPending || apiStatus.Code == resource.StatusWaiting || apiStatus.Code == resource.StatusCreating {
			if _, ok := stalledWorkloads[apiStatus.WorkloadID]; ok {
				apiStatus.Code = resource.StatusPendingCompute
			}
		}
	}

	return nil
}
