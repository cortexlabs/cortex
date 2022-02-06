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

package batchapi

import (
	"fmt"

	"github.com/cortexlabs/cortex/pkg/config"
	"github.com/cortexlabs/cortex/pkg/lib/errors"
	s "github.com/cortexlabs/cortex/pkg/lib/strings"
	"github.com/cortexlabs/cortex/pkg/types/clusterconfig"
	"github.com/cortexlabs/cortex/pkg/types/metrics"
	"github.com/cortexlabs/cortex/pkg/types/spec"
)

func apiQueueNamePrefix(apiName string) string {
	// <sqs_prefix>_b_<api_name>_
	return config.ClusterConfig.SQSNamePrefix() + "b" + clusterconfig.SQSQueueDelimiter +
		apiName + clusterconfig.SQSQueueDelimiter
}

// QueueName is cx_<hash of cluster name>_b_<api_name>_<job_id>.fifo
func getJobQueueName(jobKey spec.JobKey) string {
	return apiQueueNamePrefix(jobKey.APIName) + jobKey.ID + ".fifo"
}

func getJobQueueURL(jobKey spec.JobKey) (string, error) {
	operatorAccountID, _, err := config.AWS.GetCachedAccountID()
	if err != nil {
		return "", errors.Wrap(err, "failed to construct queue url", "unable to get account id")
	}

	return fmt.Sprintf("https://sqs.%s.amazonaws.com/%s/%s", config.AWS.Region, operatorAccountID, getJobQueueName(jobKey)), nil
}

func getQueueMetrics(jobKey spec.JobKey) (*metrics.QueueMetrics, error) {
	queueURL, err := getJobQueueURL(jobKey)
	if err != nil {
		return nil, err
	}
	return getQueueMetricsFromURL(queueURL)
}

func getQueueMetricsFromURL(queueURL string) (*metrics.QueueMetrics, error) {
	attributes, err := config.AWS.GetAllQueueAttributes(queueURL)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get queue metrics")
	}

	qMetrics := metrics.QueueMetrics{}
	parsedInt, ok := s.ParseInt(attributes["ApproximateNumberOfMessages"])
	if ok {
		qMetrics.Visible = parsedInt
	}

	parsedInt, ok = s.ParseInt(attributes["ApproximateNumberOfMessagesNotVisible"])
	if ok {
		qMetrics.NotVisible = parsedInt
	}

	return &qMetrics, nil
}
