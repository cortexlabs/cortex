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

package dequeuer

import (
	"context"
	"net/http"
	"strconv"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/sqs"
	"github.com/cortexlabs/cortex/pkg/lib/cron"
	"github.com/cortexlabs/cortex/pkg/lib/errors"
	"github.com/cortexlabs/cortex/pkg/lib/telemetry"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"go.uber.org/zap"
)

const (
	_sqsQueryTimeout          = 10 * time.Second
	_queueLengthQueryInterval = 10 * time.Second
)

type AsyncStatsReporter struct {
	sqsClient    *sqs.SQS
	queueURL     string
	running      bool
	handler      http.Handler
	latencies    *prometheus.HistogramVec
	requestCount *prometheus.CounterVec
	queueLength  prometheus.Gauge
	logger       *zap.SugaredLogger
}

func NewAsyncPrometheusStatsReporter(sqsClient *sqs.SQS, queueURL string, logger *zap.SugaredLogger) *AsyncStatsReporter {
	latenciesHist := promauto.NewHistogramVec(prometheus.HistogramOpts{
		Name: "cortex_async_latency",
		Help: "Histogram of the latencies for an AsyncAPI kind in seconds",
	}, []string{"status_code"})

	requestCounter := promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "cortex_async_request_count",
		Help: "Request count for an AsyncAPI",
	}, []string{"status_code"})

	queueLengthGauge := promauto.NewGauge(
		prometheus.GaugeOpts{
			Name: "cortex_async_queue_length",
			Help: "The number of in-queue messages for a cortex AsyncAPI",
		},
	)

	handler := promhttp.Handler()

	return &AsyncStatsReporter{
		sqsClient:    sqsClient,
		queueURL:     queueURL,
		handler:      handler,
		latencies:    latenciesHist,
		requestCount: requestCounter,
		queueLength:  queueLengthGauge,
		logger:       logger,
	}
}

func (r *AsyncStatsReporter) Start() func() {
	if r.running {
		return func() {}
	}

	errorHandler := func(err error) {
		err = errors.Wrap(err, "failed to get queue length from sqs queue")
		r.logger.Error(err)
		telemetry.Error(err)
	}

	queueLengthCron := cron.Run(r.getQueueLength, errorHandler, _queueLengthQueryInterval)
	r.running = true

	return func() {
		queueLengthCron.Cancel()
	}
}

func (r *AsyncStatsReporter) HandleEvent(event RequestEvent) {
	labels := map[string]string{
		"status_code": strconv.Itoa(event.StatusCode),
	}

	r.latencies.With(labels).Observe(event.Duration.Seconds())
	r.requestCount.With(labels).Add(1)
}

func (r *AsyncStatsReporter) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	r.handler.ServeHTTP(w, req)
}

func (r *AsyncStatsReporter) getQueueLength() error {
	ctx, cancel := context.WithTimeout(context.Background(), _sqsQueryTimeout)
	defer cancel()

	input := &sqs.GetQueueAttributesInput{
		AttributeNames: []*string{
			aws.String("ApproximateNumberOfMessages"),
			aws.String("ApproximateNumberOfMessagesNotVisible"),
		},
		QueueUrl: aws.String(r.queueURL),
	}

	output, err := r.sqsClient.GetQueueAttributesWithContext(ctx, input)
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
	r.queueLength.Set(queueLength)

	return nil
}
