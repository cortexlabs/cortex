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

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/sqs"
	"github.com/cortexlabs/cortex/pkg/lib/cron"
	"github.com/cortexlabs/cortex/pkg/lib/errors"
	"github.com/cortexlabs/cortex/pkg/lib/telemetry"
	"github.com/cortexlabs/cortex/pkg/operator/config"
	"github.com/cortexlabs/cortex/pkg/operator/operator"
	"github.com/cortexlabs/cortex/pkg/types/spec"
	"github.com/cortexlabs/cortex/pkg/types/userconfig"
	kmeta "k8s.io/apimachinery/pkg/apis/meta/v1"
	klabels "k8s.io/apimachinery/pkg/labels"
)

var jobIDMutex = sync.Mutex{}

// Job id creation optimized for listing the most recently created jobs in S3. S3 objects are listed in ascending UTF-8 binary order. This should work until the year 2262.
func MonotonicallyDecreasingJobID() string {
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

	jobID := MonotonicallyDecreasingJobID()

	tags := map[string]string{
		"apiName": apiSpec.Name,
		"apiID":   apiSpec.ID,
		"jobID":   jobID,
	}

	for key, value := range config.Cluster.Tags {
		tags[key] = value
	}

	output, err := config.AWS.SQS().CreateQueue(
		&sqs.CreateQueueInput{
			Attributes: map[string]*string{
				"FifoQueue":                 aws.String("true"),
				"ContentBasedDeduplication": aws.String("true"),
				"VisibilityTimeout":         aws.String("90"),
			},
			QueueName: aws.String("cortex" + "-" + apiName + "-" + jobID + ".fifo"),
			Tags:      aws.StringMap(tags),
		},
	)
	if err != nil {
		return nil, errors.Wrap(err) // TODO
	}

	jobSpec := spec.Job{
		JobSpec: submission.JobSpec,
		JobID: spec.JobID{
			APIName: apiSpec.Name,
			ID:      jobID,
		},
		APIID:   apiSpec.ID,
		SQSUrl:  *output.QueueUrl,
		Created: time.Now(),
	}

	err = SetEnqueuingStatus(&jobSpec)
	if err != nil {
		DeleteQueue(*output.QueueUrl)
		return nil, err
	}

	go DeployJob(&jobSpec, &submission)

	return &jobSpec, nil
}

func UploadJobSpec(jobSpec *spec.Job) error {
	err := config.AWS.UploadJSONToS3(jobSpec, config.Cluster.Bucket, spec.JobSpecKey(jobSpec.JobID))
	if err != nil {
		return err // TODO
	}

	return nil
}

func cronErrHandler(cronName string) func(error) { // TODO what happens if liveness fails? Delete API?
	return func(err error) {
		err = errors.Wrap(err, cronName+" cron failed")
		telemetry.Error(err)
		errors.PrintError(err)
	}
}

func DeployJob(jobSpec *spec.Job, submission *userconfig.JobSubmission) {
	livenessUpdater := func() error {
		return UpdateLiveness(jobSpec.JobID)
	}

	livenessCron := cron.Run(livenessUpdater, cronErrHandler(fmt.Sprintf("failed liveness check", jobSpec.ID, jobSpec.APIName)), 10*time.Second)
	defer livenessCron.Cancel()

	err := operator.CreateLogGroupForJob(jobSpec.JobID)
	if err != nil {
		fmt.Println(err.Error()) // TODO
		SetErroredStatus(jobSpec.JobID)
		DeleteJob(jobSpec.JobID)
	}

	err = operator.WriteToJobLogGroup(jobSpec.JobID, "started enqueueing partitions")
	if err != nil {
		fmt.Println(err.Error()) //  TODO
	}

	err = Enqueue(jobSpec, submission)
	if err != nil {
		fmt.Println("failed to enqueue")
		operator.WriteToJobLogGroup(jobSpec.JobID, err.Error())
		SetErroredStatus(jobSpec.JobID)
		DeleteJob(jobSpec.JobID)
	}

	err = operator.WriteToJobLogGroup(jobSpec.JobID, "completed enqueueing partitions")
	if err != nil {
		fmt.Println(err.Error()) //  TODO
	}

	err = SetRunningStatus(jobSpec)
	if err != nil {
		// TODO here
	}

	err = ApplyK8sJob(jobSpec)
	if err != nil {
		fmt.Println("DeployJob")
		operator.WriteToJobLogGroup(jobSpec.JobID, err.Error())
		SetErroredStatus(jobSpec.JobID)
		DeleteJob(jobSpec.JobID)
	}
}

func ApplyK8sJob(jobSpec *spec.Job) error {
	apiSpec, err := operator.DownloadAPISpec(jobSpec.APIName, jobSpec.APIID)
	if err != nil {
		return err // TODO
	}
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

func DeleteJob(jobID spec.JobID) error {
	_, err := config.K8s.DeleteJobs(&kmeta.ListOptions{
		LabelSelector: klabels.SelectorFromSet(map[string]string{"apiName": jobID.APIName, "jobID": jobID.ID}).String(),
	})
	if err != nil {
		return err
	}

	queueURL, err := operator.QueueURL(jobID)
	if err != nil {
		return err
	}

	fmt.Println(queueURL)
	_, err = config.AWS.SQS().DeleteQueue(&sqs.DeleteQueueInput{
		QueueUrl: aws.String(queueURL),
	})
	if err != nil {
		return err
	}

	return nil
}

func StopJob(jobID spec.JobID) error {
	return errors.FirstError(
		SetStoppedStatus(jobID),
		DeleteJob(jobID),
	)
	return nil
}
