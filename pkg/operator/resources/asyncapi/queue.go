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

package asyncapi

import (
	"fmt"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/sqs"
	"github.com/cortexlabs/cortex/pkg/lib/errors"
	"github.com/cortexlabs/cortex/pkg/operator/config"
)

func createFIFOQueue(apiName string, deploymentID string, tags map[string]string) (string, error) {
	if config.CoreConfig.IsManaged {
		managedConfig := config.ManagedConfigOrNil()
		if managedConfig != nil {
			for key, value := range managedConfig.Tags {
				tags[key] = value
			}
		}
	}

	queueName := apiQueueName(apiName, deploymentID)

	attributes := map[string]string{
		sqs.QueueAttributeNameFifoQueue:         "true",
		sqs.QueueAttributeNameVisibilityTimeout: "60",
	}

	output, err := config.AWS.SQS().CreateQueue(
		&sqs.CreateQueueInput{
			Attributes: aws.StringMap(attributes),
			QueueName:  aws.String(queueName),
			Tags:       aws.StringMap(tags),
		},
	)
	if err != nil {
		return "", errors.Wrap(err, "failed to create sqs queue", queueName)
	}

	return *output.QueueUrl, nil
}

func apiQueueName(apiName string, deploymentID string) string {
	return config.CoreConfig.SQSNamePrefix() + apiName + "-" + deploymentID + ".fifo"
}

func deleteQueueByURL(queueURL string) error {
	_, err := config.AWS.SQS().DeleteQueue(&sqs.DeleteQueueInput{
		QueueUrl: aws.String(queueURL),
	})
	if err != nil {
		return errors.Wrap(err, "failed to delete queue", queueURL)
	}

	return err
}

func getQueueURL(apiName string, deploymentID string) (string, error) {
	operatorAccountID, _, err := config.AWS.GetCachedAccountID()
	if err != nil {
		return "", errors.Wrap(err, "failed to construct queue url", "unable to get account id")
	}

	return fmt.Sprintf(
		"https://sqs.%s.amazonaws.com/%s/%s",
		config.AWS.Region, operatorAccountID, apiQueueName(apiName, deploymentID),
	), nil
}
