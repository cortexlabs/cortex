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
	metrics                   *statsd.Client
	log                       *zap.SugaredLogger
}

type BatchMessageHandlerConfig struct {
	APIName  string
	JobID    string
	QueueURL string
	Region   string
}

func NewBatchMessageHandler(config BatchMessageHandlerConfig, awsClient *awslib.Client, statsdClient *statsd.Client, log *zap.SugaredLogger) *BatchMessageHandler {
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

func (handler *BatchMessageHandler) Handle(message *sqs.Message) error {
	if isOnJobCompleteMessage(message) {
		err := handler.onJobComplete(message)
		if err != nil {
			// TODO
			handler.log.Fatal(zap.Error(err))
		}
		return nil
	}
	err := handler.handleBatch(message)
	if err != nil {
		// TODO
		handler.log.Infof("failed processing batch %s", *message.MessageId)
		err = handler.handleFailure(message)
		if err != nil {
			// TODO
			handler.log.Fatal(zap.Error(err))
		}
	}
	return nil
}

func (handler *BatchMessageHandler) recordSuccess() error {
	err := handler.metrics.Incr("cortex_batch_succeeded", handler.tags, 1.0)
	if err != nil {
		return errors.WithStack(err)
	}
	return nil
}

func (handler *BatchMessageHandler) recordFailure() error {
	err := handler.metrics.Incr("cortex_batch_failed", handler.tags, 1.0)
	if err != nil {
		return errors.WithStack(err)
	}
	return nil
}

func (handler *BatchMessageHandler) recordTimePerBatch(elapsedTime time.Duration) error {
	err := handler.metrics.Histogram("cortex_time_per_batch", elapsedTime.Seconds(), handler.tags, 1.0)
	if err != nil {
		return errors.WithStack(err)
	}
	return nil
}

func (handler *BatchMessageHandler) submitRequest(messageBody string) error {
	// TODO: add code to make HTTP request to target container
	time.Sleep(1 * time.Minute)
	handler.log.Info(messageBody)
	return nil
}

func (handler *BatchMessageHandler) handleBatch(message *sqs.Message) error {
	handler.log.Infof("processing batch %s", *message.MessageId)

	startTime := time.Now()
	err := handler.submitRequest(*message.Body)
	if err != nil {
		return err
	}

	err = handler.recordSuccess()
	if err != nil {
		return errors.Wrap(err, "failed to record success metric")
	}

	err = handler.recordTimePerBatch(time.Now().Sub(startTime))
	if err != nil {
		return errors.Wrap(err, "failed to record time per batch")
	}
	return nil
}

func (handler *BatchMessageHandler) handleFailure(_ *sqs.Message) error {
	err := handler.recordFailure()
	if err != nil {
		return err
	}

	return nil
}

func (handler *BatchMessageHandler) onJobComplete(message *sqs.Message) error {
	shouldRunOnJobComplete := false
	handler.log.Info("received job_complete message")
	for true {
		queueAttributes, err := GetQueueAttributes(handler.aws, handler.config.QueueURL)
		if err != nil {
			return err
		}

		totalMessages := queueAttributes.TotalMessages()

		if totalMessages > 1 {
			time.Sleep(handler.jobCompleteMessageRenewal)
			handler.log.Info("found other messages in queue, requeuing job_complete message")
			newMessageID := uuid.NewRandom().String()
			if _, err = handler.aws.SQS().SendMessage(
				&sqs.SendMessageInput{
					QueueUrl:    &handler.config.QueueURL,
					MessageBody: aws.String("job_complete"),
					MessageAttributes: map[string]*sqs.MessageAttributeValue{
						"job_complete": {
							DataType:    aws.String("String"),
							StringValue: aws.String("true"),
						},
						"api_name": {
							DataType:    aws.String("String"),
							StringValue: aws.String(handler.config.APIName),
						},
						"job_id": {
							DataType:    aws.String("String"),
							StringValue: aws.String(handler.config.JobID),
						},
					},
					MessageDeduplicationId: aws.String(newMessageID),
					MessageGroupId:         aws.String(newMessageID),
				},
			); err != nil {
				return err
			}

			return nil
		} else {
			if shouldRunOnJobComplete {
				handler.log.Info("processing job_complete message")
				return handler.submitRequest(*message.Body)
			}
			shouldRunOnJobComplete = true
		}

		time.Sleep(handler.jobCompleteMessageRenewal)
	}

	return nil
}

func isOnJobCompleteMessage(message *sqs.Message) bool {
	_, found := message.MessageAttributes["job_complete"]
	return found
}
