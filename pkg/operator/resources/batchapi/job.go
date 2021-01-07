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
	"github.com/cortexlabs/cortex/pkg/lib/routines"
	"github.com/cortexlabs/cortex/pkg/lib/telemetry"
	"github.com/cortexlabs/cortex/pkg/operator/config"
	"github.com/cortexlabs/cortex/pkg/operator/operator"
	"github.com/cortexlabs/cortex/pkg/operator/schema"
	"github.com/cortexlabs/cortex/pkg/types/spec"
	kmeta "k8s.io/apimachinery/pkg/apis/meta/v1"
	klabels "k8s.io/apimachinery/pkg/labels"
)

func DryRun(submission *schema.JobSubmission) ([]string, error) {
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

func SubmitJob(apiName string, submission *schema.JobSubmission) (*spec.Job, error) {
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

	jobSpec := spec.Job{
		RuntimeJobConfig: submission.RuntimeJobConfig,
		JobKey:           jobKey,
		APIID:            apiSpec.ID,
		SpecID:           apiSpec.SpecID,
		PredictorID:      apiSpec.PredictorID,
		SQSUrl:           queueURL,
		StartTime:        time.Now(),
	}

	err = uploadJobSpec(&jobSpec)
	if err != nil {
		deleteQueueByURL(queueURL)
		return nil, err
	}

	// err = createOperatorLogStreamForJob(jobSpec.JobKey)
	// if err != nil {
	// 	deleteQueueByURL(queueURL)
	// 	return nil, err
	// }

	logger, err := operator.GetJobLoggerFromSpec(apiSpec, jobKey)
	if err != nil {
		return nil, err
	}

	err = setEnqueuingStatus(jobKey)
	if err != nil {
		deleteQueueByURL(queueURL)
		return nil, err
	}

	logger.Info("started enqueuing batches")

	routines.RunWithPanicHandler(func() {
		deployJob(apiSpec, &jobSpec, submission)
	}, false)

	return &jobSpec, nil
}

func uploadJobSpec(jobSpec *spec.Job) error {
	err := config.AWS.UploadJSONToS3(jobSpec, config.Cluster.Bucket, jobSpec.SpecFilePath(config.Cluster.ClusterName))
	if err != nil {
		return err
	}
	return nil
}

func deployJob(apiSpec *spec.API, jobSpec *spec.Job, submission *schema.JobSubmission) {
	logger, err := operator.GetJobLoggerFromSpec(apiSpec, jobSpec.JobKey)
	if err != nil {
		telemetry.Error(err)
		operator.Logger.Error(err)
		return
	}

	totalBatches, err := enqueue(jobSpec, submission)
	if err != nil {
		logger.Error(errors.Wrap(err, "failed to enqueue all batches").Error())

		err := errors.FirstError(
			setEnqueueFailedStatus(jobSpec.JobKey),
			deleteJobRuntimeResources(jobSpec.JobKey),
		)
		if err != nil {
			telemetry.Error(err)
			operator.Logger.Error(err)
		}
		return
	}

	if totalBatches == 0 {
		var errs []error
		logger.Error(ErrorNoDataFoundInJobSubmission().Error())
		if submission.DelimitedFiles != nil {
			logger.Error("please verify that the files are not empty (the files being read can be retrieved by providing `dryRun=true` query param with your job submission")
		}
		errs = append(errs, setEnqueueFailedStatus(jobSpec.JobKey))
		errs = append(errs, deleteJobRuntimeResources(jobSpec.JobKey))

		err := errors.FirstError(errs...)
		if err != nil {
			telemetry.Error(err)
			operator.Logger.Error(err)
		}
		return
	}

	logger.Infof("completed enqueuing a total of %d batches", totalBatches)
	logger.Infof("spinning up workers...")

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

	err = setRunningStatus(jobSpec.JobKey)
	if err != nil {
		handleJobSubmissionError(jobSpec.JobKey, err)
		return
	}
}

func handleJobSubmissionError(jobKey spec.JobKey, jobErr error) {
	logger, err := operator.GetJobLogger(jobKey)
	if err != nil {
		telemetry.Error(err)
		operator.Logger.Error(err)
		return
	}

	logger.Error(jobErr.Error())
	err = errors.FirstError(
		setUnexpectedErrorStatus(jobKey),
		deleteJobRuntimeResources(jobKey),
	)
	if err != nil {
		telemetry.Error(err)
		operator.Logger.Error(err)
	}
}

func createK8sJob(apiSpec *spec.API, jobSpec *spec.Job) error {
	job, err := k8sJobSpec(apiSpec, jobSpec)
	if err != nil {
		return err
	}

	_, err = config.K8s.CreateJob(job)
	if err != nil {
		return err
	}

	return nil
}

func deleteK8sJob(jobKey spec.JobKey) error {
	_, err := config.K8s.DeleteJobs(&kmeta.ListOptions{
		LabelSelector: klabels.SelectorFromSet(map[string]string{"apiName": jobKey.APIName, "jobID": jobKey.ID}).String(),
	})
	if err != nil {
		return err
	}

	return nil
}

func deleteJobRuntimeResources(jobKey spec.JobKey) error {
	err := errors.FirstError(
		deleteK8sJob(jobKey),
		deleteQueueByJobKeyIfExists(jobKey),
	)

	if err != nil {
		return err
	}

	return nil
}

func StopJob(jobKey spec.JobKey) error {
	jobState, err := getJobState(jobKey)
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
		return errors.Wrap(ErrorJobIsNotInProgress(), jobKey.UserString())
	}

	logger, err := operator.GetJobLogger(jobKey)
	if err == nil {
		logger.Warn("request received to stop job; performing cleanup...")
	}

	return errors.FirstError(
		deleteJobRuntimeResources(jobKey),
		setStoppedStatus(jobKey),
	)
}
