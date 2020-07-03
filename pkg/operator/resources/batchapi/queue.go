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
	"github.com/cortexlabs/cortex/pkg/lib/debug"
	"github.com/cortexlabs/cortex/pkg/lib/errors"
	"github.com/cortexlabs/cortex/pkg/lib/k8s"
	"github.com/cortexlabs/cortex/pkg/operator/config"
	"github.com/cortexlabs/cortex/pkg/operator/operator"
	"github.com/cortexlabs/cortex/pkg/types/spec"
	"github.com/cortexlabs/cortex/pkg/types/userconfig"
)

func APIQueuePrefix(apiName string) string {
	return operator.QueueNamePrefix() + "-" + apiName
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

func Enqueue(jobSpec *spec.Job, submission *userconfig.JobSubmission) error {
	total := 0
	startTime := time.Now()
	for i, item := range submission.Items {
		// TODO kill the goroutine? This should automatically error when the next enqueue message fails
		// for k := 0; k < 100; k++ {
		randomId := k8s.RandomName()
		_, err := config.AWS.SQS().SendMessage(&sqs.SendMessageInput{
			MessageDeduplicationId: aws.String(randomId),
			QueueUrl:               aws.String(jobSpec.SQSUrl),
			MessageBody:            aws.String(string(item)),
			MessageGroupId:         aws.String(randomId),
		})
		if err != nil {
			return errors.Wrap(errors.WithStack(err), fmt.Sprintf("item %d", i)) // TODO
		}
		total++
	}

	debug.Pp(time.Now().Sub(startTime).Milliseconds())

	jobSpec.TotalPartitions = total
	err := SetRunningStatus(jobSpec)
	if err != nil {
		return err
	}

	return nil
}

func DeleteQueue(queueURL string) error {
	_, err := config.AWS.SQS().DeleteQueue(&sqs.DeleteQueueInput{
		QueueUrl: aws.String(queueURL),
	})

	return err
}
