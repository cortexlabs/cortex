/*
Copyright 2021 Cortex Labs, Inc.

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

package asyncapi

import (
	"sort"
	"time"

	"github.com/cortexlabs/cortex/pkg/lib/errors"
	"github.com/cortexlabs/cortex/pkg/lib/k8s"
	"github.com/cortexlabs/cortex/pkg/lib/parallel"
	"github.com/cortexlabs/cortex/pkg/operator/config"
	"github.com/cortexlabs/cortex/pkg/operator/operator"
	"github.com/cortexlabs/cortex/pkg/types/status"
	"github.com/cortexlabs/cortex/pkg/types/userconfig"
	kapps "k8s.io/api/apps/v1"
	kcore "k8s.io/api/core/v1"
)

type asyncResourceGroup struct {
	APIDeployment     *kapps.Deployment
	APIPods           []kcore.Pod
	GatewayDeployment *kapps.Deployment
	GatewayPods       []kcore.Pod
}

func GetStatus(apiName string) (*status.Status, error) {
	var apiDeployment *kapps.Deployment
	var gatewayDeployment *kapps.Deployment
	var gatewayPods []kcore.Pod
	var apiPods []kcore.Pod

	err := parallel.RunFirstErr(
		func() error {
			var err error
			apiDeployment, err = config.K8s.GetDeployment(operator.K8sName(apiName))
			return err
		},
		func() error {
			var err error
			gatewayDeployment, err = config.K8s.GetDeployment(getGatewayK8sName(apiName))
			return err
		},
		func() error {
			var err error
			gatewayPods, err = config.K8s.ListPodsByLabels(
				map[string]string{
					"apiName":          apiName,
					"cortex.dev/async": "gateway",
				},
			)
			return err
		},
		func() error {
			var err error
			apiPods, err = config.K8s.ListPodsByLabels(
				map[string]string{
					"apiName":          apiName,
					"cortex.dev/async": "api",
				},
			)
			return err
		},
	)
	if err != nil {
		return nil, err
	}

	if apiDeployment == nil {
		return nil, errors.ErrorUnexpected("unable to find api deployment", apiName)
	}

	if gatewayDeployment == nil {
		return nil, errors.ErrorUnexpected("unable to find gateway deployment", apiName)
	}

	return apiStatus(apiDeployment, apiPods, gatewayDeployment, gatewayPods)
}

func GetAllStatuses(deployments []kapps.Deployment, pods []kcore.Pod) ([]status.Status, error) {
	resourcesByAPI := groupResourcesByAPI(deployments, pods)
	statuses := make([]status.Status, len(resourcesByAPI))

	var i int
	for _, k8sResources := range resourcesByAPI {
		st, err := apiStatus(k8sResources.APIDeployment, k8sResources.APIPods, k8sResources.GatewayDeployment, k8sResources.GatewayPods)
		if err != nil {
			return nil, err
		}
		statuses[i] = *st
		i++
	}

	sort.Slice(statuses, func(i, j int) bool {
		return statuses[i].APIName < statuses[j].APIName
	})

	return statuses, nil
}

func namesAndIDsFromStatuses(statuses []status.Status) ([]string, []string) {
	apiNames := make([]string, len(statuses))
	apiIDs := make([]string, len(statuses))

	for i, st := range statuses {
		apiNames[i] = st.APIName
		apiIDs[i] = st.APIID
	}

	return apiNames, apiIDs
}

// let's do CRDs instead, to avoid this
func groupResourcesByAPI(deployments []kapps.Deployment, pods []kcore.Pod) map[string]*asyncResourceGroup {
	resourcesByAPI := map[string]*asyncResourceGroup{}
	for i := range deployments {
		deployment := deployments[i]
		apiName := deployment.Labels["apiName"]
		asyncType := deployment.Labels["cortex.dev/async"]
		apiResources, exists := resourcesByAPI[apiName]
		if exists {
			if asyncType == "api" {
				apiResources.APIDeployment = &deployment
			} else {
				apiResources.GatewayDeployment = &deployment
			}
		} else {
			if asyncType == "api" {
				resourcesByAPI[apiName] = &asyncResourceGroup{APIDeployment: &deployment}
			} else {
				resourcesByAPI[apiName] = &asyncResourceGroup{GatewayDeployment: &deployment}
			}
		}
	}

	for _, pod := range pods {
		apiName := pod.Labels["apiName"]
		asyncType := pod.Labels["cortex.dev/async"]
		apiResources, exists := resourcesByAPI[apiName]
		if !exists {
			// ignore pods that might still be waiting to be deleted while the deployment has already been deleted
			continue
		}

		if asyncType == "api" {
			apiResources.APIPods = append(resourcesByAPI[apiName].APIPods, pod)
		} else {
			apiResources.GatewayPods = append(resourcesByAPI[apiName].GatewayPods, pod)
		}
	}
	return resourcesByAPI
}

func apiStatus(apiDeployment *kapps.Deployment, apiPods []kcore.Pod, gatewayDeployment *kapps.Deployment, gatewayPods []kcore.Pod) (*status.Status, error) {
	autoscalingSpec, err := userconfig.AutoscalingFromAnnotations(apiDeployment)
	if err != nil {
		return nil, err
	}

	apiReplicaCounts := getReplicaCounts(apiDeployment, apiPods)
	gatewayReplicaCounts := getReplicaCounts(gatewayDeployment, gatewayPods)

	st := &status.Status{}
	st.APIName = apiDeployment.Labels["apiName"]
	st.APIID = apiDeployment.Labels["apiID"]
	st.ReplicaCounts = apiReplicaCounts
	st.Code = getStatusCode(apiReplicaCounts, gatewayReplicaCounts, autoscalingSpec.MinReplicas)

	return st, nil
}

func getStatusCode(apiCounts status.ReplicaCounts, gatewayCounts status.ReplicaCounts, apiMinReplicas int32) status.Code {
	if apiCounts.Updated.Ready >= apiCounts.Requested && gatewayCounts.Updated.Ready >= 1 {
		return status.Live
	}

	if apiCounts.Updated.ErrImagePull > 0 || gatewayCounts.Updated.ErrImagePull > 0 {
		return status.ErrorImagePull
	}

	if apiCounts.Updated.Failed > 0 || apiCounts.Updated.Killed > 0 ||
		gatewayCounts.Updated.Failed > 0 || gatewayCounts.Updated.Killed > 0 {
		return status.Error
	}

	if apiCounts.Updated.KilledOOM > 0 || gatewayCounts.Updated.KilledOOM > 0 {
		return status.OOM
	}

	if apiCounts.Updated.Stalled > 0 || gatewayCounts.Updated.Stalled > 0 {
		return status.Stalled
	}

	if apiCounts.Updated.Ready >= apiMinReplicas && gatewayCounts.Updated.Ready >= 1 {
		return status.Live
	}

	return status.Updating
}

// returns true if min_replicas are not ready and no updated replicas have errored
func isAPIUpdating(deployment *kapps.Deployment) (bool, error) {
	pods, err := config.K8s.ListPodsByLabel("apiName", deployment.Labels["apiName"])
	if err != nil {
		return false, err
	}

	replicaCounts := getReplicaCounts(deployment, pods)

	autoscalingSpec, err := userconfig.AutoscalingFromAnnotations(deployment)
	if err != nil {
		return false, err
	}

	if replicaCounts.Updated.Ready < autoscalingSpec.MinReplicas && replicaCounts.Updated.TotalFailed() == 0 {
		return true, nil
	}

	return false, nil
}

func getReplicaCounts(deployment *kapps.Deployment, pods []kcore.Pod) status.ReplicaCounts {
	counts := status.ReplicaCounts{}
	counts.Requested = *deployment.Spec.Replicas

	for i := range pods {
		pod := pods[i]

		if pod.Labels["apiName"] != deployment.Labels["apiName"] {
			continue
		}
		addPodToReplicaCounts(&pod, deployment, &counts)
	}

	return counts
}

func addPodToReplicaCounts(pod *kcore.Pod, deployment *kapps.Deployment, counts *status.ReplicaCounts) {
	var subCounts *status.SubReplicaCounts
	if isPodSpecLatest(deployment, pod) {
		subCounts = &counts.Updated
	} else {
		subCounts = &counts.Stale
	}

	if k8s.IsPodReady(pod) {
		subCounts.Ready++
		return
	}

	switch k8s.GetPodStatus(pod) {
	case k8s.PodStatusPending:
		if time.Since(pod.CreationTimestamp.Time) > _stalledPodTimeout {
			subCounts.Stalled++
		} else {
			subCounts.Pending++
		}
	case k8s.PodStatusInitializing:
		subCounts.Initializing++
	case k8s.PodStatusRunning:
		subCounts.Initializing++
	case k8s.PodStatusErrImagePull:
		subCounts.ErrImagePull++
	case k8s.PodStatusTerminating:
		subCounts.Terminating++
	case k8s.PodStatusFailed:
		subCounts.Failed++
	case k8s.PodStatusKilled:
		subCounts.Killed++
	case k8s.PodStatusKilledOOM:
		subCounts.KilledOOM++
	default:
		subCounts.Unknown++
	}
}

func isPodSpecLatest(deployment *kapps.Deployment, pod *kcore.Pod) bool {
	return deployment.Spec.Template.Labels["predictorID"] == pod.Labels["predictorID"] &&
		deployment.Spec.Template.Labels["deploymentID"] == pod.Labels["deploymentID"]
}
