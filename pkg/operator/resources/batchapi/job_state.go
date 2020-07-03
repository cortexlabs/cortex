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
	"path/filepath"
	"time"

	"github.com/cortexlabs/cortex/pkg/lib/debug"
	"github.com/cortexlabs/cortex/pkg/lib/errors"
	"github.com/cortexlabs/cortex/pkg/lib/pointer"
	"github.com/cortexlabs/cortex/pkg/lib/sets/strset"
	"github.com/cortexlabs/cortex/pkg/operator/config"
	"github.com/cortexlabs/cortex/pkg/types/spec"
	"github.com/cortexlabs/cortex/pkg/types/status"
	kbatch "k8s.io/api/batch/v1"
)

var (
	_lastUpdatedFile  = "last_updated"
	_workerCountsFile = "worker_counts.json"
)

type JobState struct {
	spec.JobID
	Keys           []string
	LastUpdatedMap map[string]time.Time
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
func (j JobState) GetStatusCode() status.JobCode {
	fileSet := strset.FromSlice(j.Keys)
	debug.Pp(fileSet)
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

func (j JobState) GetEndTime() *time.Time {
	fmt.Println("hi GetEndTime")
	if endTime, ok := j.LastUpdatedMap[_workerCountsFile]; ok {
		return &endTime
	}

	status := j.GetStatusCode()
	if status.IsCompletedPhase() {
		if endTime, ok := j.LastUpdatedMap[status.String()]; ok {
			return &endTime
		}
	}
	fmt.Println("bye GetEndTime")
	return nil
}

func GetJobState(jobID spec.JobID) (*JobState, error) {
	s3Objects, err := config.AWS.ListS3Prefix(config.Cluster.Bucket, spec.JobSpecPrefix(jobID), false, pointer.Int64(100))
	if err != nil {
		return nil, errors.Wrap(err, "failed to status", jobID.UserString())
	}

	jobState := JobState{
		JobID:          jobID,
		LastUpdatedMap: map[string]time.Time{},
	}

	for _, s3Object := range s3Objects {
		key := filepath.Base(*s3Object.Key)
		jobState.Keys = append(jobState.Keys, key)
		jobState.LastUpdatedMap[key] = *s3Object.LastModified
	}
	debug.Pp(jobState)

	return &jobState, nil
}

func UpdateLiveness(jobID spec.JobID) error {
	s3Key := filepath.Join(spec.JobSpecPrefix(jobID), _lastUpdatedFile)
	err := config.AWS.UploadJSONToS3(time.Now(), config.Cluster.Bucket, s3Key)
	if err != nil {
		return errors.Wrap(err, "failed to update liveness", jobID.UserString())
	}
	return nil
}

func SetEnqueuingStatus(jobSpec *spec.Job) error {
	err := config.AWS.UploadJSONToS3(jobSpec, config.Cluster.Bucket, spec.JobSpecKey(jobSpec.JobID))
	if err != nil {
		return err // TODO
	}

	err = config.AWS.UploadStringToS3("", config.Cluster.Bucket, filepath.Join(spec.JobSpecPrefix(jobSpec.JobID), status.JobEnqueuing.String()))
	if err != nil {
		return err // TODO
	}

	return nil
}

func SetRunningStatus(jobSpec *spec.Job) error {
	err := config.AWS.UploadJSONToS3(jobSpec, config.Cluster.Bucket, spec.JobSpecKey(jobSpec.JobID))
	if err != nil {
		return err // TODO
	}

	err = config.AWS.UploadStringToS3("", config.Cluster.Bucket, filepath.Join(spec.JobSpecPrefix(jobSpec.JobID), status.JobRunning.String()))
	if err != nil {
		return err // TODO
	}

	return nil
}

func saveWorkerCounts(jobID spec.JobID) error {
	job, err := config.K8s.GetJob(jobID.K8sName()) // TODO write empty worker file if job doesn't exist
	if err != nil {
		return err // TODO
	}

	workerCounts := status.ExtractWorkerCounts(job)

	err = config.AWS.UploadJSONToS3(workerCounts, config.Cluster.Bucket, filepath.Join(spec.JobSpecPrefix(jobID), _workerCountsFile))
	if err != nil {
		return err
	}
	return nil
}

func SetStoppedStatus(jobID spec.JobID) error {
	err := config.AWS.UploadStringToS3("", config.Cluster.Bucket, filepath.Join(spec.JobSpecPrefix(jobID), status.JobStopped.String()))
	if err != nil {
		return err // TODO
	}

	err = saveWorkerCounts(jobID)
	if err != nil {
		return err // TODO
	}

	return nil
}

func SetSucceededStatus(jobID spec.JobID) error {
	err := config.AWS.UploadStringToS3("", config.Cluster.Bucket, filepath.Join(spec.JobSpecPrefix(jobID), status.JobSucceeded.String()))
	if err != nil {
		return err // TODO
	}

	err = saveWorkerCounts(jobID)
	if err != nil {
		return err // TODO
	}

	return nil
}

func SetFailedStatus(jobID spec.JobID) error {
	err := config.AWS.UploadStringToS3("", config.Cluster.Bucket, filepath.Join(spec.JobSpecPrefix(jobID), status.JobFailed.String()))
	if err != nil {
		return err // TODO
	}

	err = saveWorkerCounts(jobID)
	if err != nil {
		return err // TODO
	}

	return nil
}

func SetIncompleteStatus(jobID spec.JobID) error {
	err := config.AWS.UploadStringToS3("", config.Cluster.Bucket, filepath.Join(spec.JobSpecPrefix(jobID), status.JobIncomplete.String()))
	if err != nil {
		return err // TODO
	}

	err = saveWorkerCounts(jobID)
	if err != nil {
		return err // TODO
	}

	return nil
}

func SetErroredStatus(jobID spec.JobID) error {
	err := config.AWS.UploadStringToS3("", config.Cluster.Bucket, filepath.Join(spec.JobSpecPrefix(jobID), status.JobErrored.String()))
	if err != nil {
		return err // TODO
	}

	err = saveWorkerCounts(jobID)
	if err != nil {
		return err // TODO
	}

	return nil
}

func DownloadJobSpec(jobID spec.JobID) (*spec.Job, error) {
	jobSpec := spec.Job{}
	debug.Pp(spec.JobSpecKey(jobID))
	err := config.AWS.ReadJSONFromS3(&jobSpec, config.Cluster.Bucket, spec.JobSpecKey(jobID))
	if err != nil {
		return nil, err // TODO
	}

	return &jobSpec, nil
}

func GetJobStatus(jobID spec.JobID) (*status.JobStatus, error) {
	jobSpec, err := DownloadJobSpec(jobID)
	if err != nil {
		return nil, err // TODO what happens if jobSpec doesn't exist
	}

	startTime := jobSpec.Created

	fmt.Println("GetJobStatus")
	jobState, err := GetJobState(jobID)
	if err != nil {
		return nil, err // TODO
	}

	debug.Pp(jobState)

	statusCode := jobState.GetStatusCode()
	fmt.Println(fmt.Sprintf("status for %s is %s", jobID.ID, statusCode.String()))

	jobStatus := status.JobStatus{
		Job:       *jobSpec,
		StartTime: startTime,
		Status:    statusCode,
	}

	if statusCode == status.JobRunning {
		metrics, err := GetRealTimeJobMetrics(jobID)
		if err != nil {
			return nil, err // TODO
		}
		jobStatus.Metrics = metrics

		k8sJob, err := config.K8s.GetJob(jobID.K8sName())
		if err != nil {
			return nil, err // TODO
		}
		workerCounts := status.ExtractWorkerCounts(k8sJob)
		jobStatus.WorkerCounts = &workerCounts
	} else if statusCode.IsCompletedPhase() {
		jobStatus.EndTime = jobState.GetEndTime()

		metrics, err := GetJobMetrics(jobID, startTime, *jobStatus.EndTime)
		if err != nil {
			return nil, err // TODO
		}
		jobStatus.Metrics = metrics

		if _, ok := jobState.LastUpdatedMap[_workerCountsFile]; ok {
			var workerCounts status.WorkerCounts
			err = config.AWS.ReadJSONFromS3(&workerCounts, config.Cluster.Bucket, filepath.Join(spec.JobSpecPrefix(jobID), _workerCountsFile))
			if err != nil {
				return nil, err // TODO
			}

			jobStatus.WorkerCounts = &workerCounts
		}
	}

	return &jobStatus, nil
}

func GetJobStatusFromK8sJob(jobID spec.JobID, k8sJob *kbatch.Job) (*status.JobStatus, error) {
	jobSpec, err := DownloadJobSpec(jobID)
	if err != nil {
		return nil, err // TODO what happens if jobSpec doesn't exist
	}

	startTime := jobSpec.Created

	jobState, err := GetJobState(jobID)
	if err != nil {
		return nil, err // TODO
	}

	statusCode := jobState.GetStatusCode()
	fmt.Println(fmt.Sprintf("status for %s is %s", jobID.ID, statusCode.String()))
	jobStatus := status.JobStatus{
		Job:       *jobSpec,
		StartTime: startTime,
		Status:    statusCode,
	}

	metrics, err := GetRealTimeJobMetrics(jobID)
	if err != nil {
		return nil, err // TODO
	}
	jobStatus.Metrics = metrics

	workerCounts := status.ExtractWorkerCounts(k8sJob)
	jobStatus.WorkerCounts = &workerCounts

	return &jobStatus, nil
}
