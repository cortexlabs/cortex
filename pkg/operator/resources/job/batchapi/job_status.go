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
	"fmt"

	"github.com/cortexlabs/cortex/pkg/lib/errors"
	"github.com/cortexlabs/cortex/pkg/operator/config"
	"github.com/cortexlabs/cortex/pkg/operator/resources/job"
	"github.com/cortexlabs/cortex/pkg/types/spec"
	"github.com/cortexlabs/cortex/pkg/types/status"
	kbatch "k8s.io/api/batch/v1"
	kcore "k8s.io/api/core/v1"
)

func GetJobStatus(jobKey spec.JobKey) (*status.JobStatus, error) {
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

func getJobStatusFromJobState(initialJobState *job.State, k8sJob *kbatch.Job, pods []kcore.Pod) (*status.JobStatus, error) {
	jobKey := initialJobState.JobKey

	jobSpec, err := downloadJobSpec(jobKey)
	if err != nil {
		return nil, err
	}

	latestJobState := initialJobState // Refetch the state of an in progress job in case the cron modifies the job state between the time the initial fetch and now

	if initialJobState.Status.IsInProgress() {
		queueURL, err := getJobQueueURL(jobKey)
		if err != nil {
			return nil, err
		}

		latestJobCode, message, err := reconcileInProgressJob(initialJobState, &queueURL, k8sJob)
		if err != nil {
			return nil, err
		}

		if latestJobCode != initialJobState.Status {
			err := errors.FirstError(
				writeToJobLogStream(jobKey, message),
				job.SetStatusForJob(jobKey, latestJobCode),
			)
			if err != nil {
				return nil, err
			}
		}

		latestJobState, err = job.GetJobState(jobKey)
		if err != nil {
			return nil, err
		}
	}

	jobStatus := status.JobStatus{
		BatchJob: *jobSpec,
		EndTime:  latestJobState.EndTime,
		Status:   latestJobState.Status,
	}

	if latestJobState.Status.IsInProgress() {
		queueMetrics, err := getQueueMetrics(jobKey)
		if err != nil {
			return nil, err
		}

		jobStatus.BatchesInQueue = queueMetrics.TotalUserMessages()

		if latestJobState.Status == status.JobEnqueuing {
			jobStatus.TotalBatchCount = queueMetrics.TotalUserMessages()
		}

		if latestJobState.Status == status.JobRunning {
			metrics, err := getRealTimeBatchMetrics(jobKey)
			if err != nil {
				return nil, err
			}
			jobStatus.BatchMetrics = metrics

			if k8sJob == nil {
				err := job.SetUnexpectedErrorStatus(jobKey)
				if err != nil {
					return nil, err
				}

				_ = writeToJobLogStream(jobKey, fmt.Sprintf("unexpected: kubernetes job not found"))
				_ = deleteJobRuntimeResources(jobKey)

				jobStatus.Status = status.JobUnexpectedError
				return &jobStatus, nil
			}

			workerCounts := getWorkerCountsForJob(*k8sJob, pods)
			jobStatus.WorkerCounts = &workerCounts
		}
	}

	if latestJobState.Status.IsCompleted() {
		metrics, err := getCompletedBatchMetrics(jobKey, jobSpec.StartTime, *latestJobState.EndTime)
		if err != nil {
			return nil, err
		}
		jobStatus.BatchMetrics = metrics
	}

	return &jobStatus, nil
}

func getJobStatusFromK8sJob(jobKey spec.JobKey, k8sJob *kbatch.Job, pods []kcore.Pod) (*status.JobStatus, error) {
	jobState, err := job.GetJobState(jobKey)
	if err != nil {
		return nil, err
	}

	return getJobStatusFromJobState(jobState, k8sJob, pods)
}
