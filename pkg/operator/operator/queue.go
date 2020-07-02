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
	"github.com/cortexlabs/cortex/pkg/lib/debug"
	"github.com/cortexlabs/cortex/pkg/lib/errors"
	s "github.com/cortexlabs/cortex/pkg/lib/strings"
	"github.com/cortexlabs/cortex/pkg/operator/config"
	"github.com/cortexlabs/cortex/pkg/types/metrics"
	"github.com/cortexlabs/cortex/pkg/types/spec"
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

func QueueName(jobID spec.JobID) string {
	queuePrefix := QueueNamePrefix()

	return queuePrefix + jobID.APIName + "-" + jobID.ID + ".fifo"
}

func QueueURL(jobID spec.JobID) (string, error) {
	prefix, err := QueuePrefix()
	if err != nil {
		return "", err // TODO
	}
	return prefix + jobID.APIName + "-" + jobID.ID + ".fifo", nil
}

func QueueNameFromURL(queueURL string) string {
	split := strings.Split(queueURL, "/")
	queueName := split[len(split)-1]
	return queueName
}

func IdentifierFromQueueURL(queueURL string) spec.JobID {
	queueName := QueueNameFromURL(queueURL)
	apiNameJobIDConcat := queueName[len("cortex-") : len(queueName)-len(".fifo")]
	underscoreSplit := strings.Split(apiNameJobIDConcat, "-")

	jobID := underscoreSplit[len(underscoreSplit)-1]

	apiNamesplit := underscoreSplit[:len(underscoreSplit)-1]
	apiName := strings.Join(apiNamesplit, "-")

	return spec.JobID{APIName: apiName, ID: jobID}
}

func ListQueues() ([]string, error) {
	queuePrefix := QueueNamePrefix()

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

func GetQueueMetrics(jobID spec.JobID) (*metrics.QueueMetrics, error) {
	queueURL, err := QueueURL(jobID)
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

	_, err = config.AWS.SQS().GetQueueUrl(
		&sqs.GetQueueUrlInput{
			QueueName:              aws.String(QueueNameFromURL(queueURL)),
			QueueOwnerAWSAccountId: aws.String(operatorAccountID),
		},
	)
	if err != nil {
		if awslib.IsErrCode(err, sqs.ErrCodeQueueDoesNotExist) {
			return false, nil
		}
		return false, err
	}

	return true, nil
}

func GetQueueMetricsFromURL(queueURL string) (*metrics.QueueMetrics, error) {
	debug.Pp(queueURL)
	output, err := config.AWS.SQS().GetQueueAttributes(&sqs.GetQueueAttributesInput{
		AttributeNames: aws.StringSlice([]string{"ApproximateNumberOfMessages", "ApproximateNumberOfMessagesNotVisible"}),
		QueueUrl:       aws.String(queueURL),
	})
	if err != nil {
		return nil, errors.WithStack(err) // TODO
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
