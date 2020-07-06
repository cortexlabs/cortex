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
	"github.com/cortexlabs/cortex/pkg/operator/config"
	"github.com/cortexlabs/cortex/pkg/operator/operator"
	"github.com/cortexlabs/cortex/pkg/types/spec"
	"github.com/cortexlabs/cortex/pkg/types/userconfig"
	kmeta "k8s.io/apimachinery/pkg/apis/meta/v1"
	klabels "k8s.io/apimachinery/pkg/labels"
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
		handleJobSubmissionError(jobSpec.JobKey, err)
		return
	}

	operator.WriteToJobLogGroup(jobSpec.JobKey, "completed enqueuing batches to queue")

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
	setErroredStatus(jobKey)
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
