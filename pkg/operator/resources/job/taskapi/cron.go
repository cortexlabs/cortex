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
	"github.com/cortexlabs/cortex/pkg/operator/lib/logging"
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
	ManageJobResourcesCronPeriod = 60 * time.Second
	_k8sJobExistenceGracePeriod  = 10 * time.Second
)

var operatorLogger = logging.GetOperatorLogger()
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

	k8sJobMap := map[string]kbatch.Job{}
	k8sJobIDSet := strset.Set{}
	for _, kJob := range jobs {
		k8sJobMap[kJob.Labels["jobID"]] = kJob
		k8sJobIDSet.Add(kJob.Labels["jobID"])
	}

	for _, jobKey := range inProgressJobKeys {
		jobLogger, err := operator.GetJobLogger(jobKey)
		if err != nil {
			telemetry.Error(err)
			operatorLogger.Error(err)
			continue
		}

		k8sJob, jobFound := k8sJobMap[jobKey.ID]

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
			}
			continue
		}

		if !jobState.Status.IsInProgress() {
			// best effort cleanup
			_ = job.DeleteInProgressFile(jobKey)
			_ = deleteJobRuntimeResources(jobKey)
			continue
		}

		// reconcile job state and k8s job
		newStatusCode, msg, err := reconcileInProgressJob(jobState, jobFound)
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

		if _, ok := _inProgressJobSpecMap[jobKey.ID]; !ok {
			jobSpec, err := operator.DownloadTaskJobSpec(jobKey)
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
			err = checkIfJobCompleted(jobKey, k8sJob)
			if err != nil {
				telemetry.Error(err)
				operatorLogger.Error(err)
			}
		}
	}

	// existing K8s job but job is not in progress
	for jobID := range strset.Difference(k8sJobIDSet, inProgressJobIDSet) {
		jobKey := spec.JobKey{
			APIName: k8sJobMap[jobID].Labels["apiName"],
			ID:      k8sJobMap[jobID].Labels["jobID"],
		}

		err := deleteJobRuntimeResources(jobKey)
		if err != nil {
			telemetry.Error(err)
			operatorLogger.Error(err)
		}
	}

	return nil
}

// verifies k8s job exists for a job in running status, if verification fails return a job code to reflect the state
func reconcileInProgressJob(jobState *job.State, jobFound bool) (status.JobCode, string, error) {
	if jobState.Status == status.JobRunning {
		if time.Now().Sub(jobState.LastUpdatedMap[status.JobRunning.String()]) <= _k8sJobExistenceGracePeriod {
			return jobState.Status, "", nil
		}

		if !jobFound { // unexpected k8s job missing
			return status.JobUnexpectedError, fmt.Sprintf("terminating job %s; unable to find kubernetes job", jobState.JobKey.UserString()), nil
		}
	}

	return jobState.Status, "", nil
}

func checkIfJobCompleted(jobKey spec.JobKey, k8sJob kbatch.Job) error {
	pods, _ := config.K8s.ListPodsByLabel("jobID", jobKey.ID)
	for i := range pods {
		if k8s.WasPodOOMKilled(&pods[i]) {
			return errors.FirstError(
				job.SetWorkerOOMStatus(jobKey),
				deleteJobRuntimeResources(jobKey),
			)
		}
	}

	if int(k8sJob.Status.Failed) == 1 {
		return errors.FirstError(
			job.SetWorkerErrorStatus(jobKey),
			deleteJobRuntimeResources(jobKey),
		)
	} else if int(k8sJob.Status.Succeeded) == 1 && len(pods) > 0 {
		return errors.FirstError(
			job.SetSucceededStatus(jobKey),
			deleteJobRuntimeResources(jobKey),
		)
	}

	return nil
}
