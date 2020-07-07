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
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/sqs"
	"github.com/cortexlabs/cortex/pkg/operator/config"
	"github.com/cortexlabs/cortex/pkg/operator/operator"
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

func deleteQueue(queueURL string) error {
	_, err := config.AWS.SQS().DeleteQueue(&sqs.DeleteQueueInput{
		QueueUrl: aws.String(queueURL),
	})

	return err
}
