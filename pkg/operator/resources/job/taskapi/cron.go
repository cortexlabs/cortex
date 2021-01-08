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
	"fmt"
	"time"

	"github.com/cortexlabs/cortex/pkg/lib/errors"
	"github.com/cortexlabs/cortex/pkg/lib/k8s"
	"github.com/cortexlabs/cortex/pkg/lib/sets/strset"
	"github.com/cortexlabs/cortex/pkg/lib/telemetry"
	"github.com/cortexlabs/cortex/pkg/operator/config"
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
	_k8sJobExistenceGracePeriod  = 10 * time.Second
)

var _jobsToDelete = strset.New()
var _inProgressJobSpecMap = map[string]*spec.TaskJob{}

func ManageJobResources() error {
	inProgressJobKeys, err := job.ListAllInProgressJobKeys(userconfig.TaskAPIKind)
	if err != nil {
		return err
	}

	inProgressJobIDSet := strset.Set{}
	for _, jobKey := range inProgressJobKeys {
		inProgressJobIDSet.Add(jobKey.ID)
	}

	for jobID := range _inProgressJobSpecMap {
		if !inProgressJobIDSet.Has(jobID) {
			delete(_inProgressJobSpecMap, jobID)
		}
	}

	jobs, err := config.K8s.ListJobs(
		&kmeta.ListOptions{
			LabelSelector: klabels.SelectorFromSet(
				map[string]string{"apiKind": userconfig.TaskAPIKind.String()},
			).String(),
		},
	)
	if err != nil {
		return err
	}

	k8sJobMap := map[string]*kbatch.Job{}
	k8sJobIDSet := strset.Set{}
	for _, kJob := range jobs {
		k8sJobMap[kJob.Labels["jobID"]] = &kJob
		k8sJobIDSet.Add(kJob.Labels["jobID"])
	}

	for _, jobKey := range inProgressJobKeys {
		k8sJob := k8sJobMap[jobKey.ID]

		jobState, err := job.GetJobState(jobKey)
		if err != nil {
			// delete delete in progress file and job
			continue
		}

		if !jobState.Status.IsInProgress() {
			// delete delete in progress file and job
		}

		// check if job is still running, if it failed, get an appropriate status code

		// add job spec to _inProgressJobSpecMap map
		if _, ok := _inProgressJobSpecMap[jobKey.ID]; !ok {
			jobSpec, err := downloadJobSpec(jobKey)
			if err != nil {
				// TODO write to log stream
				// writeToJobLogStream(jobKey, err.Error(), "terminating job and cleaning up job resources")
				err := errors.FirstError(
					job.DeleteInProgressFile(jobKey),
					deleteJobRuntimeResources(jobKey),
				)
				if err != nil {
					telemetry.Error(err)
					errors.PrintError(err)
					continue
				}
				continue
			}
			_inProgressJobSpecMap[jobKey.ID] = jobSpec
		}
		jobSpec := _inProgressJobSpecMap[jobKey.ID]

		// check if timeout was exceeded
		if jobSpec.Timeout != nil && time.Since(jobSpec.StartTime) > time.Second*time.Duration(*jobSpec.Timeout) {
			err := errors.FirstError(
				job.SetTimedOutStatus(jobKey),
				deleteJobRuntimeResources(jobKey),
				// TODO write to log stream
				// writeToJobLogStream(jobKey, fmt.Sprintf("terminating job after exceeding the specified timeout of %d seconds", *jobSpec.Timeout)),
			)
			if err != nil {
				telemetry.Error(err)
				errors.PrintError(err)
			}
			continue
		}

		// check if job has completed
		if jobState.Status == status.JobRunning {
			// err = checkIfJobCompleted(jobKey, *queueURL, k8sJob)
			// if err != nil {
			// 	telemetry.Error(err)
			// 	errors.PrintError(err)
			// }
		}
	}

	// existing k8sjob but job is not in progress
	for jobID := range strset.Difference(k8sJobIDSet, inProgressJobIDSet) {
		jobKey := spec.JobKey{APIName: k8sJobMap[jobID].Labels["apiName"], ID: k8sJobMap[jobID].Labels["jobID"]}

		// delete both k8sjob and queue
		err := deleteJobRuntimeResources(jobKey)
		if err != nil {
			telemetry.Error(err)
			errors.PrintError(err)
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
func reconcileInProgressJob(jobState *job.State, queueURL *string, k8sJob *kbatch.Job) (status.JobCode, string, error) {
	jobKey := jobState.JobKey

	if jobState.Status == status.JobEnqueuing && time.Since(jobState.LastUpdatedMap[job.LivenessFile()]) >= _enqueuingLivenessPeriod+_enqueuingLivenessBuffer {
		return status.JobEnqueueFailed, fmt.Sprintf("terminating job %s; enqueuing liveness check failed", jobKey.UserString()), nil
	}

	if jobState.Status == status.JobRunning {
		if time.Now().Sub(jobState.LastUpdatedMap[status.JobRunning.String()]) <= _k8sJobExistenceGracePeriod {
			return jobState.Status, "", nil
		}

		if k8sJob == nil { // unexpected k8s job missing
			return status.JobUnexpectedError, fmt.Sprintf("terminating job %s; unable to find kubernetes job", jobKey.UserString()), nil
		}
	}

	return jobState.Status, "", nil
}

func checkIfJobCompleted(jobKey spec.JobKey, queueURL string, k8sJob *kbatch.Job) error {
	if int(k8sJob.Status.Failed) > 0 {
		return investigateJobFailure(jobKey)
	}

	jobSpec, err := downloadJobSpec(jobKey)
	if err != nil {
		return err
	}

	// TODO check if job has finished successfully
	// if jobSpec.TotalBatchCount == batchMetrics.Succeeded {
	// 	_jobsToDelete.Remove(jobKey.ID)
	// 	return errors.FirstError(
	// 		job.SetSucceededStatus(jobKey),
	// 		deleteJobRuntimeResources(jobKey),
	// 	)
	// }

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

func investigateJobFailure(jobKey spec.JobKey) error {
	reasonFound := false

	pods, _ := config.K8s.ListPodsByLabel("jobID", jobKey.ID)
	for _, pod := range pods {
		if k8s.WasPodOOMKilled(&pod) {
			return errors.FirstError(
				// writeToJobLogStream(jobKey, "at least one worker was killed because it ran out of out of memory"),
				job.SetWorkerOOMStatus(jobKey),
				deleteJobRuntimeResources(jobKey),
			)
		}
		// podStatus := k8s.GetPodStatus(&pod)
		for _, containerStatus := range pod.Status.ContainerStatuses {
			if containerStatus.LastTerminationState.Terminated != nil {
				// exitCode := containerStatus.LastTerminationState.Terminated.ExitCode
				// reason := strings.ToLower(containerStatus.LastTerminationState.Terminated.Reason)
				// _ = writeToJobLogStream(jobKey, fmt.Sprintf("at least one worker had status %s and terminated for reason %s (exit_code=%d)", string(podStatus), reason, exitCode))
				reasonFound = true
			} else if containerStatus.State.Terminated != nil {
				// exitCode := containerStatus.State.Terminated.ExitCode
				// reason := strings.ToLower(containerStatus.State.Terminated.Reason)
				// _ = writeToJobLogStream(jobKey, fmt.Sprintf("at least one worker had status %s and terminated for reason %s (exit_code=%d)", string(podStatus), reason, exitCode))
				reasonFound = true
			}
		}
	}

	var err error
	if !reasonFound {
		// err = writeToJobLogStream(jobKey, "workers were killed for unknown reason")
	}

	return errors.FirstError(
		err,
		job.SetWorkerErrorStatus(jobKey),
		deleteJobRuntimeResources(jobKey),
	)
}
