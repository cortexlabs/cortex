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

package batchapi

import (
	"time"

	"github.com/cortexlabs/cortex/pkg/lib/k8s"
	"github.com/cortexlabs/cortex/pkg/types/status"
	kbatch "k8s.io/api/batch/v1"
	kcore "k8s.io/api/core/v1"
)

const _stalledPodTimeout = 10 * time.Minute

func getWorkerStatsForJob(k8sJob kbatch.Job, pods []kcore.Pod) status.WorkerStats {
	if k8sJob.Status.Failed > 0 {
		return status.WorkerStats{
			Failed: *k8sJob.Spec.Parallelism,
		}
	}

	workerStats := status.WorkerStats{}
	for _, pod := range pods {
		addPodToReplicaCounts(&pod, &workerStats)
	}

	return workerStats
}

func addPodToReplicaCounts(pod *kcore.Pod, counts *status.WorkerStats) {
	if k8s.IsPodReady(pod) {
		counts.Running++
		return
	}

	switch k8s.GetPodStatus(pod) {
	case k8s.PodStatusPending:
		if time.Since(pod.CreationTimestamp.Time) > _stalledPodTimeout {
			counts.Stalled++
		} else {
			counts.Pending++
		}
	case k8s.PodStatusInitializing:
		counts.Initializing++
	case k8s.PodStatusRunning:
		counts.Initializing++
	case k8s.PodStatusErrImagePull:
		counts.Failed++
	case k8s.PodStatusTerminating:
		counts.Failed++
	case k8s.PodStatusFailed:
		counts.Failed++
	case k8s.PodStatusKilled:
		counts.Failed++
	case k8s.PodStatusKilledOOM:
		counts.Failed++
	case k8s.PodStatusSucceeded:
		counts.Succeeded++
	default:
		counts.Unknown++
	}
}
