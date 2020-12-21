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

package taskapi

import (
	"time"

	"github.com/cortexlabs/cortex/pkg/lib/errors"
	"github.com/cortexlabs/cortex/pkg/lib/telemetry"
	"github.com/cortexlabs/cortex/pkg/operator/config"
	"github.com/cortexlabs/cortex/pkg/operator/operator"
	"github.com/cortexlabs/cortex/pkg/operator/resources/job"
	"github.com/cortexlabs/cortex/pkg/operator/schema"
	"github.com/cortexlabs/cortex/pkg/types/spec"
)

func SubmitJob(apiName string, submission *schema.TaskJobSubmission) (*spec.TaskJob, error) {
	// TODO: submission validation, might not be required

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
		JobKey:           jobKey,
		RuntimeJobConfig: submission.RuntimeJobConfig,
		APIID:            apiSpec.ID,
		SpecID:           apiSpec.SpecID,
		PredictorID:      apiSpec.PredictorID,
		StartTime:        time.Now(),
	}

	if err := uploadJobSpec(&jobSpec); err != nil {
		return nil, err
	}

	// TODO: create log stream for job

	if err := job.SetEnqueuingStatus(jobKey); err != nil {
		return nil, err
	}

	// TODO: write to job log stream

	go deployJob(apiSpec, &jobSpec)

	return &jobSpec, nil
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
	// FIXME write error to log stream
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

func uploadJobSpec(jobSpec *spec.TaskJob) error {
	if err := config.AWS.UploadJSONToS3(
		jobSpec, config.Cluster.Bucket, jobSpec.SpecFilePath(config.Cluster.ClusterName),
	); err != nil {
		return err
	}

	return nil
}

func deleteJobRuntimeResources(jobKey spec.JobKey) error {
	err := deleteK8sJob(jobKey)
	if err != nil {
		return err
	}

	return nil
}

func createK8sJob(apiSpec *spec.API, jobSpec *spec.TaskJob) error {
	k8sJob, err := k8sJobSpec(apiSpec, jobSpec)
	if err != nil {
		return err
	}

	_, err = config.K8s.CreateJob(k8sJob)
	if err != nil {
		return err
	}

	return nil
}
