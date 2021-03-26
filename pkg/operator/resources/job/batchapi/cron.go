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

package batchapi

import (
	"fmt"
	"strings"
	"time"

	"github.com/cortexlabs/cortex/pkg/lib/errors"
	"github.com/cortexlabs/cortex/pkg/lib/k8s"
	"github.com/cortexlabs/cortex/pkg/lib/pointer"
	"github.com/cortexlabs/cortex/pkg/lib/sets/strset"
	s "github.com/cortexlabs/cortex/pkg/lib/strings"
	"github.com/cortexlabs/cortex/pkg/lib/telemetry"
	"github.com/cortexlabs/cortex/pkg/operator/config"
	"github.com/cortexlabs/cortex/pkg/operator/operator"
	"github.com/cortexlabs/cortex/pkg/operator/resources/job"
	"github.com/cortexlabs/cortex/pkg/types/spec"
	"github.com/cortexlabs/cortex/pkg/types/status"
	"github.com/cortexlabs/cortex/pkg/types/userconfig"
	kbatch "k8s.io/api/batch/v1"
	kmeta "k8s.io/apimachinery/pkg/apis/meta/v1"
	klabels "k8s.io/apimachinery/pkg/labels"
)

const (
	ManageJobResourcesCronPeriod = 60 * time.Second // If this is going to be updated (made smaller), update the batch worker implementation
	_doesQueueExistGracePeriod   = 30 * time.Second
	_enqueuingLivenessBuffer     = 30 * time.Second
	_k8sJobExistenceGracePeriod  = 10 * time.Second
)

var _jobsToDelete = strset.New()
var _inProgressJobSpecMap = map[string]*spec.BatchJob{}

func ManageJobResources() error {
	inProgressJobKeys, err := job.ListAllInProgressJobKeys(userconfig.BatchAPIKind)
	if err != nil {
		return err
	}

	inProgressJobIDSet := strset.Set{}
	for _, jobKey := range inProgressJobKeys {
		inProgressJobIDSet.Add(jobKey.ID)
	}

	// Remove completed jobs from local cache
	for jobID := range _inProgressJobSpecMap {
		if !inProgressJobIDSet.Has(jobID) {
			delete(_inProgressJobSpecMap, jobID)
		}
	}

	queues, err := listQueueURLsForAllAPIs()
	if err != nil {
		return err
	}

	queueURLMap := map[string]string{}
	queueJobIDSet := strset.Set{}
	for _, queueURL := range queues {
		jobKey := jobKeyFromQueueURL(queueURL)
		queueJobIDSet.Add(jobKey.ID)
		queueURLMap[jobKey.ID] = queueURL
	}

	jobs, err := config.K8s.ListJobs(
		&kmeta.ListOptions{
			LabelSelector: klabels.SelectorFromSet(
				map[string]string{"apiKind": userconfig.BatchAPIKind.String()},
			).String(),
		},
	)
	if err != nil {
		return err
	}

	k8sJobMap := map[string]kbatch.Job{}
	k8sJobIDSet := strset.Set{}
	for _, kJob := range jobs {
		k8sJobMap[kJob.Labels["jobID"]] = kJob
		k8sJobIDSet.Add(kJob.Labels["jobID"])
	}

	for _, jobKey := range inProgressJobKeys {
		var queueURL *string
		if queueJobIDSet.Has(jobKey.ID) {
			queueURL = pointer.String(queueURLMap[jobKey.ID])
		}

		k8sJob, jobFound := k8sJobMap[jobKey.ID]

		jobLogger, err := operator.GetJobLogger(jobKey)
		if err != nil {
			telemetry.Error(err)
			operatorLogger.Error(err)
			continue
		}

		jobState, err := job.GetJobState(jobKey)
		if err != nil {
			jobLogger.Error(err)
			jobLogger.Error("terminating job and cleaning up job resources")
			err := errors.FirstError(
				job.DeleteInProgressFile(jobKey),
				deleteJobRuntimeResources(jobKey),
			)
			if err != nil {
				telemetry.Error(err)
				operatorLogger.Error(err)
				continue
			}
			continue
		}

		if !jobState.Status.IsInProgress() {
			// best effort cleanup
			_ = job.DeleteInProgressFile(jobKey)
			_ = deleteJobRuntimeResources(jobKey)
			continue
		}

		newStatusCode, msg, err := reconcileInProgressJob(jobState, queueURL, jobFound)
		if err != nil {
			telemetry.Error(err)
			operatorLogger.Error(err)
			continue
		}
		if newStatusCode != jobState.Status {
			jobLogger.Error(msg)
			err := job.SetStatusForJob(jobKey, newStatusCode)
			if err != nil {
				telemetry.Error(err)
				operatorLogger.Error(err)
				continue
			}
		}
		if queueURL == nil {
			// job has been submitted within the grace period, it may take a while for a newly created queues and jobs to show up in list results
			continue
		}

		if _, ok := _inProgressJobSpecMap[jobKey.ID]; !ok {
			jobSpec, err := operator.DownloadBatchJobSpec(jobKey)
			if err != nil {
				jobLogger.Error(err)
				jobLogger.Error("terminating job and cleaning up job resources")
				err := errors.FirstError(
					job.DeleteInProgressFile(jobKey),
					deleteJobRuntimeResources(jobKey),
				)
				if err != nil {
					telemetry.Error(err)
					operatorLogger.Error(err)
				}
				continue
			}
			_inProgressJobSpecMap[jobKey.ID] = jobSpec
		}

		jobSpec := _inProgressJobSpecMap[jobKey.ID]

		if jobSpec.Timeout != nil && time.Since(jobSpec.StartTime) > time.Second*time.Duration(*jobSpec.Timeout) {
			jobLogger.Errorf("terminating job after exceeding the specified timeout of %d seconds", *jobSpec.Timeout)
			err := errors.FirstError(
				job.SetTimedOutStatus(jobKey),
				deleteJobRuntimeResources(jobKey),
			)
			if err != nil {
				telemetry.Error(err)
				operatorLogger.Error(err)
			}
			continue
		}

		if jobState.Status == status.JobRunning {
			err = checkIfJobCompleted(jobState, *queueURL, k8sJob)
			if err != nil {
				telemetry.Error(err)
				operatorLogger.Error(err)
			}
		}
	}

	// existing k8sjob but job is not in progress
	for jobID := range strset.Difference(k8sJobIDSet, inProgressJobIDSet) {
		jobKey := spec.JobKey{APIName: k8sJobMap[jobID].Labels["apiName"], ID: k8sJobMap[jobID].Labels["jobID"]}

		// delete both k8sjob and queue
		err := deleteJobRuntimeResources(jobKey)
		if err != nil {
			telemetry.Error(err)
			operatorLogger.Error(err)
		}
	}

	// existing queue but no k8sjob and not in progress (existing queue, existing k8sjob and not in progress is handled by the for loop above)
	for jobID := range strset.Difference(queueJobIDSet, k8sJobIDSet, inProgressJobIDSet) {
		attributes, err := config.AWS.GetAllQueueAttributes(queueURLMap[jobID])
		if err != nil {
			telemetry.Error(err)
			operatorLogger.Error(err)
		}

		queueCreatedTimestamp := time.Time{}
		parsedSeconds, ok := s.ParseInt64(attributes["CreatedTimestamp"])
		if ok {
			queueCreatedTimestamp = time.Unix(parsedSeconds, 0)
		}

		// queue was created recently, maybe there was a delay between the time queue was created and when the in progress file was written
		if time.Now().Sub(queueCreatedTimestamp) <= _doesQueueExistGracePeriod {
			continue
		}

		jobKey := jobKeyFromQueueURL(queueURLMap[jobID])

		// delete both k8sjob and queue
		err = deleteJobRuntimeResources(jobKey)
		if err != nil {
			telemetry.Error(err)
			operatorLogger.Error(err)
		}
	}

	// Clear old jobs to delete if they are no longer considered to in progress
	for jobID := range _jobsToDelete {
		if !inProgressJobIDSet.Has(jobID) {
			_jobsToDelete.Remove(jobID)
		}
	}

	return nil
}

// verifies that queue exists for an in progress job and k8s job exists for a job in running status, if verification fails return the a job code to reflect the state
func reconcileInProgressJob(jobState *job.State, queueURL *string, jobFound bool) (status.JobCode, string, error) {
	jobKey := jobState.JobKey

	if queueURL == nil {
		if time.Now().Sub(jobState.LastUpdatedMap[status.JobEnqueuing.String()]) <= _doesQueueExistGracePeriod {
			return jobState.Status, "", nil
		}

		expectedQueueURL, err := getJobQueueURL(jobKey)
		if err != nil {
			return jobState.Status, "", err
		}

		// unexpected queue missing error
		return status.JobUnexpectedError, fmt.Sprintf("terminating job %s; sqs queue with url %s was not found", jobKey.UserString(), expectedQueueURL), nil
	}

	if jobState.Status == status.JobEnqueuing && time.Since(jobState.LastUpdatedMap[job.LivenessFile()]) >= _enqueuingLivenessPeriod+_enqueuingLivenessBuffer {
		return status.JobEnqueueFailed, fmt.Sprintf("terminating job %s; enqueuing liveness check failed", jobKey.UserString()), nil
	}

	if jobState.Status == status.JobRunning {
		if time.Now().Sub(jobState.LastUpdatedMap[status.JobRunning.String()]) <= _k8sJobExistenceGracePeriod {
			return jobState.Status, "", nil
		}

		if !jobFound { // unexpected k8s job missing
			return status.JobUnexpectedError, fmt.Sprintf("terminating job %s; unable to find kubernetes job", jobKey.UserString()), nil
		}
	}

	return jobState.Status, "", nil
}

func checkIfJobCompleted(jobState *job.State, queueURL string, k8sJob kbatch.Job) error {
	jobKey := jobState.JobKey

	jobFailed, err := checkForJobFailure(jobKey, k8sJob)
	if err != nil || jobFailed {
		return err
	}

	jobLogger, err := operator.GetJobLogger(jobKey)
	if err != nil {
		return err
	}

	queueMessages, err := getQueueMetricsFromURL(queueURL)
	if err != nil {
		return err
	}

	if !queueMessages.IsEmpty() {
		// Give time for queue metrics to reach consistency
		if int(k8sJob.Status.Active) == 0 {
			if _jobsToDelete.Has(jobKey.ID) {
				_jobsToDelete.Remove(jobKey.ID)
				jobLogger.Error("unexpected job status because cluster state indicates job has completed but metrics indicate that job is still in progress")
				return errors.FirstError(
					job.SetUnexpectedErrorStatus(jobKey),
					deleteJobRuntimeResources(jobKey),
				)
			}
			_jobsToDelete.Add(jobKey.ID)
		}
		return nil
	}

	t := time.Now()
	if jobState.EndTime != nil {
		t = *jobState.EndTime
	}
	batchMetrics, err := getBatchMetrics(jobKey, t)
	if err != nil {
		return err
	}

	jobSpec, err := operator.DownloadBatchJobSpec(jobKey)
	if err != nil {
		return err
	}

	if jobSpec.TotalBatchCount == batchMetrics.Succeeded {
		_jobsToDelete.Remove(jobKey.ID)
		return errors.FirstError(
			job.SetSucceededStatus(jobKey),
			deleteJobRuntimeResources(jobKey),
		)
	}

	// wait one more cycle for the success metrics to reach consistency
	if _jobsToDelete.Has(jobKey.ID) {
		_jobsToDelete.Remove(jobKey.ID)
		return errors.FirstError(
			job.SetCompletedWithFailuresStatus(jobKey),
			deleteJobRuntimeResources(jobKey),
		)
	}

	// It takes at least 20 seconds for a worker to exit after determining that the queue is empty.
	// Queue metrics and cloud metrics both take a few seconds to achieve consistency.
	// Wait one more cycle for the workers to exit and metrics to acheive consistency before determining job status.
	_jobsToDelete.Add(jobKey.ID)

	return nil
}

func checkForJobFailure(jobKey spec.JobKey, k8sJob kbatch.Job) (bool, error) {
	jobLogger, err := operator.GetJobLogger(jobKey)
	if err != nil {
		return false, err
	}

	reasonFound := false
	pods, _ := config.K8s.ListPodsByLabel("jobID", jobKey.ID)
	for i := range pods {
		pod := pods[i] // to avoid loop pointer bugs
		if k8s.WasPodOOMKilled(&pod) {
			jobLogger.Error("at least one worker was killed because it ran out of out of memory")
			return true, errors.FirstError(
				job.SetWorkerOOMStatus(jobKey),
				deleteJobRuntimeResources(jobKey),
			)
		}
		if int(k8sJob.Status.Failed) > 0 {
			podStatus := k8s.GetPodStatus(&pod)
			for _, containerStatus := range pod.Status.ContainerStatuses {
				if containerStatus.LastTerminationState.Terminated != nil {
					exitCode := containerStatus.LastTerminationState.Terminated.ExitCode
					reason := containerStatus.LastTerminationState.Terminated.Reason
					if reason != k8s.ReasonCompleted || exitCode != 0 {
						jobLogger.Errorf("at least one worker had status %s and terminated for reason %s (exit_code=%d)", string(podStatus), strings.ToLower(reason), exitCode)
						reasonFound = true
					}
				} else if containerStatus.State.Terminated != nil {
					exitCode := containerStatus.State.Terminated.ExitCode
					reason := containerStatus.State.Terminated.Reason
					if reason != k8s.ReasonCompleted || exitCode != 0 {
						jobLogger.Errorf("at least one worker had status %s and terminated for reason %s (exit_code=%d)", string(podStatus), strings.ToLower(reason), exitCode)
						reasonFound = true
					}
				}
			}
		}
	}

	if int(k8sJob.Status.Failed) > 0 {
		if !reasonFound {
			jobLogger.Error("workers were killed for unknown reason")
		}
		return true, errors.FirstError(
			job.SetWorkerErrorStatus(jobKey),
			deleteJobRuntimeResources(jobKey),
		)
	}

	return false, nil
}
