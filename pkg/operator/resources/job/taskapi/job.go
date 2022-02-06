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

package taskapi

import (
	"time"

	"github.com/cortexlabs/cortex/pkg/config"
	"github.com/cortexlabs/cortex/pkg/lib/errors"
	"github.com/cortexlabs/cortex/pkg/lib/telemetry"
	"github.com/cortexlabs/cortex/pkg/operator/lib/routines"
	"github.com/cortexlabs/cortex/pkg/operator/operator"
	"github.com/cortexlabs/cortex/pkg/operator/resources/job"
	"github.com/cortexlabs/cortex/pkg/operator/schema"
	"github.com/cortexlabs/cortex/pkg/types/spec"
	"github.com/cortexlabs/cortex/pkg/workloads"
)

func SubmitJob(apiName string, submission *schema.TaskJobSubmission) (*spec.TaskJob, error) {
	err := validateJobSubmission(submission)
	if err != nil {
		return nil, err
	}

	virtualService, err := config.K8s.GetVirtualService(workloads.K8sName(apiName))
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
		PodID:                apiSpec.PodID,
		StartTime:            time.Now(),
	}

	if err := uploadJobSpec(&jobSpec); err != nil {
		return nil, err
	}

	deployJob(apiSpec, &jobSpec)

	return &jobSpec, nil
}

func uploadJobSpec(jobSpec *spec.TaskJob) error {
	if err := config.AWS.UploadJSONToS3(
		jobSpec, config.ClusterConfig.Bucket, jobSpec.SpecFilePath(config.ClusterConfig.ClusterUID),
	); err != nil {
		return err
	}

	return nil
}

func deployJob(apiSpec *spec.API, jobSpec *spec.TaskJob) {
	err := createJobConfigMap(*apiSpec, *jobSpec)
	if err != nil {
		handleJobSubmissionError(jobSpec.JobKey, err)
	}

	err = createK8sJob(apiSpec, jobSpec)
	if err != nil {
		handleJobSubmissionError(jobSpec.JobKey, err)
	}

	err = job.SetRunningStatus(jobSpec.JobKey)
	if err != nil {
		handleJobSubmissionError(jobSpec.JobKey, err)
	}
}

func createJobConfigMap(apiSpec spec.API, jobSpec spec.TaskJob) error {
	configMapConfig := workloads.ConfigMapConfig{
		TaskJob: &jobSpec,
	}

	configMapData, err := configMapConfig.GenerateConfigMapData()
	if err != nil {
		return err
	}

	return createK8sConfigMap(k8sConfigMap(apiSpec, jobSpec, configMapData))
}

func handleJobSubmissionError(jobKey spec.JobKey, jobErr error) {
	jobLogger, err := operator.GetJobLogger(jobKey)
	if err != nil {
		telemetry.Error(err)
		operatorLogger.Error(err)
		return
	}
	jobLogger.Error(jobErr.Error())
	err = errors.FirstError(
		job.SetUnexpectedErrorStatus(jobKey),
		deleteJobRuntimeResources(jobKey),
	)
	if err != nil {
		telemetry.Error(err)
		errors.PrintError(err)
	}
}

func deleteJobRuntimeResources(jobKey spec.JobKey) error {
	return errors.FirstError(
		deleteK8sJob(jobKey),
		deleteK8sConfigMap(jobKey),
	)
}

func StopJob(jobKey spec.JobKey) error {
	jobState, err := job.GetJobState(jobKey)
	if err != nil {
		routines.RunWithPanicHandler(func() {
			deleteJobRuntimeResources(jobKey)
		})
		return err
	}

	if !jobState.Status.IsInProgress() {
		routines.RunWithPanicHandler(func() {
			deleteJobRuntimeResources(jobKey)
		})
		return errors.Wrap(job.ErrorJobIsNotInProgress(jobKey.Kind), jobKey.UserString())
	}

	jobLogger, err := operator.GetJobLogger(jobKey)
	if err == nil {
		jobLogger.Warn("request received to stop job; performing cleanup...")
	}

	return errors.FirstError(
		deleteJobRuntimeResources(jobKey),
		job.SetStoppedStatus(jobKey),
	)
}
