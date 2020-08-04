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
	"strings"
	"time"

	"github.com/cortexlabs/cortex/pkg/lib/errors"
	"github.com/cortexlabs/cortex/pkg/lib/k8s"
	"github.com/cortexlabs/cortex/pkg/lib/pointer"
	"github.com/cortexlabs/cortex/pkg/lib/sets/strset"
	s "github.com/cortexlabs/cortex/pkg/lib/strings"
	"github.com/cortexlabs/cortex/pkg/lib/telemetry"
	"github.com/cortexlabs/cortex/pkg/operator/config"
	"github.com/cortexlabs/cortex/pkg/types/spec"
	"github.com/cortexlabs/cortex/pkg/types/status"
	kbatch "k8s.io/api/batch/v1"
)

const (
	ManageJobResourcesCronPeriod = 60 * time.Second
	DoesQueueExistGracePeriod    = 30 * time.Second
)

func ManageJobResources() error {
	inProgressJobKeys, err := listAllInProgressJobKeys()
	if err != nil {
		return err
	}

	inProgressJobIDSet := strset.Set{}
	inProgressIDMap := map[string]spec.JobKey{}
	for _, jobKey := range inProgressJobKeys {
		inProgressJobIDSet.Add(jobKey.ID)
		inProgressIDMap[jobKey.ID] = jobKey
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

	jobs, err := config.K8s.ListJobs(nil)
	if err != nil {
		return err
	}

	k8sJobMap := map[string]*kbatch.Job{}
	k8sJobIDSet := strset.Set{}
	for _, job := range jobs {
		k8sJobMap[job.Labels["jobID"]] = &job
		k8sJobIDSet.Add(job.Labels["jobID"])
	}

	for _, jobKey := range inProgressJobKeys {
		var queueURL *string
		if queueJobIDSet.Has(jobKey.ID) {
			queueURL = pointer.String(queueURLMap[jobKey.ID])
		}

		k8sJob := k8sJobMap[jobKey.ID]

		err := reconcileInProgressJob(jobKey, queueURL, k8sJob)
		if err != nil {
			telemetry.Error(err)
			errors.PrintError(err)
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

	// existing queue but no k8sjob and not in progress (existing queue, existing k8sjob and not in progress is handled by the for loop above)
	for jobID := range strset.Difference(queueJobIDSet, k8sJobIDSet, inProgressJobIDSet) {
		attributes, err := config.AWS.GetAllQueueAttributes(queueURLMap[jobID])
		if err != nil {
			telemetry.Error(err)
			errors.PrintError(err)
		}

		queueCreatedTimestamp := time.Time{}
		parsedSeconds, ok := s.ParseInt64(attributes["CreatedTimestamp"])
		if ok {
			queueCreatedTimestamp = time.Unix(parsedSeconds, 0)
		}

		// queue was created recently, maybe there was a delay between the time queue was created and when the in progress file was written
		if time.Now().Sub(queueCreatedTimestamp) <= DoesQueueExistGracePeriod {
			continue
		}

		jobKey := jobKeyFromQueueURL(queueURLMap[jobID])

		// delete both k8sjob and queue
		err = deleteJobRuntimeResources(jobKey)
		if err != nil {
			telemetry.Error(err)
			errors.PrintError(err)
		}
	}

	return nil
}

func reconcileInProgressJob(jobKey spec.JobKey, queueURL *string, k8sJob *kbatch.Job) error {
	jobState, err := getJobState(jobKey)
	if err != nil {
		return err
	}

	if !jobState.Status.IsInProgress() {
		// best effort cleanup
		if queueURL != nil {
			deleteQueueByURL(*queueURL)
		}
		if k8sJob != nil {
			deleteK8sJob(jobKey)
		}
		deleteInProgressFile(jobKey)
	}

	if jobState.Status.IsInProgress() {
		if queueURL == nil {
			queueExists, err := doesQueueExist(jobKey) // double check queue existence because it might take 10-20 seconds for a queue to be included in list queues api response after a queue has been created
			if err != nil {
				queueExists = false
			}

			if !queueExists && time.Now().Sub(jobState.LastUpdatedMap[status.JobEnqueuing.String()]) >= DoesQueueExistGracePeriod {
				expectedQueueURL, err := getJobQueueURL(jobKey)
				if err != nil {
					return err
				}

				// unexpected queue missing error
				return errors.FirstError(
					writeToJobLogGroup(jobKey, fmt.Sprintf("terminating job %s; sqs queue with url %s was not found", jobKey.UserString(), expectedQueueURL)),
					setUnexpectedErrorStatus(jobKey),
					deleteJobRuntimeResources(jobKey),
				)
			}
		} else {
			if jobState.Status == status.JobEnqueuing && time.Now().Sub(jobState.LastUpdatedMap[_lastUpdatedFile]) >= ManageJobResourcesCronPeriod {
				err = errors.FirstError(
					writeToJobLogGroup(jobKey, fmt.Sprintf("terminating job %s; enqueuing liveness check failed", jobKey.UserString())),
					setEnqueueFailedStatus(jobKey),
					deleteJobRuntimeResources(jobKey),
				)
				if err != nil {
					return err
				}
			}
			if jobState.Status == status.JobRunning {
				if k8sJob == nil { // unexpected k8s job missing
					err = errors.FirstError(
						writeToJobLogGroup(jobKey, fmt.Sprintf("terminating job %s; unable to find kubernetes job", jobKey.UserString())),
						setUnexpectedErrorStatus(jobKey),
						deleteJobRuntimeResources(jobKey),
					)
					if err != nil {
						return err
					}
				} else {
					err := checkJobCompletion(jobKey, *queueURL, k8sJob)
					if err != nil {
						return err
					}
				}
			}
		}
	}

	return nil
}

func checkJobCompletion(jobKey spec.JobKey, queueURL string, k8sJob *kbatch.Job) error {
	queueMetrics, err := getQueueMetricsFromURL(queueURL)
	if err != nil {
		return err

	}

	if queueMetrics.IsEmpty() {
		batchMetrics, err := getRealTimeBatchMetrics(jobKey)
		if err != nil {
			return err
		}

		jobSpec, err := downloadJobSpec(jobKey)
		if err != nil {
			return err
		}

		if int(k8sJob.Status.Failed) > 0 {
			return investigateJobFailure(jobKey, k8sJob)
		}

		if jobSpec.TotalBatchCount == batchMetrics.TotalCompleted || k8sJob.Annotations["cortex/to-delete"] == "true" {
			if batchMetrics.Failed != 0 {
				return errors.FirstError(
					setCompletedWithFailuresStatus(jobKey),
					deleteJobRuntimeResources(jobKey),
				)
			}
			return errors.FirstError(
				setSucceededStatus(jobKey),
				deleteJobRuntimeResources(jobKey),
			)
		}

		// Determine the status of the job in the next iteration to give time for cloudwatch metrics to settle down and determine if the job succeeded or failed
		if k8sJob.Annotations == nil {
			k8sJob.Annotations = map[string]string{}
		}
		k8sJob.Annotations["cortex/to-delete"] = "true"
		_, err = config.K8s.UpdateJob(k8sJob)
		if err != nil {
			return err
		}
	} else {
		if int(k8sJob.Status.Active) == 0 {
			return investigateJobFailure(jobKey, k8sJob)
		}
	}
	return nil
}

func investigateJobFailure(jobKey spec.JobKey, k8sJob *kbatch.Job) error {
	pods, _ := config.K8s.ListPodsByLabel("jobID", jobKey.ID)
	for _, pod := range pods {
		if k8s.WasPodOOMKilled(&pod) {
			return errors.FirstError(
				writeToJobLogGroup(jobKey, "at least one worker was killed because it ran out of out of memory"),
				setWorkerOOMStatus(jobKey),
				deleteJobRuntimeResources(jobKey),
			)
		}
		podStatus := k8s.GetPodStatus(&pod)
		var err error
		for _, containerStatus := range pod.Status.ContainerStatuses {
			if containerStatus.LastTerminationState.Terminated != nil {
				exitCode := containerStatus.LastTerminationState.Terminated.ExitCode
				reason := strings.ToLower(containerStatus.LastTerminationState.Terminated.Reason)
				err = writeToJobLogGroup(jobKey, fmt.Sprintf("at least one worker had status %s and terminated for reason %s (exit_code=%d)", string(podStatus), reason, exitCode))
			} else if containerStatus.State.Terminated != nil {
				exitCode := containerStatus.State.Terminated.ExitCode
				reason := strings.ToLower(containerStatus.State.Terminated.Reason)
				err = writeToJobLogGroup(jobKey, fmt.Sprintf("at least one worker had status %s and terminated for reason %s (exit_code=%d)", string(podStatus), reason, exitCode))
			}
		}
		return errors.FirstError(
			err,
			setWorkerErrorStatus(jobKey),
			deleteJobRuntimeResources(jobKey),
		)

	}
	return errors.FirstError(
		writeToJobLogGroup(jobKey, "workers were killed for unknown reason"),
		setWorkerErrorStatus(jobKey),
		deleteJobRuntimeResources(jobKey),
	)
}
