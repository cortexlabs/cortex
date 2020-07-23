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
	"io"
	"math"
	"sync"
	"time"

	"github.com/aws/aws-sdk-go/service/s3"
	awslib "github.com/cortexlabs/cortex/pkg/lib/aws"
	"github.com/cortexlabs/cortex/pkg/lib/errors"
	"github.com/cortexlabs/cortex/pkg/lib/telemetry"
	"github.com/cortexlabs/cortex/pkg/operator/config"
	"github.com/cortexlabs/cortex/pkg/operator/operator"
	"github.com/cortexlabs/cortex/pkg/types/spec"
	"github.com/cortexlabs/cortex/pkg/types/status"
	"github.com/cortexlabs/cortex/pkg/types/userconfig"
	kmeta "k8s.io/apimachinery/pkg/apis/meta/v1"
	klabels "k8s.io/apimachinery/pkg/labels"
)

const (
	_lastUpdatedFile = "last_updated"
	_fileBuffer      = 32 * 1024 * 1024
)

var jobIDMutex = sync.Mutex{}

// Job id creation optimized for listing the most recently created jobs in S3. S3 objects are listed in ascending UTF-8 binary order. This should work until the year 2262.
func monotonicallyDecreasingJobID() string {
	jobIDMutex.Lock()
	defer jobIDMutex.Unlock()

	i := math.MaxInt64 - time.Now().UnixNano()
	return fmt.Sprintf("%x", i)
}

func DryRun(submission *userconfig.JobSubmission, response io.Writer) error {
	err := submission.Validate()
	if err != nil {
		return err
	}

	if submission.DelimitedFiles != nil {
		err := validateS3ListerDryRun(&submission.DelimitedFiles.S3Lister, response)
		if err != nil {
			return errors.Wrap(err, userconfig.DelimitedFilesKey)
		}
	}

	if submission.FilePathLister != nil {
		err := validateS3ListerDryRun(&submission.FilePathLister.S3Lister, response)
		if err != nil {
			return errors.Wrap(err, userconfig.FilePathListerKey)
		}
	}

	return nil
}

func validateS3ListerDryRun(s3Lister *awslib.S3Lister, response io.Writer) error {
	filesFound := 0
	for _, s3Path := range s3Lister.S3Paths {
		if !awslib.IsValidS3Path(s3Path) {
			return awslib.ErrorInvalidS3Path(s3Path)
		}

		err := config.AWS.S3IteratorFromLister(*s3Lister, func(bucket string, s3Obj *s3.Object) (bool, error) {
			filesFound++
			filePath := awslib.S3Path(bucket, *s3Obj.Key)
			_, err := io.WriteString(response, fmt.Sprintf("(dryrun) found: %s\n", filePath))
			if err != nil {
				return false, err
			}
			return true, nil
		})

		if err != nil {
			return errors.Wrap(err, s3Path)
		}
	}

	if filesFound == 0 {
		return ErrorNoS3FilesFound()
	}

	return nil
}

func validateS3Lister(s3Lister *awslib.S3Lister) error {
	filesFound := 0
	for _, s3Path := range s3Lister.S3Paths {
		if !awslib.IsValidS3Path(s3Path) {
			return awslib.ErrorInvalidS3Path(s3Path)
		}

		err := config.AWS.S3IteratorFromLister(*s3Lister, func(objPath string, s3Obj *s3.Object) (bool, error) {
			filesFound++
			return false, nil
		})

		if err != nil {
			return errors.Wrap(err, s3Path)
		}
	}

	if filesFound == 0 {
		return ErrorNoS3FilesFound()
	}

	return nil
}

func validateSubmission(submission *userconfig.JobSubmission) error {
	err := submission.Validate()
	if err != nil {
		return err
	}

	if submission.DelimitedFiles != nil {
		err := validateS3Lister(&submission.DelimitedFiles.S3Lister)
		if err != nil {
			if errors.GetKind(err) == ErrNoS3FilesFound {
				return errors.Wrap(errors.Append(err, "; you can append `dryRun=true` query param to experiment with s3_file_list criteria without initializing jobs"), userconfig.DelimitedFilesKey)
			}
			return errors.Wrap(err, userconfig.DelimitedFilesKey)
		}
	}

	if submission.FilePathLister != nil {
		err := validateS3Lister(&submission.FilePathLister.S3Lister)
		if err != nil {
			if errors.GetKind(err) == ErrNoS3FilesFound {
				return errors.Wrap(errors.Append(err, "; you can append `dryRun=true` query param to experiment with s3_file_list criteria without initializing jobs"), userconfig.FilePathListerKey)
			}
			return errors.Wrap(err, userconfig.FilePathListerKey)
		}
	}

	return nil
}

func SubmitJob(apiName string, submission *userconfig.JobSubmission) (*spec.Job, error) {
	err := validateSubmission(submission)
	if err != nil {
		return nil, err
	}

	virtualService, err := getVirtualService(apiName)
	if err != nil {
		return nil, err
	}

	apiID := virtualService.GetLabels()["apiID"]

	apiSpec, err := operator.DownloadAPISpec(apiName, apiID)
	if err != nil {
		return nil, err
	}

	jobID := monotonicallyDecreasingJobID()

	tags := map[string]string{
		"apiName": apiSpec.Name,
		"apiID":   apiSpec.ID,
		"jobID":   jobID,
	}

	jobKey := spec.JobKey{
		APIName: apiSpec.Name,
		ID:      jobID,
	}

	queueURL, err := createFIFOQueue(jobKey, tags)
	if err != nil {
		return nil, err
	}

	jobSpec := spec.Job{
		Job:        submission.Job,
		ResultsDir: fmt.Sprintf("s3://%s/job_results/%s/%s", config.Cluster.Bucket, apiName, jobID),
		JobKey:     jobKey,
		APIID:      apiSpec.ID,
		SQSUrl:     queueURL,
		Created:    time.Now(),
	}

	err = uploadJobSpec(&jobSpec)
	if err != nil {
		deleteQueueByURL(queueURL)
		return nil, err
	}

	err = setEnqueuingStatus(jobSpec.JobKey)
	if err != nil {
		deleteQueueByURL(queueURL)
		return nil, err
	}

	go deployJob(apiSpec, &jobSpec, submission)

	return &jobSpec, nil
}

func downloadJobSpec(jobKey spec.JobKey) (*spec.Job, error) {
	jobSpec := spec.Job{}
	err := config.AWS.ReadJSONFromS3(&jobSpec, config.Cluster.Bucket, jobKey.FileSpecKey())
	if err != nil {
		return nil, ErrorJobNotFound(jobKey)
	}
	return &jobSpec, nil
}

func uploadJobSpec(jobSpec *spec.Job) error {
	err := config.AWS.UploadJSONToS3(jobSpec, config.Cluster.Bucket, jobSpec.FileSpecKey())
	if err != nil {
		return err
	}
	return nil
}

func deployJob(apiSpec *spec.API, jobSpec *spec.Job, submission *userconfig.JobSubmission) {
	err := createLogGroupForJob(jobSpec.JobKey)
	if err != nil {
		handleJobSubmissionError(jobSpec.JobKey, err)
		return
	}

	writeToJobLogGroup(jobSpec.JobKey, "started enqueuing batches to queue")

	totalBatches, err := enqueue(jobSpec, submission)
	if err != nil {
		writeToJobLogGroup(jobSpec.JobKey, errors.Wrap(err, "failed to enqueue all batches").Error())
		setEnqueueFailedStatus(jobSpec.JobKey)
		deleteJobRuntimeResources(jobSpec.JobKey)
		return
	}

	if totalBatches == 0 {
		writeToJobLogGroup(jobSpec.JobKey, ErrorNoDataFoundInJobSubmission().Error())
		if submission.DelimitedFiles != nil {
			writeToJobLogGroup(jobSpec.JobKey, "please verify that the files are not empty (the files being read can be retrieved by providing `dryRun=true` query param with your job submission")
		}
		setEnqueueFailedStatus(jobSpec.JobKey)
		deleteJobRuntimeResources(jobSpec.JobKey)
		return
	}

	writeToJobLogGroup(jobSpec.JobKey, fmt.Sprintf("completed enqueuing a total %d batches", totalBatches), "spinning up workers...")

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

	err = applyK8sJob(apiSpec, jobSpec)
	if err != nil {
		handleJobSubmissionError(jobSpec.JobKey, err)
	}
}

func handleJobSubmissionError(jobKey spec.JobKey, jobErr error) {
	err := writeToJobLogGroup(jobKey, jobErr.Error())
	if err != nil {
		telemetry.Error(err)
		errors.PrintError(err)
	}
	setUnexpectedErrorStatus(jobKey)
	deleteJobRuntimeResources(jobKey)
}

func applyK8sJob(apiSpec *spec.API, jobSpec *spec.Job) error {
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
	err := deleteK8sJob(jobKey)
	if err != nil {
		return err
	}

	err = deleteQueueByJobKey(jobKey)
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

	if jobState.Status == status.JobStopped {
		go deleteJobRuntimeResources(jobKey)
		return errors.Wrap(ErrorJobIsNotInProgress(), jobKey.UserString())
	}

	if !jobState.Status.IsInProgressPhase() {
		go deleteJobRuntimeResources(jobKey)
		return errors.Wrap(ErrorJobIsNotInProgress())
	}

	writeToJobLogGroup(jobKey, "request received to stop job; performing cleanup...")
	return errors.FirstError(
		deleteJobRuntimeResources(jobKey),
		setStoppedStatus(jobKey),
	)
}
