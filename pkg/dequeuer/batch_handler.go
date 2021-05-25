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
	"bytes"
	"net/http"
	"time"

	"github.com/DataDog/datadog-go/statsd"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/sqs"
	awslib "github.com/cortexlabs/cortex/pkg/lib/aws"
	"github.com/cortexlabs/cortex/pkg/lib/errors"
	"github.com/xtgo/uuid"
	"go.uber.org/zap"
)

const _jobCompleteMessageRenewal = 10 * time.Second

type BatchMessageHandler struct {
	config                    BatchMessageHandlerConfig
	jobCompleteMessageRenewal time.Duration
	tags                      []string
	aws                       *awslib.Client
	metrics                   statsd.ClientInterface
	log                       *zap.SugaredLogger
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
		"job_id" + config.JobID,
	}

	return &BatchMessageHandler{
		config:                    config,
		jobCompleteMessageRenewal: _jobCompleteMessageRenewal,
		tags:                      tags,
		aws:                       awsClient,
		metrics:                   statsdClient,
		log:                       log,
	}
}

func (h *BatchMessageHandler) Handle(message *sqs.Message) error {
	if isOnJobCompleteMessage(message) {
		err := h.onJobComplete(message)
		if err != nil {
			// TODO
			h.log.Fatal(err)
		}
		return nil
	}
	err := h.handleBatch(message)
	if err != nil {
		// TODO
		h.log.Errorw("failed processing batch", "id", *message.MessageId, "error", err)
		err = h.handleFailure(message)
		if err != nil {
			// TODO
			h.log.Fatal(err)
		}
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

func (h *BatchMessageHandler) submitRequest(messageBody string) error {
	response, err := http.Post(h.config.TargetURL, "application/json", bytes.NewBuffer([]byte(messageBody)))
	if err != nil {
		return err
	}

	if response.StatusCode != http.StatusOK {
		return ErrorUserContainerResponseStatusCode(response.StatusCode)
	}

	return nil
}

func (h *BatchMessageHandler) handleBatch(message *sqs.Message) error {
	h.log.Info("processing batch", "id", *message.MessageId)

	startTime := time.Now()
	err := h.submitRequest(*message.Body)
	if err != nil {
		return err
	}
	endTime := time.Now().Sub(startTime)

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

func (h *BatchMessageHandler) handleFailure(_ *sqs.Message) error {
	err := h.recordFailure()
	if err != nil {
		return errors.Wrap(err, "failed to record failure metric")
	}

	return nil
}

func (h *BatchMessageHandler) onJobComplete(message *sqs.Message) error {
	shouldRunOnJobComplete := false
	h.log.Info("received job_complete message")
	for true {
		queueAttributes, err := GetQueueAttributes(h.aws, h.config.QueueURL)
		if err != nil {
			return err
		}

		totalMessages := queueAttributes.TotalMessages()

		if totalMessages > 1 {
			time.Sleep(h.jobCompleteMessageRenewal)
			h.log.Info("found other messages in queue, requeuing job_complete message")
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
				return err
			}

			return nil
		}

		if shouldRunOnJobComplete {
			h.log.Infow("processing job_complete message")
			return h.submitRequest(*message.Body)
		}
		shouldRunOnJobComplete = true

		time.Sleep(h.jobCompleteMessageRenewal)
	}

	return nil
}

func isOnJobCompleteMessage(message *sqs.Message) bool {
	_, found := message.MessageAttributes["job_complete"]
	return found
}
