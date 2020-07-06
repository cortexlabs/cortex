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
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/sqs"
	"github.com/cortexlabs/cortex/pkg/lib/cron"
	"github.com/cortexlabs/cortex/pkg/lib/debug"
	"github.com/cortexlabs/cortex/pkg/lib/errors"
	"github.com/cortexlabs/cortex/pkg/lib/k8s"
	"github.com/cortexlabs/cortex/pkg/lib/telemetry"
	"github.com/cortexlabs/cortex/pkg/operator/config"
	"github.com/cortexlabs/cortex/pkg/operator/operator"
	"github.com/cortexlabs/cortex/pkg/types/spec"
	"github.com/cortexlabs/cortex/pkg/types/userconfig"
)

func APIQueuePrefix(apiName string) string {
	return operator.ClusterBasedQueueName() + "-" + apiName
}

func QueuesPerAPI(apiName string) ([]string, error) {
	response, err := config.AWS.SQS().ListQueues(&sqs.ListQueuesInput{
		QueueNamePrefix: aws.String(APIQueuePrefix(apiName)),
	})

	if err != nil {
		return nil, err
	}

	debug.Pp(response)

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

func Enqueue(jobSpec *spec.Job, submission *userconfig.JobSubmission) error {
	err := UpdateLiveness(jobSpec.JobKey)
	if err != nil {
		return err
	}

	livenessUpdater := func() error {
		return UpdateLiveness(jobSpec.JobKey)
	}

	livenessCron := cron.Run(livenessUpdater, cronErrHandler(fmt.Sprintf("liveness check for %s (api %s)", jobSpec.ID, jobSpec.APIName)), 20*time.Second)
	defer livenessCron.Cancel()

	total := 0
	startTime := time.Now()
	for i, batch := range submission.Batches {
		for k := 0; k < 2000; k++ {
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
						return errors.Wrap(newErr, fmt.Sprintf("failed after retrying 3 times to enqueue batch %d", i))
					}
					operator.WriteToJobLogGroup(jobSpec.JobKey, newErr.Error())
				}
			}

			total++
			if total%100 == 0 {
				operator.WriteToJobLogGroup(jobSpec.JobKey, fmt.Sprintf("enqueued %d batches", total))
			}
		}
	}

	debug.Pp(time.Now().Sub(startTime).Milliseconds())

	jobSpec.TotalBatchCount = total
	err = SetRunningStatus(jobSpec)
	if err != nil {
		return err
	}

	return nil
}

func DeleteQueue(queueURL string) error {
	errors.PrintStacktrace(errors.ErrorUnexpected("unexpected"))
	_, err := config.AWS.SQS().DeleteQueue(&sqs.DeleteQueueInput{
		QueueUrl: aws.String(queueURL),
	})

	return err
}
