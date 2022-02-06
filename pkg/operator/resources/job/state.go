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
	"path"
	"path/filepath"
	"time"

	"github.com/cortexlabs/cortex/pkg/config"
	"github.com/cortexlabs/cortex/pkg/lib/errors"
	"github.com/cortexlabs/cortex/pkg/lib/pointer"
	"github.com/cortexlabs/cortex/pkg/lib/strings"
	"github.com/cortexlabs/cortex/pkg/types/spec"
	"github.com/cortexlabs/cortex/pkg/types/status"
	"github.com/cortexlabs/cortex/pkg/types/userconfig"
)

const (
	_averageFilesPerJobState = 10
)

type State struct {
	spec.JobKey
	Status         status.JobCode
	LastUpdatedMap map[string]time.Time
	EndTime        *time.Time
}

func (j State) GetLastUpdated() time.Time {
	lastUpdated := time.Time{}

	for _, fileLastUpdated := range j.LastUpdatedMap {
		if lastUpdated.After(fileLastUpdated) {
			lastUpdated = fileLastUpdated
		}
	}

	return lastUpdated
}

func (j State) GetFirstCreated() time.Time {
	firstCreated := time.Unix(1<<63-62135596801, 999999999) // Max time

	for _, fileLastUpdated := range j.LastUpdatedMap {
		if firstCreated.After(fileLastUpdated) {
			firstCreated = fileLastUpdated
		}
	}

	return firstCreated
}

// Doesn't assume only status files are present. The order below matters.
func GetTaskStatusCode(lastUpdatedMap map[string]time.Time) status.JobCode {
	if _, ok := lastUpdatedMap[status.JobStopped.String()]; ok {
		return status.JobStopped
	}

	if _, ok := lastUpdatedMap[status.JobTimedOut.String()]; ok {
		return status.JobTimedOut
	}

	if _, ok := lastUpdatedMap[status.JobWorkerOOM.String()]; ok {
		return status.JobWorkerOOM
	}

	if _, ok := lastUpdatedMap[status.JobWorkerError.String()]; ok {
		return status.JobWorkerError
	}

	if _, ok := lastUpdatedMap[status.JobEnqueueFailed.String()]; ok {
		return status.JobEnqueueFailed
	}

	if _, ok := lastUpdatedMap[status.JobUnexpectedError.String()]; ok {
		return status.JobUnexpectedError
	}

	if _, ok := lastUpdatedMap[status.JobCompletedWithFailures.String()]; ok {
		return status.JobCompletedWithFailures
	}

	if _, ok := lastUpdatedMap[status.JobSucceeded.String()]; ok {
		return status.JobSucceeded
	}

	if _, ok := lastUpdatedMap[status.JobRunning.String()]; ok {
		return status.JobRunning
	}

	if _, ok := lastUpdatedMap[status.JobEnqueuing.String()]; ok {
		return status.JobEnqueuing
	}

	if _, ok := lastUpdatedMap[status.JobPending.String()]; ok {
		return status.JobPending
	}

	return status.JobUnknown
}

func GetBatchStatusCode(lastUpdatedMap map[string]time.Time) status.JobCode {
	if _, ok := lastUpdatedMap[status.JobTimedOut.String()]; ok {
		return status.JobTimedOut
	}

	if _, ok := lastUpdatedMap[status.JobWorkerOOM.String()]; ok {
		return status.JobWorkerOOM
	}

	if _, ok := lastUpdatedMap[status.JobWorkerError.String()]; ok {
		return status.JobWorkerError
	}

	if _, ok := lastUpdatedMap[status.JobEnqueueFailed.String()]; ok {
		return status.JobEnqueueFailed
	}

	if _, ok := lastUpdatedMap[status.JobUnexpectedError.String()]; ok {
		return status.JobUnexpectedError
	}

	if _, ok := lastUpdatedMap[status.JobCompletedWithFailures.String()]; ok {
		return status.JobCompletedWithFailures
	}

	if _, ok := lastUpdatedMap[status.JobSucceeded.String()]; ok {
		return status.JobSucceeded
	}

	if _, ok := lastUpdatedMap[status.JobStopped.String()]; ok {
		return status.JobStopped
	}

	if _, ok := lastUpdatedMap[status.JobRunning.String()]; ok {
		return status.JobRunning
	}

	if _, ok := lastUpdatedMap[status.JobEnqueuing.String()]; ok {
		return status.JobEnqueuing
	}

	if _, ok := lastUpdatedMap[status.JobPending.String()]; ok {
		return status.JobPending
	}

	return status.JobUnknown
}

func GetJobState(jobKey spec.JobKey) (*State, error) {
	s3Objects, err := config.AWS.ListS3Prefix(config.ClusterConfig.Bucket, jobKey.Prefix(config.ClusterConfig.ClusterUID), false, nil, nil)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get job state", jobKey.UserString())
	}

	if len(s3Objects) == 0 {
		return nil, errors.Wrap(ErrorJobNotFound(jobKey), "failed to get job state")
	}

	lastUpdatedMap := map[string]time.Time{}
	for _, object := range s3Objects {
		lastUpdatedMap[filepath.Base(*object.Key)] = *object.LastModified
	}

	jobState := getJobStateFromFiles(jobKey, lastUpdatedMap)
	return &jobState, nil
}

func getJobStateFromFiles(jobKey spec.JobKey, lastUpdatedFileMap map[string]time.Time) State {
	var statusCode status.JobCode
	switch jobKey.Kind {
	case userconfig.BatchAPIKind:
		statusCode = GetBatchStatusCode(lastUpdatedFileMap)
	case userconfig.TaskAPIKind:
		statusCode = GetTaskStatusCode(lastUpdatedFileMap)
	}

	var jobEndTime *time.Time
	if statusCode.IsCompleted() {
		if endTime, ok := lastUpdatedFileMap[statusCode.String()]; ok {
			jobEndTime = &endTime
		}
	}

	return State{
		JobKey:         jobKey,
		LastUpdatedMap: lastUpdatedFileMap,
		Status:         statusCode,
		EndTime:        jobEndTime,
	}
}

func GetMostRecentlySubmittedJobStates(apiName string, count int, kind userconfig.Kind) ([]*State, error) {
	// a single job state may include 5 files on average, overshoot the number of files needed
	apiPrefix := strings.EnsureSuffix(spec.JobAPIPrefix(config.ClusterConfig.ClusterUID, kind, apiName), "/")

	s3Objects, err := config.AWS.ListS3Prefix(
		config.ClusterConfig.Bucket,
		apiPrefix,
		false,
		pointer.Int64(int64(count*_averageFilesPerJobState)),
		nil,
	)

	if err != nil {
		return nil, err
	}

	// job id -> file name -> last update timestamp
	lastUpdatedMaps := map[string]map[string]time.Time{}
	var jobIDOrder []string
	for _, object := range s3Objects {
		if object == nil {
			continue
		}
		fileName := filepath.Base(*object.Key)
		jobID := filepath.Base(filepath.Dir(*object.Key))
		if _, ok := lastUpdatedMaps[jobID]; !ok {
			jobIDOrder = append(jobIDOrder, jobID)
			lastUpdatedMaps[jobID] = map[string]time.Time{fileName: *object.LastModified}
		} else {
			lastUpdatedMaps[jobID][fileName] = *object.LastModified
		}
	}

	jobStates := make([]*State, 0, count)

	jobStateCount := 0
	for _, jobID := range jobIDOrder {

		// it is possible to have fragmented deletes, spec.json should always be there
		_, found := lastUpdatedMaps[jobID]["spec.json"]
		if !found {
			go config.AWS.DeleteS3Dir(config.ClusterConfig.Bucket, path.Join(apiPrefix, jobID), true)
			continue
		}

		jobState := getJobStateFromFiles(spec.JobKey{
			APIName: apiName,
			ID:      jobID,
			Kind:    kind,
		}, lastUpdatedMaps[jobID])
		jobStates = append(jobStates, &jobState)

		jobStateCount++
		if jobStateCount == count {
			break
		}
	}

	return jobStates, nil
}

func SetStatusForJob(jobKey spec.JobKey, jobStatus status.JobCode) error {
	switch jobStatus {
	case status.JobEnqueuing:
		return SetEnqueuingStatus(jobKey)
	case status.JobRunning:
		return SetRunningStatus(jobKey)
	case status.JobEnqueueFailed:
		return SetEnqueueFailedStatus(jobKey)
	case status.JobCompletedWithFailures:
		return SetCompletedWithFailuresStatus(jobKey)
	case status.JobSucceeded:
		return SetSucceededStatus(jobKey)
	case status.JobUnexpectedError:
		return SetUnexpectedErrorStatus(jobKey)
	case status.JobWorkerError:
		return SetWorkerErrorStatus(jobKey)
	case status.JobWorkerOOM:
		return SetWorkerOOMStatus(jobKey)
	case status.JobTimedOut:
		return SetTimedOutStatus(jobKey)
	case status.JobStopped:
		return SetStoppedStatus(jobKey)
	}
	return nil
}

func UpdateLiveness(jobKey spec.JobKey) error {
	s3Key := path.Join(jobKey.Prefix(config.ClusterConfig.ClusterUID), _enqueuingLivenessFile)
	err := config.AWS.UploadJSONToS3(time.Now(), config.ClusterConfig.Bucket, s3Key)
	if err != nil {
		return errors.Wrap(err, "failed to update liveness", jobKey.UserString())
	}
	return nil
}

func SetEnqueuingStatus(jobKey spec.JobKey) error {
	err := UpdateLiveness(jobKey)
	if err != nil {
		return err
	}

	err = config.AWS.UploadStringToS3("", config.ClusterConfig.Bucket, path.Join(jobKey.Prefix(config.ClusterConfig.ClusterUID), status.JobEnqueuing.String()))
	if err != nil {
		return err
	}

	err = uploadInProgressFile(jobKey)
	if err != nil {
		return err
	}

	return nil
}

func SetFailedStatus(jobKey spec.JobKey) error {
	err := config.AWS.UploadStringToS3("", config.ClusterConfig.Bucket, path.Join(jobKey.Prefix(config.ClusterConfig.ClusterUID), status.JobEnqueueFailed.String()))
	if err != nil {
		return err
	}

	err = DeleteInProgressFile(jobKey)
	if err != nil {
		return err
	}

	return nil
}

func SetRunningStatus(jobKey spec.JobKey) error {
	err := config.AWS.UploadStringToS3("", config.ClusterConfig.Bucket, path.Join(jobKey.Prefix(config.ClusterConfig.ClusterUID), status.JobRunning.String()))
	if err != nil {
		return err
	}

	err = uploadInProgressFile(jobKey) // in progress file should already be there but just in case
	if err != nil {
		return err
	}

	return nil
}

func SetStoppedStatus(jobKey spec.JobKey) error {
	err := config.AWS.UploadStringToS3("", config.ClusterConfig.Bucket, path.Join(jobKey.Prefix(config.ClusterConfig.ClusterUID), status.JobStopped.String()))
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
	err := config.AWS.UploadStringToS3("", config.ClusterConfig.Bucket, path.Join(jobKey.Prefix(config.ClusterConfig.ClusterUID), status.JobSucceeded.String()))
	if err != nil {
		return err
	}

	err = DeleteInProgressFile(jobKey)
	if err != nil {
		return err
	}

	return nil
}

func SetCompletedWithFailuresStatus(jobKey spec.JobKey) error {
	err := config.AWS.UploadStringToS3("", config.ClusterConfig.Bucket, path.Join(jobKey.Prefix(config.ClusterConfig.ClusterUID), status.JobCompletedWithFailures.String()))
	if err != nil {
		return err
	}

	err = DeleteInProgressFile(jobKey)
	if err != nil {
		return err
	}

	return nil
}

func SetWorkerErrorStatus(jobKey spec.JobKey) error {
	err := config.AWS.UploadStringToS3("", config.ClusterConfig.Bucket, path.Join(jobKey.Prefix(config.ClusterConfig.ClusterUID), status.JobWorkerError.String()))
	if err != nil {
		return err
	}

	err = DeleteInProgressFile(jobKey)
	if err != nil {
		return err
	}

	return nil
}

func SetWorkerOOMStatus(jobKey spec.JobKey) error {
	err := config.AWS.UploadStringToS3("", config.ClusterConfig.Bucket, path.Join(jobKey.Prefix(config.ClusterConfig.ClusterUID), status.JobWorkerOOM.String()))
	if err != nil {
		return err
	}

	err = DeleteInProgressFile(jobKey)
	if err != nil {
		return err
	}

	return nil
}

func SetEnqueueFailedStatus(jobKey spec.JobKey) error {
	err := config.AWS.UploadStringToS3("", config.ClusterConfig.Bucket, path.Join(jobKey.Prefix(config.ClusterConfig.ClusterUID), status.JobEnqueueFailed.String()))
	if err != nil {
		return err
	}

	err = DeleteInProgressFile(jobKey)
	if err != nil {
		return err
	}

	return nil
}

func SetUnexpectedErrorStatus(jobKey spec.JobKey) error {
	err := config.AWS.UploadStringToS3("", config.ClusterConfig.Bucket, path.Join(jobKey.Prefix(config.ClusterConfig.ClusterUID), status.JobUnexpectedError.String()))
	if err != nil {
		return err
	}

	err = DeleteInProgressFile(jobKey)
	if err != nil {
		return err
	}

	return nil
}

func SetTimedOutStatus(jobKey spec.JobKey) error {
	err := config.AWS.UploadStringToS3("", config.ClusterConfig.Bucket, path.Join(jobKey.Prefix(config.ClusterConfig.ClusterUID), status.JobTimedOut.String()))
	if err != nil {
		return err
	}

	err = DeleteInProgressFile(jobKey)
	if err != nil {
		return err
	}

	return nil
}
