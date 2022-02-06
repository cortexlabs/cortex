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

package job

import (
	"github.com/cortexlabs/cortex/pkg/lib/k8s"
	"github.com/cortexlabs/cortex/pkg/types/status"
	kbatch "k8s.io/api/batch/v1"
	kcore "k8s.io/api/core/v1"
)

func GetWorkerCountsForJob(k8sJob kbatch.Job, pods []kcore.Pod) status.WorkerCounts {
	if k8sJob.Status.Failed > 0 {
		return status.WorkerCounts{
			Failed: *k8sJob.Spec.Parallelism, // When one worker fails, the rest of the pods get deleted so you won't be able to get their statuses
		}
	}

	workerCounts := status.WorkerCounts{}
	for i := range pods {
		addPodToWorkerCounts(&pods[i], &workerCounts)
	}

	return workerCounts
}

func addPodToWorkerCounts(pod *kcore.Pod, workerCounts *status.WorkerCounts) {
	if k8s.IsPodReady(pod) {
		workerCounts.Ready++
		return
	}

	switch k8s.GetPodStatus(pod) {
	case k8s.PodStatusPending:
		workerCounts.Pending++
	case k8s.PodStatusStalled:
		workerCounts.Stalled++
	case k8s.PodStatusCreating:
		workerCounts.Creating++
	case k8s.PodStatusNotReady:
		workerCounts.NotReady++
	case k8s.PodStatusErrImagePull:
		workerCounts.ErrImagePull++
	case k8s.PodStatusTerminating:
		workerCounts.Terminating++
	case k8s.PodStatusFailed:
		workerCounts.Failed++
	case k8s.PodStatusKilled:
		workerCounts.Killed++
	case k8s.PodStatusKilledOOM:
		workerCounts.KilledOOM++
	case k8s.PodStatusSucceeded:
		workerCounts.Succeeded++
	case k8s.PodStatusUnknown:
		workerCounts.Unknown++
	}
}
