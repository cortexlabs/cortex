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

package aws

import (
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/sqs"
	"github.com/cortexlabs/cortex/pkg/lib/errors"
)

func (c *Client) ListQueuesByQueueNamePrefix(queueNamePrefix string) ([]string, error) {
	output, err := c.SQS().ListQueues(&sqs.ListQueuesInput{
		QueueNamePrefix: aws.String(queueNamePrefix),
	})
	if err != nil {
		return nil, errors.WithStack(err)
	}

	return aws.StringValueSlice(output.QueueUrls), nil
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

func (c *Client) DeleteQueues(queueNamePrefix string) error {
	output, err := c.SQS().ListQueues(
		&sqs.ListQueuesInput{
			QueueNamePrefix: aws.String(queueNamePrefix),
		},
	)
	if err != nil {
		return errors.WithStack(err)
	}

	errs := []error{}

	for _, queueURL := range output.QueueUrls {
		_, err := c.SQS().DeleteQueue(&sqs.DeleteQueueInput{ // best effort delete
			QueueUrl: queueURL,
		})
		if err != nil {
			errs = append(errs, err)
		}
	}

	return errors.FirstError(errs...)
}
