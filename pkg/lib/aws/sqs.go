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

package aws

import (
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/sqs"
	"github.com/cortexlabs/cortex/pkg/lib/errors"
)

func (c *Client) GetAllQueueAttributes(queueURL string) (map[string]string, error) {
	output, err := c.SQS().GetQueueAttributes(&sqs.GetQueueAttributesInput{
		QueueUrl:       aws.String(queueURL),
		AttributeNames: aws.StringSlice([]string{"All"}),
	})
	if err != nil {
		return nil, errors.Wrap(err, "unable to get queue attributes", queueURL)
	}

	return aws.StringValueMap(output.Attributes), nil
}

func (c *Client) ListQueuesByQueueNamePrefix(queueNamePrefix string) ([]string, error) {
	var queueURLs []string

	err := c.SQS().ListQueuesPages(&sqs.ListQueuesInput{
		QueueNamePrefix: aws.String(queueNamePrefix),
	}, func(output *sqs.ListQueuesOutput, lastPage bool) bool {
		queueURLs = append(queueURLs, aws.StringValueSlice(output.QueueUrls)...)
		return true
	})
	if err != nil {
		return nil, errors.WithStack(err)
	}

	return queueURLs, nil
}

func (c *Client) DoesQueueExist(queueName string) (bool, error) {
	_, err := c.SQS().GetQueueUrl(&sqs.GetQueueUrlInput{
		QueueName: aws.String(queueName),
	})
	if err != nil {
		if IsErrCode(err, sqs.ErrCodeQueueDoesNotExist) {
			return false, nil
		}
		return false, errors.Wrap(err, "failed to check if queue exists", queueName)
	}

	return true, nil
}

func (c *Client) DeleteQueuesWithPrefix(queueNamePrefix string) error {
	var deleteError error

	err := c.SQS().ListQueuesPages(&sqs.ListQueuesInput{
		QueueNamePrefix: aws.String(queueNamePrefix),
	}, func(output *sqs.ListQueuesOutput, lastPage bool) bool {
		for _, queueURL := range output.QueueUrls {
			_, err := c.SQS().DeleteQueue(&sqs.DeleteQueueInput{ // best effort delete
				QueueUrl: queueURL,
			})

			if deleteError != nil {
				deleteError = err
			}
		}
		return true
	})

	if err != nil {
		return errors.WithStack(err)
	}

	if deleteError != nil {
		return errors.WithStack(deleteError)
	}

	return nil
}
