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

const _stalledPodTimeout = 10 * time.Minute

func GetStatus(apiName string) (*status.Status, error) {
	var deployment *kapps.Deployment
	var pods []kcore.Pod

	err := parallel.RunFirstErr(
		func() error {
			var err error
			deployment, err = config.K8s.GetDeployment(operator.K8sName(apiName))
			return err
		},
		func() error {
			var err error
			pods, err = config.K8s.ListPodsByLabel("apiName", apiName)
			return err
		},
	)
	if err != nil {
		return nil, err
	}

	if deployment == nil {
		return nil, errors.ErrorUnexpected("unable to find deployment", apiName)
	}

	return apiStatus(deployment, pods)
}

func GetAllStatuses(deployments []kapps.Deployment, pods []kcore.Pod) ([]status.Status, error) {
	statuses := make([]status.Status, len(deployments))
	for i := range deployments {
		st, err := apiStatus(&deployments[i], pods)
		if err != nil {
			return nil, err
		}
		statuses[i] = *st
	}

	sort.Slice(statuses, func(i, j int) bool {
		return statuses[i].APIName < statuses[j].APIName
	})

	return statuses, nil
}

func apiStatus(deployment *kapps.Deployment, allPods []kcore.Pod) (*status.Status, error) {
	autoscalingSpec, err := userconfig.AutoscalingFromAnnotations(deployment)
	if err != nil {
		return nil, err
	}

	status := &status.Status{}
	status.APIName = deployment.Labels["apiName"]
	status.APIID = deployment.Labels["apiID"]
	status.ReplicaCounts = getReplicaCounts(deployment, allPods)
	status.Code = getStatusCode(&status.ReplicaCounts, autoscalingSpec.MinReplicas)

	return status, nil
}

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
