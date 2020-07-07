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
	"path"
	"sync"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/sqs"
	"github.com/cortexlabs/cortex/pkg/lib/cron"
	"github.com/cortexlabs/cortex/pkg/lib/errors"
	"github.com/cortexlabs/cortex/pkg/lib/k8s"
	"github.com/cortexlabs/cortex/pkg/lib/telemetry"
	"github.com/cortexlabs/cortex/pkg/operator/config"
	"github.com/cortexlabs/cortex/pkg/operator/operator"
	"github.com/cortexlabs/cortex/pkg/types/spec"
	"github.com/cortexlabs/cortex/pkg/types/userconfig"
	kmeta "k8s.io/apimachinery/pkg/apis/meta/v1"
	klabels "k8s.io/apimachinery/pkg/labels"
)

const (
	_lastUpdatedFile = "last_updated"
)

var jobIDMutex = sync.Mutex{}

// Job id creation optimized for listing the most recently created jobs in S3. S3 objects are listed in ascending UTF-8 binary order. This should work until the year 2262.
func monotonicallyDecreasingJobID() string {
	jobIDMutex.Lock()
	defer jobIDMutex.Unlock()

	i := math.MaxInt64 - time.Now().UnixNano()
	return fmt.Sprintf("%x", i)
}

func SubmitJob(apiName string, submission userconfig.JobSubmission) (*spec.Job, error) {
	err := submission.Validate()
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

	queueURL, err := operator.CreateQueue(apiName, jobID, tags)
	if err != nil {
		return nil, err
	}

	jobSpec := spec.Job{
		Job: submission.Job,
		JobKey: spec.JobKey{
			APIName: apiSpec.Name,
			ID:      jobID,
		},
		APIID:   apiSpec.ID,
		SQSUrl:  queueURL,
		Created: time.Now(),
	}

	err = uploadJobSpec(&jobSpec)
	if err != nil {
		deleteQueue(queueURL)
		return nil, err
	}

	err = setEnqueuingStatus(jobSpec.JobKey)
	if err != nil {
		deleteQueue(queueURL)
		return nil, err
	}

	go deployJob(apiSpec, &jobSpec, &submission)

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
	err := operator.CreateLogGroupForJob(jobSpec.JobKey)
	if err != nil {
		handleJobSubmissionError(jobSpec.JobKey, err)
		return
	}

	operator.WriteToJobLogGroup(jobSpec.JobKey, "started enqueuing batches to queue")

	totalBatches, err := enqueue(jobSpec, submission)
	if err != nil {
		operator.WriteToJobLogGroup(jobSpec.JobKey, errors.Wrap(err, "failed to enqueue all batches").Error())
		setEnqueueFailedStatus(jobSpec.JobKey)
		deleteJobRuntimeResources(jobSpec.JobKey)
		return
	}

	operator.WriteToJobLogGroup(jobSpec.JobKey, fmt.Sprintf("completed enqueuing a total %d batches", totalBatches), "spinning up workers...")

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
	err := operator.WriteToJobLogGroup(jobKey, jobErr.Error())
	if err != nil {
		fmt.Println(err.Error())
	}
	setUnexpectedErrorStatus(jobKey)
	deleteJobRuntimeResources(jobKey)
}

func cronErrHandler(cronName string) func(error) {
	return func(err error) {
		err = errors.Wrap(err, cronName+" cron failed")
		telemetry.Error(err)
		errors.PrintError(err)
	}
}

func updateLiveness(jobKey spec.JobKey) error {
	s3Key := path.Join(jobKey.PrefixKey(), _lastUpdatedFile)
	err := config.AWS.UploadJSONToS3(time.Now(), config.Cluster.Bucket, s3Key)
	if err != nil {
		return errors.Wrap(err, "failed to update liveness", jobKey.UserString())
	}
	return nil
}

func enqueue(jobSpec *spec.Job, submission *userconfig.JobSubmission) (int, error) {
	err := updateLiveness(jobSpec.JobKey)
	if err != nil {
		return 0, err
	}

	livenessUpdater := func() error {
		return updateLiveness(jobSpec.JobKey)
	}

	livenessCron := cron.Run(livenessUpdater, cronErrHandler(fmt.Sprintf("liveness check for %s", jobSpec.UserString())), 20*time.Second)
	defer livenessCron.Cancel()

	total := 0
	for i, batch := range submission.Batches {
		randomID := k8s.RandomName()

		for retry := 0; retry < 3; retry++ {
			_, err := config.AWS.SQS().SendMessage(&sqs.SendMessageInput{
				MessageDeduplicationId: aws.String(randomID),
				QueueUrl:               aws.String(jobSpec.SQSUrl),
				MessageBody:            aws.String(string(batch)),
				MessageGroupId:         aws.String(randomID),
			})
			if err != nil {
				newErr := errors.Wrap(errors.WithStack(err), fmt.Sprintf("batch %d", i))
				if retry == 2 {
					return 0, errors.Wrap(newErr, fmt.Sprintf("failed after retrying 3 times to enqueue batch %d", i))
				}
				operator.WriteToJobLogGroup(jobSpec.JobKey, newErr.Error())
			} else {
				break
			}
		}

		total++
		if total%100 == 0 {
			operator.WriteToJobLogGroup(jobSpec.JobKey, fmt.Sprintf("enqueuing %d batches...", total))
		}
	}

	return total, nil
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

	queueURL, err := operator.QueueURL(jobKey)
	if err != nil {
		return err
	}

	err = deleteQueue(queueURL)
	if err != nil {
		return err
	}

	return nil
}

func StopJob(jobKey spec.JobKey) error {
	operator.WriteToJobLogGroup(jobKey, "request received to stop job; performing cleanup...")
	return errors.FirstError(
		setStoppedStatus(jobKey),
		deleteJobRuntimeResources(jobKey),
	)
}
