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
	"path"
	"path/filepath"
	"time"

	"github.com/cortexlabs/cortex/pkg/lib/errors"
	"github.com/cortexlabs/cortex/pkg/lib/pointer"
	"github.com/cortexlabs/cortex/pkg/lib/sets/strset"
	"github.com/cortexlabs/cortex/pkg/operator/config"
	"github.com/cortexlabs/cortex/pkg/operator/operator"
	"github.com/cortexlabs/cortex/pkg/types/spec"
	"github.com/cortexlabs/cortex/pkg/types/status"
	kbatch "k8s.io/api/batch/v1"
)

var (
	_workerCountsFile = "worker_counts.json"
)

type JobState struct {
	spec.JobKey
	Status         status.JobCode
	Keys           []string
	LastUpdatedMap map[string]time.Time
	EndTime        *time.Time
}

func (j JobState) GetLastUpdated() time.Time {
	lastUpdated := time.Time{}

	for _, fileLastUpdated := range j.LastUpdatedMap {
		if lastUpdated.After(fileLastUpdated) {
			lastUpdated = fileLastUpdated
		}
	}

	return lastUpdated
}

// Doesn't assume only status files are present, order matters
func getStatusCode(fileKeys []string) status.JobCode {
	fileSet := strset.FromSlice(fileKeys)
	if fileSet.Has(status.JobStopped.String()) {
		return status.JobStopped
	}

	if fileSet.Has(status.JobIncomplete.String()) {
		return status.JobIncomplete
	}

	if fileSet.Has(status.JobFailed.String()) {
		return status.JobFailed
	}

	if fileSet.Has(status.JobErrored.String()) {
		return status.JobErrored
	}

	if fileSet.Has(status.JobSucceeded.String()) {
		return status.JobSucceeded
	}

	if fileSet.Has(status.JobRunning.String()) {
		return status.JobRunning
	}

	if fileSet.Has(status.JobEnqueuing.String()) {
		return status.JobEnqueuing
	}

	return status.JobUnknown
}

func getJobState(jobKey spec.JobKey) (*JobState, error) {
	s3Objects, err := config.AWS.ListS3Prefix(config.Cluster.Bucket, jobKey.PrefixKey(), false, pointer.Int64(100))
	if err != nil {
		return nil, errors.Wrap(err, "failed to get job state", jobKey.UserString())
	}

	keys := make([]string, 0, len(s3Objects))
	lastUpdatedMap := map[string]time.Time{}

	for _, s3Object := range s3Objects {
		key := filepath.Base(*s3Object.Key)
		keys = append(keys, key)
		lastUpdatedMap[key] = *s3Object.LastModified
	}

	statusCode := getStatusCode(keys)

	var jobEndTime *time.Time
	if endTime, ok := lastUpdatedMap[_workerCountsFile]; ok {
		jobEndTime = &endTime
	}

	if statusCode.IsCompletedPhase() {
		if endTime, ok := lastUpdatedMap[statusCode.String()]; ok {
			jobEndTime = &endTime
		}
	}

	return &JobState{
		JobKey:         jobKey,
		Keys:           keys,
		LastUpdatedMap: lastUpdatedMap,
		Status:         statusCode,
		EndTime:        jobEndTime,
	}, nil
}

func setEnqueuingStatus(jobKey spec.JobKey) error {
	err := uploadStatusFile(jobKey, status.JobEnqueuing)
	if err != nil {
		return err
	}

	err = uploadInProgressFile(jobKey)
	if err != nil {
		return err
	}

	return nil
}

func setRunningStatus(jobKey spec.JobKey) error {
	err := uploadStatusFile(jobKey, status.JobRunning)
	if err != nil {
		return err
	}
	return nil
}

func saveWorkerCounts(jobKey spec.JobKey) error {
	job, err := config.K8s.GetJob(jobKey.K8sName())
	if err != nil {
		return err
	}

	workerCounts := status.ExtractWorkerCounts(job)

	err = config.AWS.UploadJSONToS3(workerCounts, config.Cluster.Bucket, path.Join(jobKey.PrefixKey(), _workerCountsFile))
	if err != nil {
		return err
	}
	return nil
}

func setStoppedStatus(jobKey spec.JobKey) error {
	err := uploadStatusFile(jobKey, status.JobStopped)
	if err != nil {
		return err
	}

	err = saveWorkerCounts(jobKey)
	if err != nil {
		return err
	}

	err = deleteInProgressFile(jobKey)
	if err != nil {
		return err
	}

	return nil
}

func setSucceededStatus(jobKey spec.JobKey) error {
	err := uploadStatusFile(jobKey, status.JobSucceeded)
	if err != nil {
		return err
	}

	err = saveWorkerCounts(jobKey)
	if err != nil {
		return err
	}

	err = deleteInProgressFile(jobKey)
	if err != nil {
		return err
	}

	return nil
}

func setFailedStatus(jobKey spec.JobKey) error {
	err := uploadStatusFile(jobKey, status.JobFailed)
	if err != nil {
		return err
	}

	err = saveWorkerCounts(jobKey)
	if err != nil {
		return err
	}

	err = deleteInProgressFile(jobKey)
	if err != nil {
		return err
	}

	return nil
}

func setIncompleteStatus(jobKey spec.JobKey) error {
	err := uploadStatusFile(jobKey, status.JobIncomplete)
	if err != nil {
		return err
	}

	err = saveWorkerCounts(jobKey)
	if err != nil {
		return err
	}

	err = deleteInProgressFile(jobKey)
	if err != nil {
		return err
	}

	return nil
}

func setErroredStatus(jobKey spec.JobKey) error {
	err := uploadStatusFile(jobKey, status.JobErrored)
	if err != nil {
		return err
	}

	err = saveWorkerCounts(jobKey)
	if err != nil {
		return err
	}

	err = deleteInProgressFile(jobKey)
	if err != nil {
		return err
	}

	return nil
}

func uploadStatusFile(jobKey spec.JobKey, status status.JobCode) error {
	err := config.AWS.UploadStringToS3("", config.Cluster.Bucket, path.Join(jobKey.PrefixKey(), status.String()))
	if err != nil {
		return err
	}
	return nil
}

func GetJobStatus(jobKey spec.JobKey) (*status.JobStatus, error) {
	jobSpec, err := downloadJobSpec(jobKey)
	if err != nil {
		return nil, err
	}

	startTime := jobSpec.Created

	jobState, err := getJobState(jobKey)
	if err != nil {
		return nil, err
	}

	statusCode := jobState.Status

	jobStatus := status.JobStatus{
		Job:       *jobSpec,
		StartTime: startTime,
		EndTime:   jobState.EndTime,
		Status:    statusCode,
	}

	if statusCode.IsInProgressPhase() {
		queueMetrics, err := operator.GetQueueMetrics(jobKey)
		if err != nil {
			return nil, err
		}
		jobStatus.QueueMetrics = queueMetrics

		if statusCode == status.JobRunning {
			metrics, err := getRealTimeJobMetrics(jobKey)
			if err != nil {
				return nil, err
			}
			jobStatus.BatchMetrics = metrics

			k8sJob, err := config.K8s.GetJob(jobKey.K8sName())
			if err != nil {
				return nil, err
			}
			if k8sJob == nil {
				setErroredStatus(jobKey)
				operator.WriteToJobLogGroup(jobKey, fmt.Sprintf("k8s job not found"))
				deleteJobRuntimeResources(jobKey)
				jobStatus.Status = status.JobErrored
			}

			workerCounts := status.ExtractWorkerCounts(k8sJob)
			jobStatus.WorkerCounts = &workerCounts
		}
	} else {
		metrics, err := getJobMetrics(jobKey, startTime, *jobState.EndTime)
		if err != nil {
			return nil, err
		}
		jobStatus.BatchMetrics = metrics

		if _, ok := jobState.LastUpdatedMap[_workerCountsFile]; ok {
			var workerCounts status.WorkerCounts
			err = config.AWS.ReadJSONFromS3(&workerCounts, config.Cluster.Bucket, path.Join(jobKey.PrefixKey(), _workerCountsFile))
			if err != nil {
				return nil, err
			}

			jobStatus.WorkerCounts = &workerCounts
		}
	}

	return &jobStatus, nil
}

func getJobStatusFromK8sJob(jobKey spec.JobKey, k8sJob *kbatch.Job) (*status.JobStatus, error) {
	jobSpec, err := downloadJobSpec(jobKey)
	if err != nil {
		return nil, err
	}

	startTime := jobSpec.Created

	jobState, err := getJobState(jobKey)
	if err != nil {
		return nil, err
	}

	statusCode := jobState.Status

	jobStatus := status.JobStatus{
		Job:       *jobSpec,
		StartTime: startTime,
		Status:    statusCode,
	}

	queueMetrics, err := operator.GetQueueMetrics(jobKey)
	if err != nil {
		return nil, err
	}
	jobStatus.QueueMetrics = queueMetrics

	metrics, err := getRealTimeJobMetrics(jobKey)
	if err != nil {
		return nil, err
	}
	jobStatus.BatchMetrics = metrics

	workerCounts := status.ExtractWorkerCounts(k8sJob)
	jobStatus.WorkerCounts = &workerCounts

	return &jobStatus, nil
}
