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
	"strings"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/sqs"
	"github.com/cortexlabs/cortex/pkg/lib/debug"
	s "github.com/cortexlabs/cortex/pkg/lib/strings"
	"github.com/cortexlabs/cortex/pkg/operator/config"
	"github.com/cortexlabs/cortex/pkg/types/metrics"
)

func QueueNamePrefix() string {
	return "cortex-"
}

func QueuePrefix() (string, error) {
	operatorAccountID, _, err := config.AWS.GetCachedAccountID()
	if err != nil {
		return "", err // TODO
	}

	return fmt.Sprintf("https://sqs.%s.amazonaws.com/%s/%s", config.AWS.Region, operatorAccountID, QueueNamePrefix()), nil
}

func QueueName(apiName, jobID string) string {
	queuePrefix := QueueNamePrefix()

	return queuePrefix + apiName + "-" + jobID + ".fifo"
}

func QueueURL(apiName, jobID string) (string, error) {
	prefix, err := QueuePrefix()
	if err != nil {
		return "", err // TODO
	}
	return prefix + apiName + "-" + jobID + ".fifo", nil
}

func IdentifiersFromQueueURL(queueURL string) (string, string) {
	split := strings.Split(queueURL, "/")
	queueName := split[len(split)-1]
	apiAndJob := queueName[len("cortex-") : len(queueName)-len(".fifo")]
	apiAndJobSplit := strings.Split(apiAndJob, "-")
	jobID := apiAndJobSplit[len(apiAndJobSplit)-1]
	apiNamesplit := apiAndJobSplit[:len(apiAndJobSplit)-1]
	apiName := strings.Join(apiNamesplit, "-")

	return apiName, jobID
}

func ListQueues() ([]string, error) {
	queuePrefix := QueueNamePrefix()

	debug.Pp(queuePrefix)

	output, err := config.AWS.SQS().ListQueues(
		&sqs.ListQueuesInput{
			QueueNamePrefix: aws.String(queuePrefix),
		},
	)
	if err != nil {
		return nil, err
	}

	return aws.StringValueSlice(output.QueueUrls), nil
}

func GetQueueMetrics(apiName, jobID string) (*metrics.QueueMetrics, error) {
	queueURL, err := QueueURL(apiName, jobID)
	if err != nil {
		return nil, err
	}
	return GetQueueMetricsFromURL(queueURL)
}

func GetQueueMetricsFromURL(queueURL string) (*metrics.QueueMetrics, error) {
	output, err := config.AWS.SQS().GetQueueAttributes(&sqs.GetQueueAttributesInput{
		AttributeNames: aws.StringSlice([]string{"ApproximateNumberOfMessages", "ApproximateNumberOfMessagesNotVisible"}),
		QueueUrl:       aws.String(queueURL),
	})
	if err != nil {
		return nil, err // TODO
	}

	debug.Pp(output)

	metrics := metrics.QueueMetrics{}
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
