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

package asyncapi

import (
	"context"
	"fmt"
	"strconv"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/sqs"
	"github.com/cortexlabs/cortex/pkg/lib/errors"
	"github.com/cortexlabs/cortex/pkg/operator/config"
	"github.com/cortexlabs/cortex/pkg/types/userconfig"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/prometheus/common/model"
)

const (
	_sqsQueryTimeoutSeconds        = 10
	_prometheusQueryTimeoutSeconds = 10
)

var queueLengthGauge = promauto.NewGaugeVec(
	prometheus.GaugeOpts{
		Name:        "cortex_async_queue_length",
		Help:        "The number of in-queue messages for a cortex AsyncAPI",
		ConstLabels: map[string]string{"api_kind": userconfig.AsyncAPIKind.String()},
	}, []string{"api_name"},
)

func updateQueueLengthMetricsFn(apiName, queueURL string) func() error {
	return func() error {
		sqsClient := config.AWS.SQS()

		ctx, cancel := context.WithTimeout(context.Background(), _sqsQueryTimeoutSeconds*time.Second)
		defer cancel()

		input := &sqs.GetQueueAttributesInput{
			AttributeNames: []*string{
				aws.String("ApproximateNumberOfMessages"),
				aws.String("ApproximateNumberOfMessagesNotVisible"),
			},
			QueueUrl: aws.String(queueURL),
		}

		output, err := sqsClient.GetQueueAttributesWithContext(ctx, input)
		if err != nil {
			return err
		}

		visibleMessagesStr := output.Attributes["ApproximateNumberOfMessages"]
		invisibleMessagesStr := output.Attributes["ApproximateNumberOfMessagesNotVisible"]

		visibleMessages, err := strconv.ParseFloat(*visibleMessagesStr, 64)
		if err != nil {
			return err
		}

		invisibleMessages, err := strconv.ParseFloat(*invisibleMessagesStr, 64)
		if err != nil {
			return err
		}

		queueLength := visibleMessages + invisibleMessages
		queueLengthGauge.WithLabelValues(apiName).Set(queueLength)

		return nil
	}
}

func getMessagesInQueue(apiName string, window time.Duration) (*float64, error) {
	windowSeconds := int64(window.Seconds())

	// PromQL query:
	// 	sum(sum_over_time(cortex_async_queue_length{api_name="<apiName>"}[60s])) /
	//	sum(count_over_time(cortex_async_queue_length{api_name="<apiName>"}[60s]))
	query := fmt.Sprintf(
		"sum(sum_over_time(cortex_async_queue_length{api_name=\"%s\"}[%ds])) / "+
			"max(count_over_time(cortex_async_queue_length{api_name=\"%s\"}[%ds]))",
		apiName, windowSeconds,
		apiName, windowSeconds,
	)

	ctx, cancel := context.WithTimeout(context.Background(), _prometheusQueryTimeoutSeconds*time.Second)
	defer cancel()

	valuesQuery, err := config.Prometheus.Query(ctx, query, time.Now())
	if err != nil {
		return nil, err
	}

	values, ok := valuesQuery.(model.Vector)
	if !ok {
		return nil, errors.ErrorUnexpected("failed to convert prometheus metric to vector")
	}

	// no values available
	if values.Len() == 0 {
		return nil, nil
	}

	avgMessagesInQueue := float64(values[0].Value)

	return &avgMessagesInQueue, nil
}
