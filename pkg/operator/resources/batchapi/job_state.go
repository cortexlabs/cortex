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
	"strings"
	"time"

	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/cortexlabs/cortex/pkg/lib/debug"
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
	_lastUpdatedFile      = "last_updated"
	_workerCountsFile     = "worker_counts.json"
	_inProgressFilePrefix = "in_progress_jobs"
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

func GetJobState(jobKey spec.JobKey) (*JobState, error) {
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

func UpdateLiveness(jobKey spec.JobKey) error {
	s3Key := path.Join(jobKey.PrefixKey(), _lastUpdatedFile)
	err := config.AWS.UploadJSONToS3(time.Now(), config.Cluster.Bucket, s3Key)
	if err != nil {
		return errors.Wrap(err, "failed to update liveness", jobKey.UserString())
	}
	return nil
}

func SetEnqueuingStatus(jobSpec *spec.Job) error {
	err := config.AWS.UploadJSONToS3(jobSpec, config.Cluster.Bucket, jobSpec.FileSpecKey())
	if err != nil {
		return err
	}

	err = UploadStatusFile(jobSpec.JobKey, status.JobEnqueuing)
	if err != nil {
		return err
	}

	err = UploadInProgressFile(jobSpec.JobKey)
	if err != nil {
		return err
	}

	return nil
}

func SetRunningStatus(jobSpec *spec.Job) error {
	err := config.AWS.UploadJSONToS3(jobSpec, config.Cluster.Bucket, jobSpec.FileSpecKey())
	if err != nil {
		return err
	}

	err = UploadStatusFile(jobSpec.JobKey, status.JobRunning)
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

func SetStoppedStatus(jobKey spec.JobKey) error {
	err := UploadStatusFile(jobKey, status.JobStopped)
	if err != nil {
		return err
	}

	err = saveWorkerCounts(jobKey)
	if err != nil {
		return err
	}

	err = DeleteInProgressFile(jobKey)
	if err != nil {
		return err
	}

	return nil
}

func SetSucceededStatus(jobKey spec.JobKey) error {
	err := UploadStatusFile(jobKey, status.JobSucceeded)
	if err != nil {
		return err
	}

	err = saveWorkerCounts(jobKey)
	if err != nil {
		return err
	}

	err = DeleteInProgressFile(jobKey)
	if err != nil {
		return err
	}

	return nil
}

func SetFailedStatus(jobKey spec.JobKey) error {
	err := UploadStatusFile(jobKey, status.JobFailed)
	if err != nil {
		return err
	}

	err = saveWorkerCounts(jobKey)
	if err != nil {
		return err
	}

	err = DeleteInProgressFile(jobKey)
	if err != nil {
		return err
	}

	return nil
}

func SetIncompleteStatus(jobKey spec.JobKey) error {
	err := UploadStatusFile(jobKey, status.JobIncomplete)
	if err != nil {
		return err
	}

	err = saveWorkerCounts(jobKey)
	if err != nil {
		return err
	}

	err = DeleteInProgressFile(jobKey)
	if err != nil {
		return err
	}

	return nil
}

func SetErroredStatus(jobKey spec.JobKey) error {
	err := UploadStatusFile(jobKey, status.JobErrored)
	if err != nil {
		return err
	}

	err = saveWorkerCounts(jobKey)
	if err != nil {
		return err
	}

	err = DeleteInProgressFile(jobKey)
	if err != nil {
		return err
	}

	return nil
}

func UploadStatusFile(jobKey spec.JobKey, status status.JobCode) error {
	err := config.AWS.UploadStringToS3("", config.Cluster.Bucket, path.Join(jobKey.PrefixKey(), status.String()))
	if err != nil {
		return err
	}
	return nil
}

func UploadInProgressFile(jobKey spec.JobKey) error {
	err := config.AWS.UploadJSONToS3("", config.Cluster.Bucket, path.Join(_inProgressFilePrefix, jobKey.APIName, jobKey.ID))
	if err != nil {
		return err
	}
	return nil
}

func DeleteInProgressFile(jobKey spec.JobKey) error {
	err := config.AWS.DeleteS3Prefix(config.Cluster.Bucket, path.Join(_inProgressFilePrefix, jobKey.APIName, jobKey.ID), false)
	if err != nil {
		return err
	}
	return nil
}

func ListAllInProgressJobs() ([]spec.JobKey, error) {
	s3Objects, err := config.AWS.ListS3Dir(config.Cluster.Bucket, _inProgressFilePrefix, false, nil)
	if err != nil {
		return nil, err
	}

	return extractJobIDSFromS3ObjectList(s3Objects), nil
}

func ListAllInProgressJobsByAPI(apiName string) ([]spec.JobKey, error) {
	s3Objects, err := config.AWS.ListS3Dir(config.Cluster.Bucket, path.Join(_inProgressFilePrefix, apiName), false, nil)
	if err != nil {
		return nil, err
	}

	return extractJobIDSFromS3ObjectList(s3Objects), nil
}

func extractJobIDSFromS3ObjectList(s3Objects []*s3.Object) []spec.JobKey {
	jobIDs := make([]spec.JobKey, 0, len(s3Objects))
	for _, obj := range s3Objects {
		s3PathSplit := strings.Split(*obj.Key, "/")
		jobIDs = append(jobIDs, spec.JobKey{APIName: s3PathSplit[len(s3PathSplit)-2], ID: s3PathSplit[len(s3PathSplit)-1]})
	}

	return jobIDs
}

func DownloadJobSpec(jobKey spec.JobKey) (*spec.Job, error) {
	jobSpec := spec.Job{}
	err := config.AWS.ReadJSONFromS3(&jobSpec, config.Cluster.Bucket, jobSpec.FileSpecKey())
	if err != nil {
		return nil, ErrorJobNotFound(jobKey)
	}
	return &jobSpec, nil
}

func GetJobStatus(jobKey spec.JobKey) (*status.JobStatus, error) {
	fmt.Println("GetJobStatus")
	jobSpec, err := DownloadJobSpec(jobKey)
	if err != nil {
		return nil, err
	}

	startTime := jobSpec.Created

	fmt.Println("GetJobStatus")
	jobState, err := GetJobState(jobKey)
	if err != nil {
		return nil, err
	}

	debug.Pp(jobState)

	statusCode := jobState.Status
	fmt.Println(fmt.Sprintf("status for %s is %s", jobKey.ID, statusCode.String()))

	jobStatus := status.JobStatus{
		Job:       *jobSpec,
		StartTime: startTime,
		Status:    statusCode,
	}

	if statusCode == status.JobRunning {
		metrics, err := GetRealTimeJobMetrics(jobKey)
		if err != nil {
			return nil, err
		}
		jobStatus.Metrics = metrics

		k8sJob, err := config.K8s.GetJob(jobKey.K8sName())
		if err != nil {
			return nil, err
		}
		if k8sJob == nil {
			SetErroredStatus(jobKey)
			operator.WriteToJobLogGroup(jobKey, fmt.Sprintf("k8s job not found"))
			DeleteJobRuntimeResources(jobKey)
			jobStatus.Status = status.JobErrored
		}

		workerCounts := status.ExtractWorkerCounts(k8sJob)
		jobStatus.WorkerCounts = &workerCounts
	} else if statusCode.IsCompletedPhase() {
		jobStatus.EndTime = jobState.EndTime

		metrics, err := GetJobMetrics(jobKey, startTime, *jobStatus.EndTime)
		if err != nil {
			return nil, err
		}
		jobStatus.Metrics = metrics

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

func GetJobStatusFromK8sJob(jobKey spec.JobKey, k8sJob *kbatch.Job) (*status.JobStatus, error) {
	jobSpec, err := DownloadJobSpec(jobKey)
	if err != nil {
		return nil, err
	}

	startTime := jobSpec.Created

	jobState, err := GetJobState(jobKey)
	if err != nil {
		return nil, err
	}

	statusCode := jobState.Status
	fmt.Println(fmt.Sprintf("status for %s is %s", jobKey.ID, statusCode.String()))
	jobStatus := status.JobStatus{
		Job:       *jobSpec,
		StartTime: startTime,
		Status:    statusCode,
	}

	metrics, err := GetRealTimeJobMetrics(jobKey)
	if err != nil {
		return nil, err
	}
	jobStatus.Metrics = metrics

	workerCounts := status.ExtractWorkerCounts(k8sJob)
	jobStatus.WorkerCounts = &workerCounts

	return &jobStatus, nil
}
