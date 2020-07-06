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

package operator

import (
	"fmt"
	"strings"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/sqs"
	awslib "github.com/cortexlabs/cortex/pkg/lib/aws"
	"github.com/cortexlabs/cortex/pkg/lib/errors"
	"github.com/cortexlabs/cortex/pkg/lib/hash"
	s "github.com/cortexlabs/cortex/pkg/lib/strings"
	"github.com/cortexlabs/cortex/pkg/operator/config"
	"github.com/cortexlabs/cortex/pkg/types/metrics"
	"github.com/cortexlabs/cortex/pkg/types/spec"
)

func ClusterBasedQueueName() string {
	return hash.String(config.Cluster.ClusterName)[:11] + "-"
}

func QueuePrefix() (string, error) {
	operatorAccountID, _, err := config.AWS.GetCachedAccountID()
	if err != nil {
		return "", errors.Wrap(err, "failed to construct queue url", "unable to get account id")
	}

	return fmt.Sprintf("https://sqs.%s.amazonaws.com/%s/%s", config.AWS.Region, operatorAccountID, ClusterBasedQueueName()), nil
}

func QueueName(jobKey spec.JobKey) string {
	queuePrefix := ClusterBasedQueueName()

	return queuePrefix + jobKey.APIName + "-" + jobKey.ID + ".fifo"
}

func QueueURL(jobKey spec.JobKey) (string, error) {
	prefix, err := QueuePrefix()
	if err != nil {
		return "", err
	}
	return prefix + jobKey.APIName + "-" + jobKey.ID + ".fifo", nil
}

func QueueNameFromURL(queueURL string) string {
	split := strings.Split(queueURL, "/")
	queueName := split[len(split)-1]
	return queueName
}

func IdentifierFromQueueURL(queueURL string) spec.JobKey {
	queueName := QueueNameFromURL(queueURL)
	apiNameJobIDConcat := queueName[len(ClusterBasedQueueName()) : len(queueName)-len(".fifo")]
	underscoreSplit := strings.Split(apiNameJobIDConcat, "-")

	jobID := underscoreSplit[len(underscoreSplit)-1]

	apiNamesplit := underscoreSplit[:len(underscoreSplit)-1]
	apiName := strings.Join(apiNamesplit, "-")

	return spec.JobKey{APIName: apiName, ID: jobID}
}

func CreateQueue(apiName string, jobID string, tags map[string]string) (string, error) {
	for key, value := range config.Cluster.Tags {
		tags[key] = value
	}

	queueName := ClusterBasedQueueName() + apiName + "-" + jobID + ".fifo"

	output, err := config.AWS.SQS().CreateQueue(
		&sqs.CreateQueueInput{
			Attributes: map[string]*string{
				"FifoQueue":                 aws.String("true"),
				"ContentBasedDeduplication": aws.String("true"),
				"VisibilityTimeout":         aws.String("90"),
			},
			QueueName: aws.String(queueName),
			Tags:      aws.StringMap(tags),
		},
	)
	if err != nil {
		return "", errors.Wrap(err, "failed to create sqs queue", queueName)
	}

	return *output.QueueUrl, nil
}

func ListQueues() ([]string, error) {
	queuePrefix := ClusterBasedQueueName()

	output, err := config.AWS.SQS().ListQueues(
		&sqs.ListQueuesInput{
			QueueNamePrefix: aws.String(queuePrefix),
		},
	)
	if err != nil {
		return nil, errors.WithStack(err)
	}

	return aws.StringValueSlice(output.QueueUrls), nil
}

func GetQueueMetrics(jobKey spec.JobKey) (*metrics.QueueMetrics, error) {
	queueURL, err := QueueURL(jobKey)
	if err != nil {
		return nil, err
	}
	return GetQueueMetricsFromURL(queueURL)
}

func DoesQueueExist(queueURL string) (bool, error) {
	operatorAccountID, _, err := config.AWS.GetCachedAccountID()
	if err != nil {
		return false, err
	}

	queueName := QueueNameFromURL(queueURL)

	_, err = config.AWS.SQS().GetQueueUrl(
		&sqs.GetQueueUrlInput{
			QueueName:              aws.String(queueName),
			QueueOwnerAWSAccountId: aws.String(operatorAccountID),
		},
	)
	if err != nil {
		if awslib.IsErrCode(err, sqs.ErrCodeQueueDoesNotExist) {
			return false, nil
		}
		return false, errors.Wrap(err, "failed to check if queue exists", queueName)
	}

	return true, nil
}

func GetQueueMetricsFromURL(queueURL string) (*metrics.QueueMetrics, error) {
	output, err := config.AWS.SQS().GetQueueAttributes(&sqs.GetQueueAttributesInput{
		AttributeNames: aws.StringSlice([]string{"ApproximateNumberOfMessages", "ApproximateNumberOfMessagesNotVisible"}),
		QueueUrl:       aws.String(queueURL),
	})
	if err != nil {
		return nil, errors.Wrap(err, "failed to get queue metrics", queueURL)
	}

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
