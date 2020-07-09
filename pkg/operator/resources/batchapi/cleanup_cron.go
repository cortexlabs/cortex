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
	"github.com/cortexlabs/cortex/pkg/lib/sets/strset"
	"github.com/cortexlabs/cortex/pkg/lib/telemetry"
	"github.com/cortexlabs/cortex/pkg/operator/config"
	"github.com/cortexlabs/cortex/pkg/types/spec"
	"github.com/cortexlabs/cortex/pkg/types/status"
	kbatch "k8s.io/api/batch/v1"
)

const ManageJobResourcesCronPeriod = 60 * time.Second

func ManageJobResources() error {
	inProgressJobIDs, err := listAllInProgressJobs()
	if err != nil {
		return err
	}

	inProgressJobIDSet := strset.Set{}
	inProgressIDMap := map[string]spec.JobKey{}
	for _, jobKey := range inProgressJobIDs {
		inProgressJobIDSet.Add(jobKey.ID)
		inProgressIDMap[jobKey.ID] = jobKey
	}

	queues, err := listQueuesForAllAPIs()
	if err != nil {
		return err
	}

	jobs, err := config.K8s.ListJobs(nil)
	if err != nil {
		return err
	}

	k8sJobMap := map[string]kbatch.Job{}
	jobIDSetK8s := strset.Set{}
	for _, job := range jobs {
		k8sJobMap[job.Labels["jobID"]] = job
		jobIDSetK8s.Add(job.Labels["jobID"])
	}

	queueURLMap := map[string]string{}
	jobIDSetQueueURL := strset.Set{}
	for _, queueURL := range queues {
		jobKey := jobKeyFromQueueURL(queueURL)
		jobIDSetQueueURL.Add(jobKey.ID)
		queueURLMap[jobKey.ID] = queueURL
	}

	for _, jobKey := range inProgressJobIDs {
		jobState, err := getJobState(jobKey)
		if err != nil {
			fmt.Println(err.Error())
			telemetry.Error(err)
		}

		if jobState.Status.IsCompletedPhase() {
			if jobIDSetQueueURL.Has(jobKey.ID) {
				deleteQueueByURL(queueURLMap[jobKey.ID])
			}
			if jobIDSetK8s.Has(jobKey.ID) {
				deleteK8sJob(jobKey)
			}
		}

		if jobState.Status.IsInProgressPhase() {
			if !jobIDSetQueueURL.Has(jobKey.ID) {
				queueURL, err := getJobQueueURL(jobKey)
				if err != nil {
					fmt.Println(err.Error())
					telemetry.Error(err)
				}
				queueExists, err := doesQueueExist(jobKey)
				if err != nil {
					queueExists = false
				}

				if !queueExists && time.Now().Sub(jobState.LastUpdatedMap[status.JobEnqueuing.String()]) >= ManageJobResourcesCronPeriod {
					// unexpected queue missing error
					errorJobAndDelete(jobKey, fmt.Sprintf("terminating job %s; sqs queue with url %s was not found", jobKey.UserString(), queueURL))
				}
			} else {
				if jobState.Status == status.JobEnqueuing && time.Now().Sub(jobState.LastUpdatedMap[_lastUpdatedFile]) >= ManageJobResourcesCronPeriod {
					writeToJobLogGroup(jobKey, fmt.Sprintf("terminating job %s; enqueuing liveness check failed", jobKey.UserString()))
					setEnqueueFailedStatus(jobKey)
					deleteJobRuntimeResources(jobKey)
				}

				if jobState.Status == status.JobRunning {

					if !jobIDSetK8s.Has(jobKey.ID) {
						// unexpected k8s job missing
						errorJobAndDelete(jobKey, fmt.Sprintf("terminating job %s; unable to find kubernetes job", jobKey.UserString()))
					} else {
						k8sJob := k8sJobMap[jobKey.ID]
						err := checkJobCompletion(jobKey, queueURLMap[jobKey.ID], &k8sJob)
						if err != nil {
							fmt.Println(err.Error())
							telemetry.Error(err)
						}
					}
				}
			}
		} else {
			deleteInProgressFile(jobKey)
		}
	}

	for jobID := range strset.Difference(jobIDSetK8s, inProgressJobIDSet) {
		deleteJobRuntimeResources(
			spec.JobKey{APIName: k8sJobMap[jobID].Labels["apiName"], ID: k8sJobMap[jobID].Labels["jobID"]},
		)
	}

	for jobID := range strset.Difference(jobIDSetQueueURL, jobIDSetK8s, inProgressJobIDSet) {
		jobKey := jobKeyFromQueueURL(queueURLMap[jobID])
		deleteJobRuntimeResources(jobKey)
	}

	return nil
}

func errorJobAndDelete(jobKey spec.JobKey, message string, messages ...string) {
	writeToJobLogGroup(jobKey, message, messages...)
	setUnexpectedErrorStatus(jobKey)
	deleteJobRuntimeResources(jobKey)
}

func checkJobCompletion(jobKey spec.JobKey, queueURL string, k8sJob *kbatch.Job) error {
	queueMetrics, err := getQueueMetricsFromURL(queueURL)
	if err != nil {
		return err

	}
	jobMetrics, err := getRealTimeJobMetrics(jobKey)
	if err != nil {
		return err
	}

	if queueMetrics.IsEmpty() {
		jobSpec, err := downloadJobSpec(jobKey)
		if err != nil {
			return err
		}

		if jobSpec.TotalBatchCount == jobMetrics.TotalCompleted || k8sJob.Annotations["cortex/to-delete"] == "true" {
			if jobMetrics.Failed != 0 {
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
			InvestigateJobFailure(jobKey, k8sJob)
		}
	}
	return nil
}

func InvestigateJobFailure(jobKey spec.JobKey, k8sJob *kbatch.Job) {
	pods, _ := config.K8s.ListPodsByLabel("jobID", jobKey.ID)
	for _, pod := range pods {
		if k8s.WasPodOOMKilled(&pod) {
			writeToJobLogGroup(jobKey, "at least one worker was killed because it ran out of out of memory")
			setWorkerOOMStatus(jobKey)
			deleteJobRuntimeResources(jobKey)
		} else {
			podStatus := k8s.GetPodStatus(&pod)
			for _, containerStatus := range pod.Status.ContainerStatuses {
				if containerStatus.LastTerminationState.Terminated != nil {
					exitCode := containerStatus.LastTerminationState.Terminated.ExitCode
					reason := strings.ToLower(containerStatus.LastTerminationState.Terminated.Reason)
					writeToJobLogGroup(jobKey, fmt.Sprintf("at least one worker had status %s and terminated for reason %s (exit_code=%d)", string(podStatus), reason, exitCode))
				} else if containerStatus.State.Terminated != nil {
					exitCode := containerStatus.State.Terminated.ExitCode
					reason := strings.ToLower(containerStatus.State.Terminated.Reason)
					writeToJobLogGroup(jobKey, fmt.Sprintf("at least one worker had status %s and terminated for reason %s (exit_code=%d)", string(podStatus), reason, exitCode))
				}
			}
			setWorkerErrorStatus(jobKey)
			deleteJobRuntimeResources(jobKey)
		}
	}
}
