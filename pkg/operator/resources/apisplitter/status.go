/*
Copyright 2020 Cortex Labs, Inc.

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

package apisplitter

import (
	"sort"
	"time"

	"github.com/cortexlabs/cortex/pkg/lib/errors"
	"github.com/cortexlabs/cortex/pkg/lib/k8s"
	"github.com/cortexlabs/cortex/pkg/operator/config"
	"github.com/cortexlabs/cortex/pkg/operator/operator"
	"github.com/cortexlabs/cortex/pkg/types/status"
	istioclientnetworking "istio.io/client-go/pkg/apis/networking/v1alpha3"
	kapps "k8s.io/api/apps/v1"
	kcore "k8s.io/api/core/v1"
)

const _stalledPodTimeout = 10 * time.Minute

func GetStatus(apiName string) (*status.Status, error) {
	var virtualService *istioclientnetworking.VirtualService

	virtualService, err := config.K8s.GetVirtualService(operator.K8sName(apiName))
	if err != nil {
		return nil, err
	}

	if virtualService == nil {
		return nil, errors.ErrorUnexpected("unable to find trafficsplitter", apiName)
	}

	return trafficSplitterStatus(virtualService)
}

func GetAllStatuses() ([]status.Status, error) {
	var virtualServices []istioclientnetworking.VirtualService

	virtualServices, err := config.K8s.ListVirtualServicesWithLabelKeys("apiName")
	if err != nil {
		return nil, err
	}

	statuses := make([]status.Status, len(virtualServices))
	for i, virtualService := range virtualServices {
		status, err := trafficSplitterStatus(&virtualService)
		if err != nil {
			return nil, err
		}
		statuses[i] = *status
	}

	sort.Slice(statuses, func(i, j int) bool {
		return statuses[i].APIName < statuses[j].APIName
	})

	return statuses, nil
}

func trafficSplitterStatus(virtualService *istioclientnetworking.VirtualService) (*status.Status, error) {

	statusResponse := &status.Status{}
	statusResponse.APIName = virtualService.Labels["apiName"]
	statusResponse.APIID = virtualService.Labels["apiID"]
	// if virtual service deploy the trafficsplitter is actice
	// maybe need to check if backends are active
	statusResponse.Code = status.Live

	return statusResponse, nil
}

func getReplicaCounts(deployment *kapps.Deployment, pods []kcore.Pod) status.ReplicaCounts {
	counts := status.ReplicaCounts{}
	counts.Requested = *deployment.Spec.Replicas

	for _, pod := range pods {
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

func getStatusCode(counts *status.ReplicaCounts, minReplicas int32) status.Code {
	if counts.Updated.Ready >= counts.Requested {
		return status.Live
	}

	if counts.Updated.ErrImagePull > 0 {
		return status.ErrorImagePull
	}

	if counts.Updated.Failed > 0 || counts.Updated.Killed > 0 {
		return status.Error
	}

	if counts.Updated.KilledOOM > 0 {
		return status.OOM
	}

	if counts.Updated.Stalled > 0 {
		return status.Stalled
	}

	if counts.Updated.Ready >= minReplicas {
		return status.Live
	}

	return status.Updating
}
