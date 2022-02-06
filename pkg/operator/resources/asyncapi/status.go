/*
Copyright 2022 Cortex Labs, Inc.

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
	"github.com/cortexlabs/cortex/pkg/lib/k8s"
	"github.com/cortexlabs/cortex/pkg/types/status"
	kapps "k8s.io/api/apps/v1"
	kcore "k8s.io/api/core/v1"
)

func GetReplicaCounts(deployment *kapps.Deployment, pods []kcore.Pod) *status.ReplicaCounts {
	counts := status.ReplicaCounts{}
	counts.Requested = *deployment.Spec.Replicas

	for i := range pods {
		pod := pods[i]

		if pod.Labels["apiName"] != deployment.Labels["apiName"] {
			continue
		}
		addPodToReplicaCounts(&pod, deployment, &counts)
	}

	return &counts
}

func addPodToReplicaCounts(pod *kcore.Pod, deployment *kapps.Deployment, counts *status.ReplicaCounts) {
	latest := false
	if isPodSpecLatest(deployment, pod) {
		latest = true
	}

	isPodReady := k8s.IsPodReady(pod)
	if latest && isPodReady {
		counts.Ready++
		return
	} else if !latest && isPodReady {
		counts.ReadyOutOfDate++
		return
	}

	podStatus := k8s.GetPodStatus(pod)

	if podStatus == k8s.PodStatusTerminating {
		counts.Terminating++
		return
	}

	if !latest {
		return
	}

	switch podStatus {
	case k8s.PodStatusPending:
		counts.Pending++
	case k8s.PodStatusStalled:
		counts.Stalled++
	case k8s.PodStatusCreating:
		counts.Creating++
	case k8s.PodStatusReady:
		counts.Ready++
	case k8s.PodStatusNotReady:
		counts.NotReady++
	case k8s.PodStatusErrImagePull:
		counts.ErrImagePull++
	case k8s.PodStatusFailed:
		counts.Failed++
	case k8s.PodStatusKilled:
		counts.Killed++
	case k8s.PodStatusKilledOOM:
		counts.KilledOOM++
	case k8s.PodStatusUnknown:
		counts.Unknown++
	}
}
