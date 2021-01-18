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

package taskapi

import (
	"github.com/cortexlabs/cortex/pkg/operator/config"
	"github.com/cortexlabs/cortex/pkg/operator/operator"
	"github.com/cortexlabs/cortex/pkg/operator/resources/job"
	"github.com/cortexlabs/cortex/pkg/types/spec"
	"github.com/cortexlabs/cortex/pkg/types/status"
	kbatch "k8s.io/api/batch/v1"
	kcore "k8s.io/api/core/v1"
)

func GetJobStatus(jobKey spec.JobKey) (*status.TaskJobStatus, error) {
	jobState, err := job.GetJobState(jobKey)
	if err != nil {
		return nil, err
	}

	k8sJob, err := config.K8s.GetJob(jobKey.K8sName())
	if err != nil {
		return nil, err
	}

	pods, err := config.K8s.ListPodsByLabels(map[string]string{"apiName": jobKey.APIName, "jobID": jobKey.ID})
	if err != nil {
		return nil, err
	}

	return getJobStatusFromJobState(jobState, k8sJob, pods)
}

func getJobStatusFromK8sJob(jobKey spec.JobKey, k8sJob *kbatch.Job, pods []kcore.Pod) (*status.TaskJobStatus, error) {
	jobState, err := job.GetJobState(jobKey)
	if err != nil {
		return nil, err
	}

	return getJobStatusFromJobState(jobState, k8sJob, pods)
}

func getJobStatusFromJobState(jobState *job.State, k8sJob *kbatch.Job, pods []kcore.Pod) (*status.TaskJobStatus, error) {
	jobKey := jobState.JobKey

	jobSpec, err := operator.DownloadTaskJobSpec(jobKey)
	if err != nil {
		return nil, err
	}

	jobStatus := status.TaskJobStatus{
		TaskJob: *jobSpec,
		EndTime: jobState.EndTime,
		Status:  jobState.Status,
	}

	if jobState.Status.IsInProgress() && k8sJob != nil {
		workerCounts := job.GetWorkerCountsForJob(*k8sJob, pods)
		jobStatus.WorkerCounts = &workerCounts
	}

	return &jobStatus, nil
}
