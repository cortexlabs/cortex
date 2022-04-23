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

package dequeuer

import (
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/sqs"
	awslib "github.com/cortexlabs/cortex/pkg/lib/aws"
	"github.com/cortexlabs/cortex/pkg/lib/errors"
	s "github.com/cortexlabs/cortex/pkg/lib/strings"
)

type QueueAttributes struct {
	VisibleMessages   int
	InvisibleMessages int
	HasRedrivePolicy  bool
}

func (attr QueueAttributes) TotalMessages() int {
	return attr.VisibleMessages + attr.InvisibleMessages
}

func GetQueueAttributes(client *awslib.Client, queueURL string) (QueueAttributes, error) {
	result, err := client.SQS().GetQueueAttributes(
		&sqs.GetQueueAttributesInput{
			QueueUrl:       aws.String(queueURL),
			AttributeNames: aws.StringSlice([]string{"All"}),
		},
	)
	if err != nil {
		return QueueAttributes{}, errors.WithStack(err)
	}

	attributes := aws.StringValueMap(result.Attributes)

	var visibleCount int
	var notVisibleCount int
	var hasRedrivePolicy bool
	if val, found := attributes["ApproximateNumberOfMessages"]; found {
		count, ok := s.ParseInt(val)
		if ok {
			visibleCount = count
		}
	}

	if val, found := attributes["ApproximateNumberOfMessagesNotVisible"]; found {
		count, ok := s.ParseInt(val)
		if ok {
			notVisibleCount = count
		}
	}

	_, hasRedrivePolicy = attributes["RedrivePolicy"]

	return QueueAttributes{
		VisibleMessages:   visibleCount,
		InvisibleMessages: notVisibleCount,
		HasRedrivePolicy:  hasRedrivePolicy,
	}, nil
}
