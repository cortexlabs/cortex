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
	"time"

	"github.com/cortexlabs/cortex/pkg/lib/errors"
	"github.com/cortexlabs/cortex/pkg/lib/telemetry"
	"github.com/cortexlabs/cortex/pkg/operator/config"
	"github.com/cortexlabs/cortex/pkg/operator/lib/routines"
	"github.com/cortexlabs/cortex/pkg/operator/operator"
	"github.com/cortexlabs/cortex/pkg/operator/resources/job"
	"github.com/cortexlabs/cortex/pkg/operator/schema"
	"github.com/cortexlabs/cortex/pkg/types/spec"
)

func DryRun(submission *schema.BatchJobSubmission) ([]string, error) {
	err := validateJobSubmission(submission)
	if err != nil {
		return nil, err
	}

	if submission.FilePathLister != nil {
		s3Files, err := listFilesDryRun(&submission.FilePathLister.S3Lister)
		if err != nil {
			return nil, errors.Wrap(err, schema.FilePathListerKey)
		}

		return s3Files, nil
	}

	if submission.DelimitedFiles != nil {
		s3Files, err := listFilesDryRun(&submission.DelimitedFiles.S3Lister)
		if err != nil {
			return nil, errors.Wrap(err, schema.DelimitedFilesKey)
		}

		return s3Files, nil
	}

	return nil, nil
}

func SubmitJob(apiName string, submission *schema.BatchJobSubmission) (*spec.BatchJob, error) {
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

	tags := map[string]string{
		"apiName": apiSpec.Name,
		"apiID":   apiSpec.ID,
		"jobID":   jobID,
	}

	queueURL, err := createFIFOQueue(jobKey, submission.SQSDeadLetterQueue, tags)
	if err != nil {
		return nil, err
	}

	jobSpec := spec.BatchJob{
		RuntimeBatchJobConfig: submission.RuntimeBatchJobConfig,
		JobKey:                jobKey,
		APIID:                 apiSpec.ID,
		SpecID:                apiSpec.SpecID,
		PredictorID:           apiSpec.PredictorID,
		SQSUrl:                queueURL,
		StartTime:             time.Now(),
	}

	err = uploadJobSpec(&jobSpec)
	if err != nil {
		deleteQueueByURL(queueURL)
		return nil, err
	}

	jobLogger, err := operator.GetJobLoggerFromSpec(apiSpec, jobKey)
	if err != nil {
		deleteQueueByURL(queueURL)
		return nil, err
	}

	err = job.SetEnqueuingStatus(jobKey)
	if err != nil {
		deleteQueueByURL(queueURL)
		return nil, err
	}

	jobLogger.Info("started enqueuing batches")

	routines.RunWithPanicHandler(func() {
		deployJob(apiSpec, &jobSpec, submission)
	})

	return &jobSpec, nil
}

func uploadJobSpec(jobSpec *spec.BatchJob) error {
	err := config.AWS.UploadJSONToS3(jobSpec, config.CoreConfig.Bucket, jobSpec.SpecFilePath(config.ClusterName()))
	if err != nil {
		return err
	}
	return nil
}

func deployJob(apiSpec *spec.API, jobSpec *spec.BatchJob, submission *schema.BatchJobSubmission) {
	jobLogger, err := operator.GetJobLoggerFromSpec(apiSpec, jobSpec.JobKey)
	if err != nil {
		telemetry.Error(err)
		operatorLogger.Error(err)
		return
	}

	totalBatches, err := enqueue(jobSpec, submission)
	if err != nil {
		jobLogger.Error(errors.Wrap(err, "failed to enqueue all batches").Error())

		err := errors.FirstError(
			job.SetEnqueueFailedStatus(jobSpec.JobKey),
			deleteJobRuntimeResources(jobSpec.JobKey),
		)
		if err != nil {
			telemetry.Error(err)
			operatorLogger.Error(err)
		}
		return
	}

	if totalBatches == 0 {
		var errs []error
		jobLogger.Error(ErrorNoDataFoundInJobSubmission())
		if submission.DelimitedFiles != nil {
			jobLogger.Error("please verify that the files are not empty (the files being read can be retrieved by providing `dryRun=true` query param with your job submission")
		}
		errs = append(errs, job.SetEnqueueFailedStatus(jobSpec.JobKey))
		errs = append(errs, deleteJobRuntimeResources(jobSpec.JobKey))

		err := errors.FirstError(errs...)
		if err != nil {
			telemetry.Error(err)
			operatorLogger.Error(err)
		}
		return
	}

	jobLogger.Infof("completed enqueuing a total of %d batches", totalBatches)
	jobLogger.Infof("spinning up workers...")

	jobSpec.TotalBatchCount = totalBatches

	err = uploadJobSpec(jobSpec)
	if err != nil {
		handleJobSubmissionError(jobSpec.JobKey, err)
		return
	}

	err = createK8sJob(apiSpec, jobSpec)
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
		operatorLogger.Error(err)
	}
}

// delete k8s job, queue and save batch metrics from prometheus to cloud
func deleteJobRuntimeResources(jobKey spec.JobKey) error {
	err := errors.FirstError(
		deleteK8sJob(jobKey),
		deleteQueueWithDelay(jobKey),
		saveMetricsToCloud(jobKey),
	)

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
