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
	"time"

	"github.com/cortexlabs/cortex/pkg/lib/errors"
	"github.com/cortexlabs/cortex/pkg/lib/routines"
	"github.com/cortexlabs/cortex/pkg/lib/telemetry"
	"github.com/cortexlabs/cortex/pkg/operator/config"
	"github.com/cortexlabs/cortex/pkg/operator/operator"
	"github.com/cortexlabs/cortex/pkg/operator/resources/job"
	"github.com/cortexlabs/cortex/pkg/operator/schema"
	"github.com/cortexlabs/cortex/pkg/types/spec"
)

func SubmitJob(apiName string, submission *schema.TaskJobSubmission) (*spec.TaskJob, error) {
	err := validateJobSubmission(submission)
	if err != nil {
		return nil, err
	}

	virtualService, err := config.K8s.GetVirtualService(operator.K8sName(apiName))
	if err != nil {
		return nil, err
	}

	apiID := virtualService.Labels["apiID"]

	apiSpec, err := operator.DownloadAPISpec(apiName, apiID)
	if err != nil {
		return nil, err
	}

	jobID := spec.MonotonicallyDecreasingID()

	jobKey := spec.JobKey{
		APIName: apiSpec.Name,
		ID:      jobID,
		Kind:    apiSpec.Kind,
	}

	jobSpec := spec.TaskJob{
		JobKey:               jobKey,
		RuntimeTaskJobConfig: submission.RuntimeTaskJobConfig,
		APIID:                apiSpec.ID,
		SpecID:               apiSpec.SpecID,
		PredictorID:          apiSpec.PredictorID,
		StartTime:            time.Now(),
	}

	if err := uploadJobSpec(&jobSpec); err != nil {
		return nil, err
	}

	routines.RunWithPanicHandler(func() {
		deployJob(apiSpec, &jobSpec)
	}, false)

	return &jobSpec, nil
}

func downloadJobSpec(jobKey spec.JobKey) (*spec.TaskJob, error) {
	jobSpec := spec.TaskJob{}
	err := config.ReadJSONFromBucket(&jobSpec, jobKey.SpecFilePath(config.Cluster.ClusterName))
	if err != nil {
		return nil, errors.Wrap(err, "unable to download job specification", jobKey.UserString())
	}
	return &jobSpec, nil
}

func uploadJobSpec(jobSpec *spec.TaskJob) error {
	if err := config.UploadJSONToBucket(
		jobSpec, jobSpec.SpecFilePath(config.Cluster.ClusterName),
	); err != nil {
		return err
	}

	return nil
}

func deployJob(apiSpec *spec.API, jobSpec *spec.TaskJob) {
	err := createK8sJob(apiSpec, jobSpec)
	if err != nil {
		handleJobSubmissionError(jobSpec.JobKey, err)
	}

	err = job.SetRunningStatus(jobSpec.JobKey)
	if err != nil {
		handleJobSubmissionError(jobSpec.JobKey, err)
		return
	}
}

func handleJobSubmissionError(jobKey spec.JobKey, jobErr error) {
	err := errors.FirstError(
		//writeToJobLogStream(jobKey, jobErr.Error()),
		job.SetUnexpectedErrorStatus(jobKey),
		deleteJobRuntimeResources(jobKey),
	)
	if err != nil {
		telemetry.Error(err)
		errors.PrintError(err)
	}
}

func deleteJobRuntimeResources(jobKey spec.JobKey) error {
	err := deleteK8sJob(jobKey)
	if err != nil {
		return err
	}

	return nil
}

func StopJob(jobKey spec.JobKey) error {
	jobState, err := job.GetJobState(jobKey)
	if err != nil {
		routines.RunWithPanicHandler(func() {
			deleteJobRuntimeResources(jobKey)
		}, false)
		return err
	}

	if !jobState.Status.IsInProgress() {
		routines.RunWithPanicHandler(func() {
			deleteJobRuntimeResources(jobKey)
		}, false)
		return errors.Wrap(job.ErrorJobIsNotInProgress(jobKey.Kind), jobKey.UserString())
	}

	// writeToJobLogStream(jobKey, "request received to stop job; performing cleanup...")
	return errors.FirstError(
		deleteJobRuntimeResources(jobKey),
		job.SetStoppedStatus(jobKey),
	)
}
