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
	"bytes"
	"net/http"
	"time"

	"github.com/DataDog/datadog-go/statsd"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/sqs"
	awslib "github.com/cortexlabs/cortex/pkg/lib/aws"
	"github.com/cortexlabs/cortex/pkg/lib/errors"
	"github.com/cortexlabs/cortex/pkg/lib/urls"
	"github.com/xtgo/uuid"
	"go.uber.org/zap"
)

const (
	// CortexJobIDHeader is the header containing the job id for the user container
	CortexJobIDHeader        = "X-Cortex-Job-ID"
	_jobCompleteMessageDelay = 10 * time.Second
)

type BatchMessageHandler struct {
	config                  BatchMessageHandlerConfig
	jobCompleteMessageDelay time.Duration
	tags                    []string
	aws                     *awslib.Client
	metrics                 statsd.ClientInterface
	log                     *zap.SugaredLogger
	httpClient              *http.Client
}

type BatchMessageHandlerConfig struct {
	APIName   string
	JobID     string
	QueueURL  string
	Region    string
	TargetURL string
}

func NewBatchMessageHandler(config BatchMessageHandlerConfig, awsClient *awslib.Client, statsdClient statsd.ClientInterface, log *zap.SugaredLogger) *BatchMessageHandler {
	tags := []string{
		"api_name:" + config.APIName,
		"job_id:" + config.JobID,
	}

	return &BatchMessageHandler{
		config:                  config,
		jobCompleteMessageDelay: _jobCompleteMessageDelay,
		tags:                    tags,
		aws:                     awsClient,
		metrics:                 statsdClient,
		log:                     log,
		httpClient:              &http.Client{},
	}
}

func (h *BatchMessageHandler) Handle(message *sqs.Message) error {
	if isOnJobCompleteMessage(message) {
		err := h.onJobComplete(message)
		if err != nil {
			return errors.Wrap(err, "failed to handle 'onJobComplete' message")
		}
		return nil
	}
	err := h.handleBatch(message)
	if err != nil {
		return err
	}
	return nil
}

func (h *BatchMessageHandler) recordSuccess() error {
	err := h.metrics.Incr("cortex_batch_succeeded", h.tags, 1.0)
	if err != nil {
		return errors.WithStack(err)
	}
	return nil
}

func (h *BatchMessageHandler) recordFailure() error {
	err := h.metrics.Incr("cortex_batch_failed", h.tags, 1.0)
	if err != nil {
		return errors.WithStack(err)
	}
	return nil
}

func (h *BatchMessageHandler) recordTimePerBatch(elapsedTime time.Duration) error {
	err := h.metrics.Histogram("cortex_time_per_batch", elapsedTime.Seconds(), h.tags, 1.0)
	if err != nil {
		return errors.WithStack(err)
	}
	return nil
}

func (h *BatchMessageHandler) submitRequest(messageBody string, isOnJobComplete bool) error {
	targetURL := h.config.TargetURL
	if isOnJobComplete {
		targetURL = urls.Join(targetURL, "/on-job-complete")
	}

	req, err := http.NewRequest(http.MethodPost, targetURL, bytes.NewBuffer([]byte(messageBody)))
	if err != nil {
		return errors.WithStack(err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set(CortexJobIDHeader, h.config.JobID)
	response, err := h.httpClient.Do(req)
	if err != nil {
		return ErrorUserContainerNotReachable(err)
	}
	defer func() {
		_ = response.Body.Close()
	}()

	if response.StatusCode == http.StatusNotFound && isOnJobComplete {
		return nil
	}

	if response.StatusCode != http.StatusOK {
		return ErrorUserContainerResponseStatusCode(response.StatusCode)
	}

	return nil
}

func (h *BatchMessageHandler) handleBatch(message *sqs.Message) error {
	h.log.Infow("processing batch", "id", *message.MessageId)

	startTime := time.Now()

	err := h.submitRequest(*message.Body, false)
	if err != nil {
		h.log.Errorw("failed to process batch", "id", *message.MessageId, "error", err)
		recordFailureErr := h.recordFailure()
		if recordFailureErr != nil {
			return errors.Wrap(recordFailureErr, "failed to record failure metric")
		}
		return nil
	}

	endTime := time.Since(startTime)

	err = h.recordSuccess()
	if err != nil {
		return errors.Wrap(err, "failed to record success metric")
	}

	err = h.recordTimePerBatch(endTime)
	if err != nil {
		return errors.Wrap(err, "failed to record time per batch")
	}
	return nil
}

func (h *BatchMessageHandler) onJobComplete(message *sqs.Message) error {
	shouldRunOnJobComplete := false
	h.log.Info("received job_complete message")
	for {
		queueAttributes, err := GetQueueAttributes(h.aws, h.config.QueueURL)
		if err != nil {
			return err
		}

		totalMessages := queueAttributes.TotalMessages()

		if totalMessages > 1 {
			time.Sleep(h.jobCompleteMessageDelay)
			h.log.Infow("found other messages in queue, requeuing job_complete message", "id", *message.MessageId)
			newMessageID := uuid.NewRandom().String()
			if _, err = h.aws.SQS().SendMessage(
				&sqs.SendMessageInput{
					QueueUrl:    &h.config.QueueURL,
					MessageBody: aws.String("job_complete"),
					MessageAttributes: map[string]*sqs.MessageAttributeValue{
						"job_complete": {
							DataType:    aws.String("String"),
							StringValue: aws.String("true"),
						},
						"api_name": {
							DataType:    aws.String("String"),
							StringValue: aws.String(h.config.APIName),
						},
						"job_id": {
							DataType:    aws.String("String"),
							StringValue: aws.String(h.config.JobID),
						},
					},
					MessageDeduplicationId: aws.String(newMessageID),
					MessageGroupId:         aws.String(newMessageID),
				},
			); err != nil {
				return errors.WithStack(err)
			}

			return nil
		}

		if shouldRunOnJobComplete {
			h.log.Infow("processing job_complete message", "id", *message.MessageId)
			return h.submitRequest(*message.Body, true)
		}
		shouldRunOnJobComplete = true

		time.Sleep(h.jobCompleteMessageDelay)
	}
}

func isOnJobCompleteMessage(message *sqs.Message) bool {
	_, found := message.MessageAttributes["job_complete"]
	return found
}
