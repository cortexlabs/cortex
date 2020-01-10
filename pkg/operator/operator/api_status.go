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
	"github.com/cortexlabs/cortex/pkg/types/spec"
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

	replicaCountsMap, podStatusMap := getReplicaCountsMap(podList, deployments, ctx)

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
		apiStatus.ReplicaCounts = replicaCountsMap[resourceID]
		apiStatus.PodStatuses = podStatusMap[resourceID]
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
) (map[string]resource.ReplicaCounts, map[string][]k8s.PodStatus) {

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
	podStatusMap := make(map[string][]k8s.PodStatus)
	for _, pod := range podList {
		resourceID := pod.Labels["resourceID"]
		podAPIComputeID := APIPodComputeID(pod.Spec.Containers)
		podStatus := k8s.GetPodStatus(&pod)
		isReady := k8s.IsPodReady(&pod)

		replicaCounts := replicaCountsMap[resourceID]

		computeMatches := false
		ctxAPIComputeID, ok := apiComputeIDMap[resourceID]
		if ok && ctxAPIComputeID == podAPIComputeID {
			computeMatches = true
		}

		if isReady {
			if computeMatches {
				replicaCounts.ReadyUpdatedCompute++
			} else {
				replicaCounts.ReadyStaleCompute++
			}
		}
		if podStatus == k8s.PodStatusFailed || podStatus == k8s.PodStatusKilled || podStatus == k8s.PodStatusKilledOOM {
			if computeMatches {
				replicaCounts.FailedUpdatedCompute++
			} else {
				replicaCounts.FailedStaleCompute++
			}
		}

		replicaCountsMap[resourceID] = replicaCounts
		podStatusMap[resourceID] = append(podStatusMap[resourceID], podStatus)
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

	return replicaCountsMap, podStatusMap
}

func numUpdatedReadyReplicas(ctx *context.Context, api *context.API) (int32, error) {
	podList, err := config.Kubernetes.ListPodsByLabels(map[string]string{
		"workloadType": workloadTypeAPI,
		"appName":      ctx.App.Name,
		"resourceID":   api.ID,
		"userFacing":   "true",
	})
	if err != nil {
		return 0, errors.Wrap(err, ctx.App.Name)
	}

	var readyReplicas int32
	apiComputeID := api.Compute.IDWithoutReplicas()
	for _, pod := range podList {
		if k8s.IsPodReady(&pod) && APIPodComputeID(pod.Spec.Containers) == apiComputeID {
			readyReplicas++
		}
	}

	return readyReplicas, nil
}

func apiStatusCode(apiStatus *resource.APIStatus) resource.StatusCode {
	if apiStatus.MaxReplicas == 0 {
		if apiStatus.TotalReady() > 0 {
			return resource.StatusStopping
		}
		return resource.StatusStopped
	}

	if apiStatus.FailedUpdatedCompute > 0 {
		for _, podStatus := range apiStatus.PodStatuses {
			if podStatus == k8s.PodStatusKilledOOM {
				return resource.StatusKilledOOM
			}
		}

		for _, podStatus := range apiStatus.PodStatuses {
			if podStatus == k8s.PodStatusKilled {
				return resource.StatusKilled
			}
		}

		return resource.StatusError
	}

	if apiStatus.ReadyUpdatedCompute >= apiStatus.MinReplicas {
		return resource.StatusLive
	}

	if apiStatus.K8sRequested != 0 {
		return resource.StatusUpdating
	}

	return resource.StatusPending
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
		groupedReplicaCounts := getGroupedReplicaCounts(apiStatuses, ctx)

		apiGroupStatuses[apiName] = &resource.APIGroupStatus{
			APIName:              apiName,
			ActiveStatus:         getActiveAPIStatus(apiStatuses, ctx),
			Code:                 apiGroupStatusCode(apiStatuses, groupedReplicaCounts, ctx),
			GroupedReplicaCounts: groupedReplicaCounts,
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

func apiGroupStatusCode(apiStatuses []*resource.APIStatus, groupedReplicaCounts resource.GroupedReplicaCounts, ctx *context.Context) resource.StatusCode {
	apiName := apiStatuses[0].APIName
	ctxAPI := ctx.APIs[apiName]

	var currentAPIStatus *resource.APIStatus
	if ctxAPI != nil {
		for _, apiStatus := range apiStatuses {
			if apiStatus.ResourceID == ctxAPI.ID {
				currentAPIStatus = apiStatus
			}
		}
	}

	if ctxAPI == nil || currentAPIStatus == nil || groupedReplicaCounts.Requested == 0 {
		if groupedReplicaCounts.Available() > 0 {
			return resource.StatusStopping
		}
		return resource.StatusStopped
	}

	return currentAPIStatus.Code
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
			groupedReplicaCounts.Requested = getRequestedReplicas(ctxAPI, apiStatus.K8sRequested, nil)
		} else {
			groupedReplicaCounts.ReadyStaleModel += apiStatus.TotalReady()
			groupedReplicaCounts.FailedStaleModel += apiStatus.TotalFailed()
		}
	}

	return groupedReplicaCounts
}

func getRequestedReplicasFromDeployment(api *spec.API, deployment *kapps.Deployment) int32 {
	var k8sRequested int32
	if deployment != nil && deployment.Spec.Replicas != nil {
		k8sRequested = *deployment.Spec.Replicas
	}
	return getRequestedReplicas(api, k8sRequested)
}

func getRequestedReplicas(api *spec.API, k8sRequested int32) int32 {
	requestedReplicas := api.Compute.InitReplicas
	if k8sRequested > 0 {
		requestedReplicas = k8sRequested
	}
	if requestedReplicas < api.Compute.MinReplicas {
		requestedReplicas = api.Compute.MinReplicas
	}
	if requestedReplicas > api.Compute.MaxReplicas {
		requestedReplicas = api.Compute.MaxReplicas
	}
	return requestedReplicas
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
		if apiStatus.Code == resource.StatusPending || apiStatus.Code == resource.StatusWaiting || apiStatus.Code == resource.StatusUpdating {
			if _, ok := stalledWorkloads[apiStatus.WorkloadID]; ok {
				apiStatus.Code = resource.StatusPendingCompute
			}
		}
	}

	return nil
}

func calculateAPISavedStatuses(podList []kcore.Pod, appName string) ([]*resource.APISavedStatus, error) {
	podMap := make(map[string]map[string][]kcore.Pod)
	for _, pod := range podList {
		resourceID := pod.Labels["resourceID"]
		workloadID := pod.Labels["workloadID"]
		if _, ok := podMap[resourceID]; !ok {
			podMap[resourceID] = make(map[string][]kcore.Pod)
		}
		podMap[resourceID][workloadID] = append(podMap[resourceID][workloadID], pod)
	}

	var savedStatuses []*resource.APISavedStatus
	for resourceID := range podMap {
		for workloadID, pods := range podMap[resourceID] {
			savedStatus, err := getAPISavedStatus(resourceID, workloadID, appName)
			if err != nil {
				return nil, err
			}
			if savedStatus == nil {
				savedStatus = &resource.APISavedStatus{
					BaseSavedStatus: resource.BaseSavedStatus{
						ResourceID:   resourceID,
						ResourceType: resource.APIType,
						WorkloadID:   workloadID,
						AppName:      pods[0].Labels["appName"],
					},
					APIName: pods[0].Labels["apiName"],
				}
			}

			updateAPISavedStatusStartTime(savedStatus, pods)

			savedStatuses = append(savedStatuses, savedStatus)
		}
	}

	return savedStatuses, nil
}
