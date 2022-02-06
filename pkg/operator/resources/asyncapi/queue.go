/*
Copyright 2022 Cortex Labs, Inc.

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
	"github.com/cortexlabs/cortex/pkg/config"
	"github.com/cortexlabs/cortex/pkg/lib/errors"
	s "github.com/cortexlabs/cortex/pkg/lib/strings"
	"github.com/cortexlabs/cortex/pkg/types/clusterconfig"
)

func createFIFOQueue(apiName string, initialDeploymentTime int64, tags map[string]string) (string, error) {
	for key, value := range config.ClusterConfig.Tags {
		tags[key] = value
	}

	queueName := apiQueueName(apiName, initialDeploymentTime)

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

func apiQueueName(apiName string, initialDeploymentTime int64) string {
	// initialDeploymentTime is incorporated so that the queue name changes when doing a deploy after a delete
	// (if the queue name doesn't change, the user would have to wait 60 seconds before recreating the queue)
	initialDeploymentTimeStr := s.Int64(initialDeploymentTime)
	initialDeploymentTimeID := initialDeploymentTimeStr[len(initialDeploymentTimeStr)-10:]
	return config.ClusterConfig.SQSNamePrefix() + apiName + clusterconfig.SQSQueueDelimiter + initialDeploymentTimeID + ".fifo"
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

func getQueueURL(apiName string, initialDeploymentTime int64) (string, error) {
	operatorAccountID, _, err := config.AWS.GetCachedAccountID()
	if err != nil {
		return "", errors.Wrap(err, "failed to construct queue url", "unable to get account id")
	}

	return fmt.Sprintf(
		"https://sqs.%s.amazonaws.com/%s/%s",
		config.AWS.Region, operatorAccountID, apiQueueName(apiName, initialDeploymentTime),
	), nil
}
