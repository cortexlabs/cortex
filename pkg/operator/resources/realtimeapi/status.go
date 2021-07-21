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

package realtimeapi

import (
	"time"

	"github.com/cortexlabs/cortex/pkg/consts"
	"github.com/cortexlabs/cortex/pkg/lib/k8s"
	"github.com/cortexlabs/cortex/pkg/types/status"
	kapps "k8s.io/api/apps/v1"
	kcore "k8s.io/api/core/v1"
)

func getReplicaCounts(deployment *kapps.Deployment, pods []kcore.Pod) status.ReplicaCounts {
	counts := status.ReplicaCounts{}
	counts.Requested = *deployment.Spec.Replicas

	for i := range pods {
		pod := pods[i]
		if pod.Labels["apiName"] != deployment.Labels["apiName"] {
			continue
		}
		addPodToReplicaCounts(&pods[i], deployment, &counts)
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
		if time.Since(pod.CreationTimestamp.Time) > consts.WaitForInitializingReplicasTimeout {
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
