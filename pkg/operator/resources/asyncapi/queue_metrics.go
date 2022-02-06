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

package asyncapi

import (
	"context"
	"strconv"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/sqs"
	"github.com/cortexlabs/cortex/pkg/config"
	"github.com/cortexlabs/cortex/pkg/lib/errors"
	"github.com/cortexlabs/cortex/pkg/types/userconfig"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

const (
	_sqsQueryTimeoutSeconds = 10
)

var activeGauge = promauto.NewGaugeVec(
	prometheus.GaugeOpts{
		Name:        "cortex_async_active",
		Help:        "The number of messages that are actively being processed by an AsyncAPI",
		ConstLabels: map[string]string{"api_kind": userconfig.AsyncAPIKind.String()},
	}, []string{"api_name"},
)

var queuedGauge = promauto.NewGaugeVec(
	prometheus.GaugeOpts{
		Name:        "cortex_async_queued",
		Help:        "The number queued messages for an AsyncAPI",
		ConstLabels: map[string]string{"api_kind": userconfig.AsyncAPIKind.String()},
	}, []string{"api_name"},
)

var inFlightGauge = promauto.NewGaugeVec(
	prometheus.GaugeOpts{
		Name:        "cortex_async_in_flight",
		Help:        "The number of in-flight messages for an AsyncAPI (including active and queued)",
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
			return errors.WithStack(err)
		}

		visibleMessagesStr := output.Attributes["ApproximateNumberOfMessages"]
		invisibleMessagesStr := output.Attributes["ApproximateNumberOfMessagesNotVisible"]

		visibleMessages, err := strconv.ParseFloat(*visibleMessagesStr, 64)
		if err != nil {
			return errors.WithStack(err)
		}

		invisibleMessages, err := strconv.ParseFloat(*invisibleMessagesStr, 64)
		if err != nil {
			return errors.WithStack(err)
		}

		activeGauge.WithLabelValues(apiName).Set(invisibleMessages)
		queuedGauge.WithLabelValues(apiName).Set(visibleMessages)
		inFlightGauge.WithLabelValues(apiName).Set(invisibleMessages + visibleMessages)

		return nil
	}
}
