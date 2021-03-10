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
	"time"

	"github.com/cortexlabs/cortex/pkg/lib/k8s"
	"github.com/cortexlabs/cortex/pkg/operator/config"
	"github.com/cortexlabs/cortex/pkg/types/status"
	"github.com/cortexlabs/cortex/pkg/types/userconfig"
	"k8s.io/api/apps/v1"
	v12 "k8s.io/api/core/v1"
)

// returns true if min_replicas are not ready and no updated replicas have errored
func isAPIUpdating(deployment *v1.Deployment) (bool, error) {
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

func getReplicaCounts(deployment *v1.Deployment, pods []v12.Pod) status.ReplicaCounts {
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

func addPodToReplicaCounts(pod *v12.Pod, deployment *v1.Deployment, counts *status.ReplicaCounts) {
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

func isPodSpecLatest(deployment *v1.Deployment, pod *v12.Pod) bool {
	return deployment.Spec.Template.Labels["predictorID"] == pod.Labels["predictorID"] &&
		deployment.Spec.Template.Labels["deploymentID"] == pod.Labels["deploymentID"]
}
