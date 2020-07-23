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
	"fmt"
	"strings"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/sqs"
	"github.com/cortexlabs/cortex/pkg/lib/errors"
	s "github.com/cortexlabs/cortex/pkg/lib/strings"
	"github.com/cortexlabs/cortex/pkg/operator/config"
	"github.com/cortexlabs/cortex/pkg/types/metrics"
	"github.com/cortexlabs/cortex/pkg/types/spec"
)

func getJobQueueName(jobKey spec.JobKey) string {
	return config.Cluster.SQSNamePrefix() + "-" + jobKey.APIName + "-" + jobKey.ID + ".fifo"
}

func getJobQueueURL(jobKey spec.JobKey) (string, error) {
	operatorAccountID, _, err := config.AWS.GetCachedAccountID()
	if err != nil {
		return "", errors.Wrap(err, "failed to construct queue url", "unable to get account id")
	}

	return fmt.Sprintf("https://sqs.%s.amazonaws.com/%s/%s", config.AWS.Region, operatorAccountID, getJobQueueName(jobKey)), nil
}

func jobKeyFromQueueURL(queueURL string) spec.JobKey {
	split := strings.Split(queueURL, "/")
	queueName := split[len(split)-1]

	dashSplit := strings.Split(queueName, "-")

	jobID := strings.TrimSuffix(dashSplit[len(dashSplit)-1], ".fifo")

	apiNamesplit := dashSplit[1 : len(dashSplit)-1]
	apiName := strings.Join(apiNamesplit, "-")

	return spec.JobKey{APIName: apiName, ID: jobID}
}

func createFIFOQueue(jobKey spec.JobKey, tags map[string]string) (string, error) {
	for key, value := range config.Cluster.Tags {
		tags[key] = value
	}

	queueName := getJobQueueName(jobKey)

	output, err := config.AWS.SQS().CreateQueue(
		&sqs.CreateQueueInput{
			Attributes: map[string]*string{
				"FifoQueue":         aws.String("true"),
				"VisibilityTimeout": aws.String("90"),
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

func doesQueueExist(jobKey spec.JobKey) (bool, error) {
	return config.AWS.DoesQueueExist(getJobQueueName(jobKey))
}

func listQueueURLsForAPI(apiName string) ([]string, error) {
	queuesForAPIPrefix := config.Cluster.SQSNamePrefix() + "-" + apiName + "-"

	queueURLs, err := config.AWS.ListQueuesByQueueNamePrefix(queuesForAPIPrefix)
	if err != nil {
		return nil, err
	}

	return queueURLs, nil
}

func listQueuesForAllAPIs() ([]string, error) {
	queueURLs, err := config.AWS.ListQueuesByQueueNamePrefix(config.Cluster.SQSNamePrefix() + "-")
	if err != nil {
		return nil, err
	}

	return queueURLs, nil
}

func deleteQueueByJobKey(jobKey spec.JobKey) error {
	queueURL, err := getJobQueueURL(jobKey)
	if err != nil {
		return err
	}

	return deleteQueueByURL(queueURL)
}

func deleteQueueByURL(queueURL string) error {
	_, err := config.AWS.SQS().DeleteQueue(&sqs.DeleteQueueInput{
		QueueUrl: aws.String(queueURL),
	})

	return err
}

func getQueueMetrics(jobKey spec.JobKey) (*metrics.QueueMetrics, error) {
	queueURL, err := getJobQueueURL(jobKey)
	if err != nil {
		return nil, err
	}
	return getQueueMetricsFromURL(queueURL)
}

func getQueueMetricsFromURL(queueURL string) (*metrics.QueueMetrics, error) {
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
