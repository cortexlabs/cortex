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

	if fileSet.Has(status.JobWorkerOOM.String()) {
		return status.JobWorkerOOM
	}

	if fileSet.Has(status.JobWorkerError.String()) {
		return status.JobWorkerError
	}

	if fileSet.Has(status.JobEnqueueFailed.String()) {
		return status.JobEnqueueFailed
	}

	if fileSet.Has(status.JobUnexpectedError.String()) {
		return status.JobUnexpectedError
	}

	if fileSet.Has(status.JobCompletedWithFailures.String()) {
		return status.JobCompletedWithFailures
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
	s3Objects, err := config.AWS.ListS3Prefix(config.Cluster.Bucket, jobKey.PrefixKey(), false, pointer.Int64(10))
	if err != nil {
		return nil, errors.Wrap(err, "failed to get job state", jobKey.UserString())
	}

	lastUpdatedMap := map[string]time.Time{}

	for _, s3Object := range s3Objects {
		lastUpdatedMap[filepath.Base(*s3Object.Key)] = *s3Object.LastModified
	}

	jobState := getJobStateFromFiles(jobKey, lastUpdatedMap)
	return &jobState, nil
}

func getJobStateFromFiles(jobKey spec.JobKey, lastUpdatedFileMap map[string]time.Time) JobState {
	keys := make([]string, 0, len(lastUpdatedFileMap))

	for key := range lastUpdatedFileMap {
		keys = append(keys, key)
	}

	statusCode := getStatusCode(keys)

	var jobEndTime *time.Time
	if endTime, ok := lastUpdatedFileMap[_workerCountsFile]; ok {
		jobEndTime = &endTime
	}

	if statusCode.IsCompletedPhase() {
		if endTime, ok := lastUpdatedFileMap[statusCode.String()]; ok {
			jobEndTime = &endTime
		}
	}

	return JobState{
		JobKey:         jobKey,
		Keys:           keys,
		LastUpdatedMap: lastUpdatedFileMap,
		Status:         statusCode,
		EndTime:        jobEndTime,
	}
}

func getMostRecentlySubmittedJobStates(apiName string, count int) ([]*JobState, error) {
	s3Objects, err := config.AWS.ListS3Prefix(config.Cluster.Bucket, spec.APIJobPrefix(apiName), false, pointer.Int64(int64(count*10))) // overshoot the number of files needed
	if err != nil {
		return nil, err
	}

	lastUpdatedMaps := map[string]map[string]time.Time{}

	jobIDOrder := []string{}
	for _, s3Object := range s3Objects {
		fileName := filepath.Base(*s3Object.Key)
		jobID := filepath.Base(filepath.Dir(*s3Object.Key))

		if _, ok := lastUpdatedMaps[jobID]; !ok {
			jobIDOrder = append(jobIDOrder, jobID)
			lastUpdatedMaps[jobID] = map[string]time.Time{fileName: *s3Object.LastModified}
		} else {
			lastUpdatedMaps[jobID][fileName] = *s3Object.LastModified
		}
	}

	jobStates := make([]*JobState, 0, count)

	jobStateCount := 0
	for _, jobID := range jobIDOrder {
		jobState := getJobStateFromFiles(spec.JobKey{APIName: apiName, ID: jobID}, lastUpdatedMaps[jobID])
		jobStates = append(jobStates, &jobState)

		jobStateCount++
		if jobStateCount == count {
			break
		}
	}

	return jobStates, nil
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

func setCompletedWithFailuresStatus(jobKey spec.JobKey) error {
	err := uploadStatusFile(jobKey, status.JobCompletedWithFailures)
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

func setWorkerErrorStatus(jobKey spec.JobKey) error {
	err := uploadStatusFile(jobKey, status.JobWorkerError)
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

func setWorkerOOMStatus(jobKey spec.JobKey) error {
	err := uploadStatusFile(jobKey, status.JobWorkerOOM)
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

func setEnqueueFailedStatus(jobKey spec.JobKey) error {
	err := uploadStatusFile(jobKey, status.JobEnqueueFailed)
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

func setUnexpectedErrorStatus(jobKey spec.JobKey) error {
	err := uploadStatusFile(jobKey, status.JobUnexpectedError)
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

func getJobStatusFromJobState(jobState *JobState, k8sJob *kbatch.Job) (*status.JobStatus, error) {
	jobKey := jobState.JobKey

	jobSpec, err := downloadJobSpec(jobKey)
	if err != nil {
		return nil, err
	}

	startTime := jobSpec.Created

	statusCode := jobState.Status

	jobStatus := status.JobStatus{
		Job:       *jobSpec,
		StartTime: startTime,
		EndTime:   jobState.EndTime,
		Status:    statusCode,
	}

	if statusCode.IsInProgressPhase() {
		queueMetrics, err := getQueueMetrics(jobKey)
		if err != nil {
			return nil, err
		}
		jobStatus.QueueMetrics = queueMetrics

		if statusCode == status.JobEnqueuing {
			jobStatus.TotalBatchCount = queueMetrics.TotalInQueue()
		}

		if statusCode == status.JobRunning {
			metrics, err := getRealTimeJobMetrics(jobKey)
			if err != nil {
				return nil, err
			}
			jobStatus.BatchMetrics = metrics

			if k8sJob == nil {
				setUnexpectedErrorStatus(jobKey)
				writeToJobLogGroup(jobKey, fmt.Sprintf("kubernetes job not found"))
				deleteJobRuntimeResources(jobKey)
				jobStatus.Status = status.JobUnexpectedError
			}

			workerCounts := status.ExtractWorkerCounts(k8sJob)
			jobStatus.WorkerCounts = &workerCounts
		}
	}

	if statusCode.IsCompletedPhase() {
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

func GetJobStatus(jobKey spec.JobKey) (*status.JobStatus, error) {
	jobState, err := getJobState(jobKey)
	if err != nil {
		return nil, err
	}

	var k8sJob *kbatch.Job
	if jobState.Status == status.JobRunning {
		k8sJob, err = config.K8s.GetJob(jobKey.K8sName())
		if err != nil {
			return nil, err
		}
	}

	return getJobStatusFromJobState(jobState, k8sJob)
}

func getJobStatusFromK8sJob(jobKey spec.JobKey, k8sJob *kbatch.Job) (*status.JobStatus, error) {
	jobState, err := getJobState(jobKey)
	if err != nil {
		return nil, err
	}

	return getJobStatusFromJobState(jobState, k8sJob)
}
