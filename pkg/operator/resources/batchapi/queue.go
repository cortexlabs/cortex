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
	"path"
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
)

const (
	_lastUpdatedFile = "last_updated"
)

func apiQueuePrefix(apiName string) string {
	return operator.ClusterBasedQueueName() + "-" + apiName
}

func listQueuesPerAPI(apiName string) ([]string, error) {
	response, err := config.AWS.SQS().ListQueues(&sqs.ListQueuesInput{
		QueueNamePrefix: aws.String(apiQueuePrefix(apiName)),
	})

	if err != nil {
		return nil, err
	}

	queueNames := make([]string, len(response.QueueUrls))
	for i := range queueNames {
		queueNames[i] = *response.QueueUrls[i]
	}

	return queueNames, nil
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
			}
		}

		total++
		if total%100 == 0 {
			operator.WriteToJobLogGroup(jobSpec.JobKey, fmt.Sprintf("enqueued %d batches", total))
		}
	}

	return total, nil
}

func deleteQueue(queueURL string) error {
	_, err := config.AWS.SQS().DeleteQueue(&sqs.DeleteQueueInput{
		QueueUrl: aws.String(queueURL),
	})

	return err
}
