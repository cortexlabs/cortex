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
	"github.com/cortexlabs/cortex/pkg/operator/operator"
	"github.com/cortexlabs/cortex/pkg/types/spec"
	"github.com/cortexlabs/cortex/pkg/types/status"
	kbatch "k8s.io/api/batch/v1"
)

func CleanupJobs() error {
	queues, err := operator.ListQueues()
	if err != nil {
		return err
	}

	jobs, err := config.K8s.ListJobs(nil)
	if err != nil {
		return err
	}

	k8sjobMap := map[string]kbatch.Job{}
	jobIDSetK8s := strset.Set{}
	for _, job := range jobs {
		k8sjobMap[job.Labels["jobID"]] = job
		jobIDSetK8s.Add(job.Labels["jobID"])
	}

	queueURLMap := map[string]string{}
	jobIDSetQueueURL := strset.Set{}
	for _, queueURL := range queues {
		jobID := operator.IdentifierFromQueueURL(queueURL)
		jobIDSetQueueURL.Add(jobID.ID)
		queueURLMap[jobID.ID] = queueURL
	}

	for id := range strset.Difference(jobIDSetK8s, jobIDSetQueueURL) {
		apiName := k8sjobMap[id].Labels["apiName"]
		jobID := spec.JobID{APIName: apiName, ID: id}
		err := handleK8sJobWithoutQueue(jobID)
		if err != nil {
			fmt.Println(err.Error())
			telemetry.Error(err)
		}
	}

	for jobID := range strset.Difference(jobIDSetQueueURL, jobIDSetK8s) {
		queueURL := queueURLMap[jobID]
		err := handleQueueWithoutK8sJob(queueURL)
		if err != nil {
			fmt.Println(err.Error())
			telemetry.Error(err)
		}
	}

	for jobID := range strset.Intersection(jobIDSetQueueURL, jobIDSetK8s) {
		queueURL := queueURLMap[jobID]
		k8sJob := k8sjobMap[jobID]
		err := handleQueueAndK8sJobPresent(queueURL, &k8sJob)
		if err != nil {
			fmt.Println(err.Error())
			telemetry.Error(err)
		}
	}

	return nil
}

func handleK8sJobWithoutQueue(jobID spec.JobID) error {
	fmt.Println("handleK8sJobWithoutQueue")
	queueURL, err := operator.QueueURL(jobID)
	if err != nil {
		return err
	}
	queueExists, err := operator.DoesQueueExist(queueURL) // double check queue existence because newly created queues take at least 30 seconds to be listed in operator.ListQueues()
	if err != nil {
		return err
	}

	if !queueExists {
		err := errors.FirstError(
			operator.WriteToJobLogGroup(jobID, fmt.Sprintf("terminating job %s because sqs queue with url %s was not found", jobID.UserString(), queueURL)),
			SetErroredStatus(jobID),
			DeleteJob(jobID),
		)
		if err != nil {
			return err
		}
	}

	return nil
}

func handleQueueWithoutK8sJob(queueURL string) error {
	fmt.Println("handleQueueWithoutK8sJob")

	jobID := operator.IdentifierFromQueueURL(queueURL)

	jobState, err := GetJobState(jobID)
	if err != nil {
		return err
	}

	jobStatusCode := jobState.GetStatusCode()
	if jobStatusCode == status.JobEnqueuing && time.Now().Sub(jobState.GetLastUpdated()) > time.Second*60 {
		err := errors.FirstError(
			operator.WriteToJobLogGroup(jobID, fmt.Sprintf("terminating job %s because liveness check failed", jobID.UserString(), queueURL)),
			SetErroredStatus(jobID),
			// DeleteJob(jobID),
		)
		if err != nil {
			return err
		}
	} else if jobStatusCode != status.JobRunning {
		DeleteQueue(queueURL) // queues take a while to delete, swallow this error
	}

	return nil
}

func handleQueueAndK8sJobPresent(queueURL string, k8sJob *kbatch.Job) error {
	fmt.Println("handleQueueAndK8sJobPresent")

	fmt.Println(queueURL)
	jobID := operator.IdentifierFromQueueURL(queueURL)

	queueMetrics, err := operator.GetQueueMetricsFromURL(queueURL)
	if err != nil {
		return err // TODO

	}
	jobMetrics, err := GetRealTimeJobMetrics(jobID)
	if err != nil {
		return err // TODO
	}

	if queueMetrics.IsEmpty() {
		// The deletion is delayed to give time for cloudwatch metrics to settle to determine if job succeeded or failed
		if k8sJob.Annotations["cortex/to-delete"] == "true" {
			if jobMetrics.Failed != 0 {
				return errors.FirstError(
					SetFailedStatus(jobID),
					DeleteJob(jobID),
				)
			} else {
				return errors.FirstError(
					SetSucceededStatus(jobID),
					DeleteJob(jobID),
				)
			}
		} else {
			if k8sJob.Annotations == nil {
				k8sJob.Annotations = map[string]string{}
			}
			k8sJob.Annotations["cortex/to-delete"] = "true"
			_, err := config.K8s.UpdateJob(k8sJob)
			if err != nil {
				return err
			}
		}
	} else {
		if int(k8sJob.Status.Active) == 0 {
			pods, _ := config.K8s.ListPodsByLabel("jobID", jobID.ID)
			for _, pod := range pods {
				podStatus := k8s.GetPodStatus(&pod)
				for _, containerStatus := range pod.Status.ContainerStatuses {
					if containerStatus.LastTerminationState.Terminated != nil {
						exitCode := containerStatus.LastTerminationState.Terminated.ExitCode
						reason := strings.ToLower(containerStatus.LastTerminationState.Terminated.Reason)
						operator.WriteToJobLogGroup(jobID, fmt.Sprintf("a worker had status %s and terminated for reason %s (exit_code=%d)", string(podStatus), reason, exitCode))

					} else if containerStatus.State.Terminated != nil {
						exitCode := containerStatus.State.Terminated.ExitCode
						reason := strings.ToLower(containerStatus.State.Terminated.Reason)
						operator.WriteToJobLogGroup(jobID, fmt.Sprintf("a worker had status %s and terminated for reason %s (exit_code=%d)", string(podStatus), reason, exitCode))
					}
				}
			}
			return errors.FirstError(
				nil,
				SetIncompleteStatus(jobID),
				DeleteJob(jobID),
			)
		}
	}
	return nil
}
