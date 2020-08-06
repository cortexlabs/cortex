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
	"math"
	"sync"
	"time"

	"github.com/cortexlabs/cortex/pkg/lib/errors"
	"github.com/cortexlabs/cortex/pkg/lib/telemetry"
	"github.com/cortexlabs/cortex/pkg/operator/config"
	"github.com/cortexlabs/cortex/pkg/operator/operator"
	"github.com/cortexlabs/cortex/pkg/operator/schema"
	"github.com/cortexlabs/cortex/pkg/types/spec"
	kmeta "k8s.io/apimachinery/pkg/apis/meta/v1"
	klabels "k8s.io/apimachinery/pkg/labels"
)

var jobIDMutex = sync.Mutex{}

// Job ID creation optimized for listing the most recently created jobs in S3. S3 objects are listed in ascending UTF-8 binary order. This should work until the year 2262.
func monotonicallyDecreasingJobID() string {
	jobIDMutex.Lock()
	defer jobIDMutex.Unlock()

	i := math.MaxInt64 - time.Now().UnixNano()
	return fmt.Sprintf("%x", i)
}

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

	apiID := virtualService.GetLabels()["apiID"]

	apiSpec, err := operator.DownloadAPISpec(apiName, apiID)
	if err != nil {
		return nil, err
	}

	jobID := monotonicallyDecreasingJobID()

	jobKey := spec.JobKey{
		APIName: apiSpec.Name,
		ID:      jobID,
	}

	tags := map[string]string{
		"apiName": apiSpec.Name,
		"apiID":   apiSpec.ID,
		"jobID":   jobID,
	}

	queueURL, err := createFIFOQueue(jobKey, tags)
	if err != nil {
		return nil, err
	}

	jobSpec := spec.Job{
		RuntimeJobConfig: submission.RuntimeJobConfig,
		JobKey:           jobKey,
		APIID:            apiSpec.ID,
		SQSUrl:           queueURL,
		StartTime:        time.Now(),
	}

	err = uploadJobSpec(&jobSpec)
	if err != nil {
		deleteQueueByURL(queueURL)
		return nil, err
	}

	err = createOperatorLogStreamForJob(jobSpec.JobKey)
	if err != nil {
		deleteQueueByURL(queueURL)
		return nil, err
	}

	err = setEnqueuingStatus(jobKey)
	if err != nil {
		deleteQueueByURL(queueURL)
		return nil, err
	}

	err = writeToJobLogStream(jobSpec.JobKey, "started enqueuing batches")
	if err != nil {
		deleteQueueByURL(queueURL)
		return nil, err
	}

	go deployJob(apiSpec, &jobSpec, submission)

	return &jobSpec, nil
}

func downloadJobSpec(jobKey spec.JobKey) (*spec.Job, error) {
	jobSpec := spec.Job{}
	err := config.AWS.ReadJSONFromS3(&jobSpec, config.Cluster.Bucket, jobKey.SpecFilePath())
	if err != nil {
		return nil, errors.Wrap(err, "unable to download job specification", jobKey.UserString())
	}
	return &jobSpec, nil
}

func uploadJobSpec(jobSpec *spec.Job) error {
	err := config.AWS.UploadJSONToS3(jobSpec, config.Cluster.Bucket, jobSpec.SpecFilePath())
	if err != nil {
		return err
	}
	return nil
}

func deployJob(apiSpec *spec.API, jobSpec *spec.Job, submission *schema.JobSubmission) {
	totalBatches, err := enqueue(jobSpec, submission)
	if err != nil {
		err := errors.FirstError(
			writeToJobLogStream(jobSpec.JobKey, errors.Wrap(err, "failed to enqueue all batches").Error()),
			setEnqueueFailedStatus(jobSpec.JobKey),
			deleteJobRuntimeResources(jobSpec.JobKey),
		)
		if err != nil {
			telemetry.Error(err)
			errors.PrintError(err)
		}
		return
	}

	if totalBatches == 0 {
		var errs []error
		writeToJobLogStream(jobSpec.JobKey, ErrorNoDataFoundInJobSubmission().Error())
		if submission.DelimitedFiles != nil {
			errs = append(errs, writeToJobLogStream(jobSpec.JobKey, "please verify that the files are not empty (the files being read can be retrieved by providing `dryRun=true` query param with your job submission"))
		}
		errs = append(errs, setEnqueueFailedStatus(jobSpec.JobKey))
		errs = append(errs, deleteJobRuntimeResources(jobSpec.JobKey))

		err := errors.FirstError(errs...)
		if err != nil {
			telemetry.Error(err)
			errors.PrintError(err)
		}
		return
	}

	writeToJobLogStream(jobSpec.JobKey, fmt.Sprintf("completed enqueuing a total of %d batches", totalBatches), "spinning up workers...")

	jobSpec.TotalBatchCount = totalBatches

	err = uploadJobSpec(jobSpec)
	if err != nil {
		handleJobSubmissionError(jobSpec.JobKey, err)
		return
	}

	err = setRunningStatus(jobSpec.JobKey)
	if err != nil {
		handleJobSubmissionError(jobSpec.JobKey, err)
		return
	}

	err = createK8sJob(apiSpec, jobSpec)
	if err != nil {
		handleJobSubmissionError(jobSpec.JobKey, err)
	}
}

func handleJobSubmissionError(jobKey spec.JobKey, jobErr error) {
	err := errors.FirstError(
		writeToJobLogStream(jobKey, jobErr.Error()),
		setUnexpectedErrorStatus(jobKey),
		deleteJobRuntimeResources(jobKey),
	)
	if err != nil {
		telemetry.Error(err)
		errors.PrintError(err)
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
		deleteQueueByJobKey(jobKey),
	)

	if err != nil {
		return err
	}

	return nil
}

func StopJob(jobKey spec.JobKey) error {
	jobState, err := getJobState(jobKey)
	if err != nil {
		go deleteJobRuntimeResources(jobKey)
		return err
	}

	if !jobState.Status.IsInProgress() {
		go deleteJobRuntimeResources(jobKey)
		return errors.Wrap(ErrorJobIsNotInProgress(), jobKey.UserString())
	}

	writeToJobLogStream(jobKey, "request received to stop job; performing cleanup...")
	return errors.FirstError(
		deleteJobRuntimeResources(jobKey),
		setStoppedStatus(jobKey),
	)
}
