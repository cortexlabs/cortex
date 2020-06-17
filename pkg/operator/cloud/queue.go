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

package cloud

import (
	"fmt"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/sqs"
	"github.com/cortexlabs/cortex/pkg/lib/debug"
	s "github.com/cortexlabs/cortex/pkg/lib/strings"
	"github.com/cortexlabs/cortex/pkg/operator/config"
)

func QueuePrefix(apiName string) (string, error) {
	operatorAccountID, _, err := config.AWS.GetCachedAccountID()
	if err != nil {
		return "", err
	}

	return fmt.Sprintf("https://sqs.%s.amazonaws.com/%s/cortex-%s", config.AWS.Region, operatorAccountID, apiName), nil
}

func QueueURL(apiName, jobID string) (string, error) {
	queueURL, err := QueuePrefix(apiName)
	if err != nil {
		return "", nil
	}

	return queueURL + "-" + jobID + ".fifo", nil
}

type QueueMetrics struct {
	InQueue    int `json:"in_queue"`
	NotVisible int `json:"not_visible"`
}

func GetQueueMetrics(apiName, jobID string) (*QueueMetrics, error) {
	queueURL, err := QueueURL(apiName, jobID)
	if err != nil {
		return nil, err // TODO
	}

	fmt.Println(queueURL)

	output, err := config.AWS.SQS().GetQueueAttributes(&sqs.GetQueueAttributesInput{
		AttributeNames: aws.StringSlice([]string{"ApproximateNumberOfMessages", "ApproximateNumberOfMessagesNotVisible"}),
		QueueUrl:       aws.String(queueURL),
	})
	if err != nil {
		return nil, err // TODO
	}

	debug.Pp(output)

	metrics := QueueMetrics{}
	parsedInt, ok := s.ParseInt(*output.Attributes["ApproximateNumberOfMessages"])
	if ok {
		metrics.InQueue = parsedInt
	}

	parsedInt, ok = s.ParseInt(*output.Attributes["ApproximateNumberOfMessagesNotVisible"])
	if ok {
		metrics.NotVisible = parsedInt
	}

	return &metrics, nil
}
